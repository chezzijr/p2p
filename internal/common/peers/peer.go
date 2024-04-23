package peers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

var (
    ErrMalformedPeers = errors.New("Received malformed peers")
)

type Peer struct {
	Ip   net.IP
	Port uint16
}

// 4 bytes for ip
// 2 bytes for port
const peerSize = 6

func Unmarshal(peersBin []byte) ([]Peer, error) {
    if len(peersBin) % peerSize != 0 {
        return nil, ErrMalformedPeers
    }
    numPeers := len(peersBin) / peerSize
    peers := make([]Peer, numPeers)
    for i := 0; i < numPeers; i++ {
        offset := i * peerSize
        peers[i].Ip = net.IP(peersBin[offset : offset+4])
        peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
    }
    return peers, nil
}

func Marshal(peers... Peer) []byte {
    peersBin := make([]byte, len(peers) * peerSize)
    for i, peer := range peers {
        offset := i * peerSize
        copy(peersBin[offset:offset+4], peer.Ip)
        binary.BigEndian.PutUint16(peersBin[offset+4:offset+6], peer.Port)
    }
    return peersBin
}

func (p Peer) String() string {
    return net.JoinHostPort(p.Ip.String(), fmt.Sprintf("%d", p.Port))
}
