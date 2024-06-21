package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/peers"
	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/server/database"
	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
)

var (
	ErrAlreadyExists = errors.New("peer already exists")
)

type Sha1Hash torrent.Sha1Hash

type Tracker interface {
	AddPeer(ctx context.Context, infoHash Sha1Hash, peer peers.Peer) error
	GetPeers(ctx context.Context, infoHash Sha1Hash) ([]peers.Peer, error)
	AnnounceHandler(c *fiber.Ctx) error
}

// Stores the peers that are connecting to the server
type tracker struct {
	interval time.Duration
	redis    database.Redis
}

func NewTrackerServer(redis database.Redis) Tracker {
	return &tracker{
		interval: time.Minute * 15,
		redis:    redis,
	}
}

func (t *tracker) AddPeer(ctx context.Context, infoHash Sha1Hash, peer peers.Peer) error {
	key := string(infoHash[:])
	value := string(peers.Marshal(peer))

	err := t.redis.AddOrUpdateTTL(ctx, key, value, t.interval)
	return err
}

func (t *tracker) GetPeers(ctx context.Context, infoHash Sha1Hash) ([]peers.Peer, error) {
	key := string(infoHash[:])
	value, err := t.redis.GetAll(ctx, key)
	if err != nil {
		return nil, err
	}
	// concatenate string with no delimiter
	concat := strings.Join(value, "")

	return peers.Unmarshal([]byte(concat))
}

func (t *tracker) RemovePeer(ctx context.Context, infoHash Sha1Hash, peer peers.Peer) error {
	key := string(infoHash[:])
	value := string(peers.Marshal(peer))

	err := t.redis.Remove(ctx, key, value)
	return err
}

func (t *tracker) AnnounceHandler(c *fiber.Ctx) error {
	var req api.AnnounceRequest
	if err := c.QueryParser(&req); err != nil {
		return err
	}

	var infoHash Sha1Hash
	copy(infoHash[:], []byte(req.InfoHash))

	var peerID Sha1Hash
	copy(peerID[:], []byte(req.PeerID))

	ip := net.ParseIP(c.IP())

	peer := peers.Peer{
		Ip:   ip,
		Port: uint16(req.Port),
	}

	connectingPeers, err := t.GetPeers(context.Background(), infoHash)
	if err != nil {
		return err
	}

	peerBytes := peers.Marshal(connectingPeers...)

	// check if req.event equals "started", or "completed"
	if req.Event == api.Started || req.Event == api.Completed {
		err := t.AddPeer(context.Background(), infoHash, peer)
		if err != nil {
			slog.Error("Error adding peer", "error", err)
		}
	} else if req.Event == api.Stopped {
		// remove the peer from the tracker
		err := t.RemovePeer(context.TODO(), infoHash, peer)
		if err != nil {
			slog.Error("Error removing peer", "error", err)
		}
	} else {
		// return event error
		// return errors.New("Invalid event")
	}

	err = bencode.Marshal(c, api.AnnounceResponse{
		Interval: time.Minute * 15,
		Peers:    string(peerBytes),
	})

	if err != nil {
		slog.Error("Receving error", "error", err)
	}

	return err
}
