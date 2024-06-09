package peer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/peers"
	"github.com/chezzijr/p2p/internal/common/torrent"
)

const (
	MaxBlockSize = 16 * 1024
	MaxBacklog   = 5
)

var (
	ErrIntegrity = errors.New("Integrity check failed")
)

// for peer to download from other peers
type DownloadSession struct {
	*torrent.TorrentFile
	peerInfo *Peer
	fd       *os.File
	filepath string
	peers    []peers.Peer
	bitfield connection.BitField
	done     bool
}

type pieceInfo struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

type pieceDownloadSession struct {
	index          int
	assignedClient *DownloadClient
	buf            []byte
	downloaded     int
	requested      int
	backlog        int
}

func (s *pieceDownloadSession) readMessage() error {
	msg, err := s.assignedClient.Read() // this call blocks
	if err != nil {
		return err
	}

	if msg == nil { // keep-alive
		return nil
	}

	switch msg.ID {
	case connection.MsgUnchoke:
		s.assignedClient.Choked = false
	case connection.MsgChoke:
		s.assignedClient.Choked = true
	case connection.MsgHave:
		index, err := connection.ParseHaveMsg(msg)
		if err != nil {
			return err
		}
		s.assignedClient.Bitfield.SetPiece(index)
	case connection.MsgPiece:
		n, err := connection.ParsePieceMsg(s.index, s.buf, msg)
		if err != nil {
			return err
		}
		s.downloaded += n
		s.backlog--
	}
	return nil
}

func checkIntegrity(buf []byte, piece *pieceInfo) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], piece.hash[:]) {
		return ErrIntegrity
	}
	return nil
}

func attemptDownloadPiece(c *DownloadClient, pi *pieceInfo) ([]byte, error) {
	session := pieceDownloadSession{
		index:          pi.index,
		assignedClient: c,
		buf:            make([]byte, pi.length),
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // Disable the deadline

	for session.downloaded < pi.length {
		// If unchoked, send requests until we have enough unfulfilled requests
		if !session.assignedClient.Choked {
			for session.backlog < MaxBacklog && session.requested < pi.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if pi.length-session.requested < blockSize {
					blockSize = pi.length - session.requested
				}

				err := c.SendRequest(uint32(pi.index), uint32(session.requested), uint32(blockSize))
				if err != nil {
					return nil, err
				}

				session.backlog++
				session.requested += blockSize
			}
		}

		err := session.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return session.buf, nil
}

func (ds *DownloadSession) downloadFromPeer(ctx context.Context, peer peers.Peer, pQ chan *pieceInfo, rQ chan *pieceResult) {
	c, err := NewClient(ctx, peer, ds.peerInfo.PeerID, ds.InfoHash)
	if err != nil {
		return
	}

	defer func() {
		c.SendNotInterested()
		c.Close()
	}()
	c.SendInterested()

	for pi := range pQ {
		if !c.Bitfield.HasPiece(pi.index) {
			pQ <- pi
			continue
		}

		// download piece
		buf, err := attemptDownloadPiece(c, pi)
		if err != nil {
			pQ <- pi
			return
		}

		if err := checkIntegrity(buf, pi); err != nil {
			pQ <- pi
			return
		}

		c.SendHave(pi.index)
		rQ <- &pieceResult{index: pi.index, buf: buf}
	}

}

func (ds *DownloadSession) getPieceBoundAt(index int) (int, int) {
	begin := index * ds.PieceLength
	end := begin + ds.PieceLength
	if end > ds.Length {
		end = ds.Length
	}
	return begin, end
}

func (ds *DownloadSession) Download(ctx context.Context, filepath string) error {
	piecesQueue := make(chan *pieceInfo, ds.NumPieces())
	defer close(piecesQueue)

	resultsQueue := make(chan *pieceResult, ds.NumPieces())
	defer close(resultsQueue)

	for i, hash := range ds.PieceHashes {
		// check if the piece is already downloaded
		if ds.bitfield.HasPiece(i) {
			continue
		}

		begin, end := ds.getPieceBoundAt(i)

		piecesQueue <- &pieceInfo{
			index:  i,
			hash:   hash,
			length: end - begin,
		}
	}

	// start retrieving pieces
	for _, peer := range ds.peers {
		go ds.downloadFromPeer(ctx, peer, piecesQueue, resultsQueue)
	}

	// assemble pieces
	// buf := make([]byte, ds.Length)
	donePieces := 0
	for donePieces < ds.NumPieces() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			res := <-resultsQueue
			begin, _ := ds.getPieceBoundAt(res.index)
			// copy(ds.buf[begin:end], res.buf)
			_, err := ds.fd.WriteAt(res.buf, int64(begin))
			if err != nil {
				return err
			}
			donePieces++

			ds.bitfield.SetPiece(res.index)

			// update progress to tracker server
		}
	}
	ds.done = true

	return nil
}

func (ds *DownloadSession) Close() {
	ds.fd.Close()
	if cache, ok := ds.peerInfo.cache[ds.InfoHash.String()]; ok {
		cache.Bitfield = ds.bitfield
	}
    if ds.done {
        // rename file
        delete(ds.peerInfo.downloadingPeers, ds.InfoHash.String())
        filename := strings.TrimSuffix(ds.filepath, ".tmp")
        os.Rename(ds.filepath, filename)
    }
}
