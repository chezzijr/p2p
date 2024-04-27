package server

import (
	"log/slog"
	"net"
	"time"

	"github.com/chezzijr/p2p/internal/common/torrent"
    "github.com/chezzijr/p2p/internal/common/api"
	"github.com/gofiber/fiber/v2"
	"github.com/jackpal/bencode-go"
    "github.com/chezzijr/p2p/internal/common/peers"
)

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/announce", s.announceHandler)

    s.App.Post("/upload", s.uploadHandler)
}


func (s *FiberServer) announceHandler(c *fiber.Ctx) error {
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

    connectingPeers := s.tracker.GetPeers(infoHash)

    peerBytes := peers.Marshal(connectingPeers...)

    s.tracker.AddPeer(infoHash, peer)

    err := bencode.Marshal(c, api.AnnounceResponse{
        Interval: time.Minute * 15,
        Peers:    string(peerBytes),
    })

    if err != nil {
        slog.Error("Receving error", "error", err)
    }

    return err
}

func (s *FiberServer) uploadHandler(ctx *fiber.Ctx) error {
    form, err := ctx.MultipartForm()
    if err != nil {
        return err
    }

    fileHeaders, ok := form.File["metainfo_files"]
    if !ok {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "no file found",
        })
    }

    if len(fileHeaders) == 0 {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "no file found",
        })
    }

    torrents := make([]*torrent.TorrentFile, 0, len(fileHeaders))

    for _, fileHeader := range fileHeaders {
        file, err := fileHeader.Open()
        if err != nil {
            return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Could not open file",
            })
        }
        defer file.Close()

        torrent, err := torrent.OpenFromReader(file)
        if err != nil {
            return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "File is not a valid torrent file",
            })
        }

        torrents = append(torrents, torrent)
    }

    s.db.AddBulkTorrents(torrents)

    return nil
}

