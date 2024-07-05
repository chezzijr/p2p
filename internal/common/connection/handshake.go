package connection

import (
	"errors"
	"io"
)

var (
	ErrInvalidProtocol = errors.New("invalid protocol")
)

type Handshake struct {
	Protocol string
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Protocol: "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Serialize() []byte {
    // 1 byte for protocol length
    // 8 bytes for reserved bytes
    // 20 bytes for info hash
    // 20 bytes for peer id
    // so the total length is 49 + len(h.Protocol)

	buf := make([]byte, len(h.Protocol)+49)
	buf[0] = byte(len(h.Protocol))
	curr := 1
	curr += copy(buf[curr:], h.Protocol)
	curr += copy(buf[curr:], make([]byte, 8))
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0]) // protocol length

	if pstrlen == 0 {
		return nil, ErrInvalidProtocol
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := Handshake{
		Protocol: string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}
