package peers

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestUnmarshal(t testing.T) {
	buf := make([]byte, 6)
	buf[0] = 192
	buf[1] = 168
	buf[2] = 1
	buf[3] = 1
	buf[4] = 0x30
	buf[5] = 0x39

	peers, err := Unmarshal(buf)
	if err != nil {
		t.Errorf("Failed to unmarshall peers: %s", err)
	}

	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers[0].Ip.String() != "192.168.1.1" {
		t.Errorf("Expected ip 192.168.1.1, git %s", peers[0].Ip.String())
	}

	if peers[0].Port != 12345 {
		t.Errorf("Expected port 12345, got %d", peers[0].Port)
	}
}

func TestMarshal(t testing.T) {
	peer := Peer{
		Ip:   net.ParseIP("123.45.67.89"),
		Port: 12345,
	}

	buf := Marshal(peer)

	if len(buf) != 6 {
		t.Errorf("Expected 6 bytes, got %d", len(buf))
	}

	if buf[0] != 123 || buf[1] != 45 || buf[2] != 67 || buf[3] != 89 {
		t.Errorf("Expected ip 123.45.67.89, got %d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
	}

	if buf[4] != 0x30 || buf[5] != 0x39 {
		t.Errorf("Expected port 12345, got %d", binary.BigEndian.Uint16(buf[4:6]))
	}
}
