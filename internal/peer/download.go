package peer

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"log/slog"
	"p2p/internal/common/connection"
	"p2p/internal/common/peers"
	"p2p/internal/common/torrent"
	"time"
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
	peerID [20]byte
	peers  []peers.Peer
	*torrent.TorrentFile
}

func NewDownloadSession(peerID [20]byte, peers []peers.Peer, tf *torrent.TorrentFile) *DownloadSession {
    return &DownloadSession{
        peerID:       peerID,
        peers:        peers,
        TorrentFile:  tf,
    }
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
        slog.Info("Received piece", "index", s.index, "downloaded", s.downloaded)
	}
	return nil
}

func checkIntegrity(buf []byte, piece *pieceInfo) error {
	hash := sha1.Sum(buf)
    slog.Info("Checking integrity", "expected", piece.hash, "actual", hash)
	if !bytes.Equal(hash[:], piece.hash[:]) {
		return ErrIntegrity
	}
	return nil
}

func attemptDownloadPiece(c *DownloadClient, pi *pieceInfo) ([]byte, error) {
    slog.Info("Create new piece download session", "index", pi.index, "length", pi.length)
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
				if pi.length - session.requested < blockSize {
					blockSize = pi.length - session.requested
				}

                slog.Info("Sending request", "index", pi.index, "begin", session.requested, "length", blockSize)

				err := c.SendRequest(uint32(pi.index), uint32(session.requested), uint32(blockSize))
				if err != nil {
                    slog.Error("Failed to send request", "index", pi.index, "begin", session.requested, "length", blockSize, "error", err)
					return nil, err
				}

				session.backlog++
				session.requested += blockSize
			}
		}

        slog.Info("Reading message", "index", pi.index)
		err := session.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return session.buf, nil
}

func (ts *DownloadSession) downloadFromPeer(peer peers.Peer, pQ chan *pieceInfo, rQ chan *pieceResult) {
    slog.Info("Connecting to peer", "peer", peer.String())
	c, err := NewClient(peer, ts.peerID, ts.InfoHash)
	if err != nil {
		slog.Error("Failed to handshake with peer", "peer", peer.String(), "error", err)
		return
	}

	defer c.Close()
	slog.Info("Handshake completed", "peer", peer.String())

    slog.Info("Sending unchoke", "peer", peer.String())
	c.SendUnchoke()

    slog.Info("Sending interested", "peer", peer.String())
	c.SendInterested()

	for pi := range pQ {
		if !c.Bitfield.HasPiece(pi.index) {
			pQ <- pi
			continue
		}

        slog.Info("Downloading piece", "index", pi.index, "peer", peer.String())
		// download piece
        buf, err := attemptDownloadPiece(c, pi)
        if err != nil {
            slog.Error("Failed to download piece", "index", pi.index, "peer", peer.String(), "error", err)
            pQ <- pi
            return
        }
        slog.Info("Downloaded piece", "bytes", buf[:], "index", pi.index, "length", pi.length)

        slog.Info("Checking integrity", "index", pi.index, "peer", peer.String())
        if err := checkIntegrity(buf, pi); err != nil {
            slog.Error("Integrity check failed", "index", pi.index, "peer", peer.String())
            pQ <- pi
            return
        }

        slog.Info("Downloaded piece", "index", pi.index, "length", pi.length, "peer", peer.String())
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

func (ts *DownloadSession) Download() ([]byte, error) {
	slog.Info("Start downloading", "filename", ts.Name)
    slog.Info("Torrent description", "length", ts.Length, "piece length", ts.PieceLength, "num pieces", len(ts.PieceHashes))

	piecesQueue := make(chan *pieceInfo, len(ts.PieceHashes))
	defer close(piecesQueue)

	resultsQueue := make(chan *pieceResult)
	defer close(resultsQueue)

	for i, hash := range ts.PieceHashes {
		begin, end := ts.getPieceBoundAt(i)

		piecesQueue <- &pieceInfo{
			index:  i,
			hash:   hash,
			length: end - begin,
		}
        slog.Info("Enqueued piece", "index", i, "length", end - begin, "hash", hash)
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

        slog.Info("Downloaded piece", "index", res.index, "progress", donePieces*100/len(ts.PieceHashes))

		// update progress to tracker server
	}

	return buf, nil
}
