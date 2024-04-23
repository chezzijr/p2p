package tracker

import (
	"bytes"
	"github.com/chezzijr/p2p/internal/common/peers"
    "errors"
)

var (
    ErrAlreadyExists = errors.New("peer already exists")
)

type Peer struct {
	ID [20]byte
	peers.Peer
}

type Tracker struct {
	infoHashesToPeer map[[20]byte][]Peer
}

func New() *Tracker {
	return &Tracker{
		infoHashesToPeer: make(map[[20]byte][]Peer),
	}
}

func (t *Tracker) AddPeer(infoHash [20]byte, peerID [20]byte, peer peers.Peer) error {
	// check if the peer is already in the tracker
	for _, p := range t.infoHashesToPeer[infoHash] {
		if bytes.Equal(p.ID[:], peerID[:]) {
            return ErrAlreadyExists
		}
	}
	t.infoHashesToPeer[infoHash] = append(t.infoHashesToPeer[infoHash], Peer{ID: peerID, Peer: peer})
    return nil
}

func (t *Tracker) GetPeers(infoHash [20]byte) []peers.Peer {
	connectingPeers, ok := t.infoHashesToPeer[infoHash]
	if !ok {
        return []peers.Peer{}
	}

    ps := make([]peers.Peer, len(connectingPeers))
    for i, p := range connectingPeers {
        ps[i] = p.Peer
    }

    return ps
}
