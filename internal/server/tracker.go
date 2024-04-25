package server

import (
	"bytes"
	"github.com/chezzijr/p2p/internal/common/peers"
    "errors"
)

var (
    ErrAlreadyExists = errors.New("peer already exists")
)

type Tracker interface {
    AddPeer(infoHash [20]byte, peer peers.Peer) error
    GetPeers(infoHash [20]byte) []peers.Peer
}

// Stores the peers that are connecting to the server
type tracker struct {
	infoHashesToPeer map[string][]peers.Peer
}

func NewTracker() Tracker {
	return &tracker{
		infoHashesToPeer: make(map[string][]peers.Peer),
	}
}

func (t *tracker) AddPeer(infoHash [20]byte, peer peers.Peer) error {
	// check if the peer is already in the tracker
    key := string(infoHash[:])
	for _, p := range t.infoHashesToPeer[key] {
        if bytes.Equal(p.Ip, peer.Ip) && p.Port == peer.Port {
            return ErrAlreadyExists
        }
	}
    t.infoHashesToPeer[key] = append(t.infoHashesToPeer[key], peer)
    return nil
}

func (t *tracker) GetPeers(infoHash [20]byte) []peers.Peer {
    key := string(infoHash[:])
	connectingPeers, ok := t.infoHashesToPeer[key]
	if !ok {
        return []peers.Peer{}
	}
    return connectingPeers
}
