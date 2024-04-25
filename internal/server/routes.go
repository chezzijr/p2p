package server

import (
	"log/slog"
	"net"
	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/peers"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
)

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/announce", s.announceHandler)
}

func (s *FiberServer) announceHandler(c *fiber.Ctx) error {
    var req api.AnnounceRequest
    if err := c.QueryParser(&req); err != nil {
        slog.Error("Error parsing query", "error", err)
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

    connectingPeers := s.tracker.GetPeers(infoHash)
    slog.Info("Connecting peers", "infoHash", infoHash, "peers", connectingPeers)

    peerBytes := peers.Marshal(connectingPeers...)

    slog.Info("Adding peer", "infoHash", infoHash, "peerID", peerID, "peer", peer)
    s.tracker.AddPeer(infoHash, peer)

    c.Set("Content-Type", "text/plain; charset=utf-8")
    err := bencode.Marshal(c, api.AnnounceResponse{
        Interval: time.Minute * 15,
        Peers:    string(peerBytes),
    })

    if err != nil {
        slog.Error("Receving error", "error", err)
    }

    return err
}
