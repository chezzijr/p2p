package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/peers"
	"github.com/chezzijr/p2p/internal/server/database"
	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
)

var (
	ErrAlreadyExists = errors.New("peer already exists")
)

type Tracker interface {
	AddPeer(infoHash [20]byte, peer peers.Peer) error
	GetPeers(infoHash [20]byte) []peers.Peer
	AnnounceHandler(c *fiber.Ctx) error
}

// Stores the peers that are connecting to the server
type tracker struct {
	infoHashesToPeer map[string][]peers.Peer
	interval         time.Duration
	redis            database.Redis
}

func NewTrackerServer(redis database.Redis) Tracker {
	return &tracker{
		infoHashesToPeer: make(map[string][]peers.Peer),
		redis:            redis,
        interval:         time.Minute * 15,
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

func (t *tracker) AnnounceHandler(c *fiber.Ctx) error {
	var req api.AnnounceRequest
	if err := c.QueryParser(&req); err != nil {
		return err
	}

	var infoHash [20]byte
	copy(infoHash[:], []byte(req.InfoHash))

	var peerID [20]byte
	copy(peerID[:], []byte(req.PeerID))

	ip := net.ParseIP(c.IP())

	peer := peers.Peer{
		Ip:   ip,
		Port: uint16(req.Port),
	}

	connectingPeers := t.GetPeers(infoHash)

	peerBytes := peers.Marshal(connectingPeers...)

	t.AddPeer(infoHash, peer)

	err := bencode.Marshal(c, api.AnnounceResponse{
		Interval: time.Minute * 15,
		Peers:    string(peerBytes),
	})

	if err != nil {
		slog.Error("Receving error", "error", err)
	}

	return err
}
