package peer

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"log/slog"
	"net"
	"os"
	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/torrent"
	"time"
)

var (
	ErrTorrentNotFound = errors.New("torrent not found")
    ErrOutOfBound = errors.New("out of bound")
)

// for other peers to download from this peer
type UploadSession struct {
	conn       net.Conn
	peerID     [20]byte
	// fd         io.Reader
    pieces     [][]byte
	t          *torrent.TorrentFile
	choked     bool
	interested bool
	// used to timeout connection
}

func (s *Peer) respondHandshake(conn net.Conn) (*torrent.TorrentFile, error) {
	// handshake
	req, err := connection.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	// find corresponding torrent file
	var t *torrent.TorrentFile
	for _, torrent := range s.Torrents {
		if bytes.Equal(req.InfoHash[:], torrent.InfoHash[:]) {
			t = torrent
			break
		}
	}

	// if the torrent file is not found, reject the connection
	var infoHash [20]byte
	if t == nil || !bytes.Equal(req.InfoHash[:], t.InfoHash[:]) {
		infoHash = sha1.Sum([]byte("invalid infohash"))
	} else {
		infoHash = t.InfoHash
	}

	res := connection.NewHandshake(infoHash, s.PeerID)
	_, err = conn.Write(res.Serialize())
	if err != nil {
		return nil, err
	}

	if t == nil {
		return nil, ErrTorrentNotFound
	}

	return t, nil
}

func (ds *UploadSession) sendBitfield(conn net.Conn) error {
	// bitfield
	bufLen := len(ds.t.PieceHashes)/8 + 1
	// create a bitfield with all pieces set to 1
	// because we have all pieces
	bf := make([]byte, bufLen)
	for i := range bf {
		bf[i] = 0xff
	}
	offset := len(ds.t.PieceHashes) % 8
	bf[bufLen-1] = (0xff >> uint8(7-offset)) << uint8(7-offset)

	msg := &connection.Message{
		ID:      connection.MsgBitfield,
		Payload: bf,
	}
	_, err := conn.Write(msg.Serialize())
	if err != nil {
		return err
	}

	return nil
}

func (session *UploadSession) getPiece(index, begin, length uint32) ([]byte, error) {
    if index >= uint32(len(session.pieces)) {
        return nil, ErrOutOfBound
    }

    if begin + length > uint32(len(session.pieces[index])) {
        return nil, ErrOutOfBound
    }

	buf := make([]byte, length + 8)

    binary.BigEndian.PutUint32(buf[0:4], index)
    binary.BigEndian.PutUint32(buf[4:8], begin)

    copy(buf[8:], session.pieces[index][begin:begin+length])
	return buf, nil
}

func (session *UploadSession) uploadToPeer() error {

    slog.Info("Sending bitfield to peer")
	err := session.sendBitfield(session.conn)
	if err != nil {
        slog.Error("Failed to send bitfield", "error", err)
		return err
	}

    session.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

	for {
		msg, err := session.readMessage()
		if err != nil {
            // i/o timeout
            if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                slog.Info("Connection timeout, closing connection")
                return nil
            }
            slog.Error("Failed to read message", "error", err)
		}

		switch msg.ID {
		case connection.MsgRequest:
			index, begin, length, err := connection.ParseRequestMsg(msg)
            slog.Info("Received request", "index", index, "begin", begin, "length", length)
			if err != nil {
				slog.Error("Failed to parse request message", "error", err)
				continue
			}
			buf, err := session.getPiece(index, begin, length)
			if err != nil {
				slog.Error("Failed to get piece", "error", err)
				continue
			}
			msg := &connection.Message{
				ID:      connection.MsgPiece,
				Payload: buf,
			}
            slog.Info("Sending piece", "index", index, "begin", begin, "length", length)
			_, err = session.conn.Write(msg.Serialize())
			if err != nil {
				slog.Error("Failed to send piece", "error", err)
			}
		case connection.MsgInterested:
			session.interested = true
        case connection.MsgNotInterested:
            break
		case connection.MsgUnchoke:
			session.choked = false
		case connection.MsgHave:
			// we were informed that the peer has a piece
			index, err := connection.ParseHaveMsg(msg)
			if err != nil {
				slog.Error("Failed to parse have message", "error", err)
				continue
			}
			slog.Info("Peer has piece", "index", index)
		}
	}
}

func (session *UploadSession) readMessage() (*connection.Message, error) {
	return connection.ReadMsg(session.conn)
}

func (s *Peer) handleConn(conn net.Conn) error {
	defer conn.Close()

	// handshake on a torrent file
	// if the torrent file is not found, reject the connection
    slog.Info("Respond to handshake")
	t, err := s.respondHandshake(conn)
	if err != nil {
        slog.Error("Failed to respond to handshake", "error", err)
		return err
	}

    slog.Info("Opening file", "file", t.Name)
	fd, err := os.Open(t.Name)
	if err != nil {
		return err
	}
    defer fd.Close()

    fileStat, err := fd.Stat()
    if err != nil {
        return err
    }

    pieces := make([][]byte, len(t.PieceHashes))
    for i := 0; i < len(t.PieceHashes); i++ {
        begin := int64(i) * int64(t.PieceLength)
        end := min(begin + int64(t.PieceLength), fileStat.Size())

        pieces[i] = make([]byte, end - begin)
        _, err := fd.Read(pieces[i][:])
        if err != nil {
            return err
        }
    }

	ds := &UploadSession{
		conn:       conn,
		t:          t,
		peerID:     s.PeerID,
        pieces:     pieces,
		choked:     true,
		interested: false,
	}

	return ds.uploadToPeer()
}
