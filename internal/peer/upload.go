package peer

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/torrent"
)

var (
	ErrTorrentNotFound = errors.New("torrent not found")
	ErrOutOfBound      = errors.New("out of bound")
)

// for other peers to download from this peer
type UploadSession struct {
	conn   net.Conn
	peerID [20]byte
	// fd         io.Reader
	pieces     [][]byte
	t          *torrent.TorrentFile
	choked     bool
	interested bool
	// used to timeout connection
}

func (p *Peer) respondHandshake(conn net.Conn) (*torrent.TorrentFile, error) {
	// handshake
	req, err := connection.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	// find corresponding torrent file
    t, ok := p.seedingTorrents[string(req.InfoHash[:])]

	// if the torrent file is not found, reject the connection
	var infoHash [20]byte
	if !ok {
		infoHash = sha1.Sum([]byte("invalid infohash"))
	} else {
		infoHash = t.InfoHash
	}

	res := connection.NewHandshake(infoHash, p.PeerID)
	_, err = conn.Write(res.Serialize())
	if err != nil {
		return nil, err
	}

	if t == nil {
		return nil, ErrTorrentNotFound
	}

	return t, nil
}

func (us *UploadSession) sendBitfield(conn net.Conn) error {
	// bitfield
	bufLen := len(us.t.PieceHashes)/8 + 1
	// create a bitfield with all pieces set to 1
	// because we have all pieces
	bf := make([]byte, bufLen)
	for i := range bf {
		bf[i] = 0xff
	}
	offset := len(us.t.PieceHashes) % 8
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

func (us *UploadSession) getPiece(index, begin, length uint32) ([]byte, error) {
	if index >= uint32(len(us.pieces)) {
		return nil, ErrOutOfBound
	}

	if begin+length > uint32(len(us.pieces[index])) {
		return nil, ErrOutOfBound
	}

	buf := make([]byte, length+8)

	binary.BigEndian.PutUint32(buf[0:4], index)
	binary.BigEndian.PutUint32(buf[4:8], begin)

	copy(buf[8:], us.pieces[index][begin:begin+length])
	return buf, nil
}

func (session *UploadSession) uploadToPeer() error {
	err := session.sendBitfield(session.conn)
	if err != nil {
		slog.Error("Failed to send bitfield", "error", err)
		return err
	}

	session.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

	for {
		msg, err := session.readMessage()
		if err != nil {
			// i/o timeout or connection closed
			switch errNet, ok := err.(net.Error); {
			case errors.Is(err, net.ErrClosed),
				errors.Is(err, io.EOF),
				errors.Is(err, syscall.EPIPE):
				slog.Info("Connection closed")
				return nil
            case ok && errNet.Timeout():
                slog.Info("Connection timeout")
                return nil
			default:
				slog.Error("Failed to read message", "error", err)
				continue
			}

		}

		switch msg.ID {
		case connection.MsgRequest:
			index, begin, length, err := connection.ParseRequestMsg(msg)
			if err != nil {
				continue
			}
			buf, err := session.getPiece(index, begin, length)
			if err != nil {
				continue
			}
			msg := &connection.Message{
				ID:      connection.MsgPiece,
				Payload: buf,
			}
			_, err = session.conn.Write(msg.Serialize())
			if err != nil {
                continue
			}
		case connection.MsgInterested:
			session.interested = true
            err := session.sendUnchoke()
            if err != nil {
                slog.Error("Failed to send unchoke", "error", err)
                continue
            }
            session.choked = false
            
		case connection.MsgNotInterested:
			break
		case connection.MsgHave:
			// we were informed that the peer has a piece
			index, err := connection.ParseHaveMsg(msg)
			if err != nil {
				continue
			}
			slog.Info("Peer has piece", "index", index)
		}
	}
}

func (us *UploadSession) sendUnchoke() error {
    msg := &connection.Message{
        ID: connection.MsgUnchoke,
    }
    _, err := us.conn.Write(msg.Serialize())
    return err
}

func (session *UploadSession) readMessage() (*connection.Message, error) {
	return connection.ReadMsg(session.conn)
}

func (p *Peer) handleConn(conn net.Conn) error {
	defer conn.Close()

	// handshake on a torrent file
	// if the torrent file is not found, reject the connection
	slog.Info("Respond to handshake")
	t, err := p.respondHandshake(conn)
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

	pieces := make([][]byte, t.NumPieces())
	for i := 0; i < len(t.PieceHashes); i++ {
		begin := int64(i) * int64(t.PieceLength)
		end := min(begin+int64(t.PieceLength), fileStat.Size())

		pieces[i] = make([]byte, end-begin)
		_, err := fd.Read(pieces[i][:])
		if err != nil {
			return err
		}
	}

	us := &UploadSession{
		conn:       conn,
		t:          t,
		peerID:     p.PeerID,
		pieces:     pieces,
		choked:     true,
		interested: false,
	}

	return us.uploadToPeer()
}
