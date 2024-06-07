package peer

import (
	"bytes"
	"crypto/sha1"
	"errors"
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
    *Peer
	*torrent.TorrentFile
	peers    []peers.Peer
	bitfield connection.BitField
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

func (ts *DownloadSession) downloadFromPeer(peer peers.Peer, pQ chan *pieceInfo, rQ chan *pieceResult) {
	c, err := NewClient(peer, ts.PeerID, ts.InfoHash)
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

func (ts *DownloadSession) getPieceBoundAt(index int) (int, int) {
	begin := index * ts.PieceLength
	end := begin + ts.PieceLength
	if end > ts.Length {
		end = ts.Length
	}
	return begin, end
}

func (ts *DownloadSession) Download(filepath string) ([]byte, error) {
	piecesQueue := make(chan *pieceInfo, ts.NumPieces())
	defer close(piecesQueue)

	resultsQueue := make(chan *pieceResult, ts.NumPieces())
	defer close(resultsQueue)

	for i, hash := range ts.PieceHashes {
		begin, end := ts.getPieceBoundAt(i)

		piecesQueue <- &pieceInfo{
			index:  i,
			hash:   hash,
			length: end - begin,
		}
	}

	// start retrieving pieces
	for _, peer := range ts.peers {
		go ts.downloadFromPeer(peer, piecesQueue, resultsQueue)
	}

	// assemble pieces
	buf := make([]byte, ts.Length)
	donePieces := 0
	for donePieces < len(ts.PieceHashes) {
		res := <-resultsQueue
		begin, end := ts.getPieceBoundAt(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		ts.bitfield.SetPiece(res.index)

		// update progress to tracker server
	}

	return buf, nil
}
