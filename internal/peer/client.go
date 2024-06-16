package peer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/peers"
)

var (
	ErrInfoHashMismatch = errors.New("infohash mismatch")
	ErrInvalidMessage   = errors.New("invalid message")
)

// client to download pieces
type DownloadClient struct {
	Conn     net.Conn
	Choked   bool
	InfoHash [20]byte
	PeerID   [20]byte
	Bitfield connection.BitField

	Peer peers.Peer
}

func NewClient(ctx context.Context, p peers.Peer, peerID [20]byte, infoHash [20]byte) (*DownloadClient, error) {
	// conn, err := net.Dial("tcp", p.String())
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", p.String())
	if err != nil {
		return nil, err
	}

	slog.Info("Attempting handshake with", "peer", conn.RemoteAddr())
	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	slog.Info("Receiving bitfield from", "peer", conn.RemoteAddr())
	bf, err := recvBitField(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &DownloadClient{
		Conn:     conn,
		Choked:   true,
		InfoHash: infoHash,
		PeerID:   peerID,
		Peer:     p,
		Bitfield: bf,
	}, nil
}

func (c *DownloadClient) Close() error {
	return c.Conn.Close()
}

func (c *DownloadClient) Read() (*connection.Message, error) {
	return connection.ReadMsg(c.Conn)
}

func (c *DownloadClient) SendRequest(index, begin, length uint32) error {
	msg := connection.BuildRequestMsg(index, begin, length)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *DownloadClient) SendInterested() error {
	msg := connection.Message{ID: connection.MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *DownloadClient) SendNotInterested() error {
	msg := connection.Message{ID: connection.MsgNotInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *DownloadClient) SendHave(index int) error {
	msg := connection.BuildHaveMsg(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *DownloadClient) SendUnchoke() error {
	c.Choked = false
	msg := connection.Message{ID: connection.MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func completeHandshake(conn net.Conn, infoHash, peerID [20]byte) (*connection.Handshake, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	slog.Info("Attempting handshake with", "peer", conn.RemoteAddr())

	req := connection.NewHandshake(infoHash, peerID)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := connection.ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, ErrInfoHashMismatch
	}

	return res, nil
}

func recvBitField(conn net.Conn) (connection.BitField, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	slog.Info("Receiving bitfield from", "peer", conn.RemoteAddr())

	msg, err := connection.ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.ID != connection.MsgBitfield {
		return nil, ErrInvalidMessage
	}

	return connection.BitField(msg.Payload), nil
}
