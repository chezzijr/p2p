package connection

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

type messageID uint8

const (
	MsgChoke messageID = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterested
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel
)

var (
	ErrInvalidMessage = errors.New("invalid message")
	ErrInvalidLength  = errors.New("invalid length")
)

type Message struct {
	ID      messageID
	Payload []byte
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1)
	buf := make([]byte, length+4)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

func ReadMsg(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return nil, nil
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      messageID(buf[0]),
		Payload: buf[1:],
	}, nil
}

func BuildRequestMsg(index, begin, length uint32) *Message {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], index)
	binary.BigEndian.PutUint32(buf[4:8], begin)
	binary.BigEndian.PutUint32(buf[8:12], length)

	return &Message{
		ID:      MsgRequest,
		Payload: buf,
	}
}

func BuildHaveMsg(index int) *Message {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(index))

	return &Message{
		ID:      MsgHave,
		Payload: buf,
	}
}

func ParseHaveMsg(msg *Message) (int, error) {
	if msg == nil || msg.ID != MsgHave {
		return 0, ErrInvalidMessage
	}

	if len(msg.Payload) != 4 {
		return 0, ErrInvalidLength
	}

	return int(binary.BigEndian.Uint32(msg.Payload)), nil
}

func ParsePieceMsg(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("Expected PIECE (ID %d), got ID %d", MsgPiece, msg.ID)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("Payload too short. %d < 8", len(msg.Payload))
	}

	parsedIndex := binary.BigEndian.Uint32(msg.Payload[0:4])
	if int(parsedIndex) != index {
		return 0, fmt.Errorf("Expected index %d, got %d", index, parsedIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if int(begin) >= len(buf) {
		return 0, fmt.Errorf("Begin offset %d is out of bounds, max = %d", begin, len(buf))
	}

	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("Data goes out of bounds. Begin = %d, len = %d, buf len = %d", begin, len(data), len(buf))
	}

	slog.Info("Received piece", "bytes", data)

	copy(buf[begin:], data)
	return len(data), nil
}

func ParseRequestMsg(msg *Message) (index uint32, begin uint32, length uint32, err error) {
	if msg.ID != MsgRequest {
		return 0, 0, 0, fmt.Errorf("Expected REQUEST (ID %d), got ID %d", MsgRequest, msg.ID)
	}

	if len(msg.Payload) != 12 {
		return 0, 0, 0, fmt.Errorf("Payload too short. %d < 12", len(msg.Payload))
	}

	index = binary.BigEndian.Uint32(msg.Payload[0:4])
	begin = binary.BigEndian.Uint32(msg.Payload[4:8])
	length = binary.BigEndian.Uint32(msg.Payload[8:12])

	return index, begin, length, nil
}
