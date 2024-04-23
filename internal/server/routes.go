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
    copy(infoHash[:], req.InfoHash)

    var peerID [20]byte
    copy(peerID[:], req.PeerID)

    ip := net.ParseIP(c.IP())

    peer := peers.Peer{
        Ip:   ip,
        Port: uint16(req.Port),
    }

    slog.Info("Received Announce Request", "infoHash", infoHash, "peerID", peerID, "peer", peer)
    connectingPeers := s.tracker.GetPeers(infoHash)
    peerBytes := peers.Marshal(connectingPeers...)

    s.tracker.AddPeer(infoHash, peerID, peer)

    slog.Info("Sending Announce Response", "peers", connectingPeers)
    c.Set("Content-Type", "text/plain; charset=utf-8")
    c.Set("Content-Disposition", "inline")
    err := bencode.Marshal(c, api.AnnounceResponse{
        Interval: time.Minute * 15,
        Peers:    peerBytes,
    })
    if err != nil {
        slog.Error("Receving error", "error", err)
    }
    return nil
}
