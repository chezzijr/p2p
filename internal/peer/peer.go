package peer

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/torrent"
)

type Peer struct {
	PeerID [20]byte
	Port   uint16

	// used to seed to server for other peers to download
	// must have the files
	Torrents []*torrent.TorrentFile

	StopServer chan struct{}
}

func NewPeer(port uint16) (*Peer, error) {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return nil, err
	}

	slog.Info("Joining the network", "peerID", peerID, "port", port)

	return &Peer{
		PeerID:     peerID,
		Port:       port,
		Torrents:   make([]*torrent.TorrentFile, 0),
		StopServer: make(chan struct{}, 1),
	}, nil
}

func (s *Peer) AddTorrent(t ...*torrent.TorrentFile) {
	s.Torrents = append(s.Torrents, t...)
}

func (s *Peer) RunServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	defer lis.Close()

	go func() {
		for {
            for _, t := range s.Torrents {
                s.Seed(t)
            }
            time.Sleep(10 * time.Second)
		}
	}()

	slog.Info("Start accepting connections", "port", s.Port)
	for {
		conn, err := lis.Accept()
		if err != nil {
			slog.Error("Failed to accept connection", "error", err)
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Peer) Download(t *torrent.TorrentFile, filepath string) error {
	slog.Info("Requesting peers from tracker", "torrent", t.Name)
	peers, err := t.RequestPeers(s.PeerID, s.Port)
	if err != nil {
		slog.Error("Failed to request peers", "error", err)
		return err
	}

    if len(peers) == 0 {
        slog.Error("No peers available")
        return fmt.Errorf("No active peers available")
    }

	slog.Info("Downloading", "torrent", t.Name)
	session := NewDownloadSession(s.PeerID, peers, t)
	buf, err := session.Download()
	if err != nil {
		slog.Error("Failed to download", "error", err)
		return err
	}

	f, err := os.Create(path.Join(filepath, t.Name))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(buf)
	return err
}

func (s *Peer) Seed(t *torrent.TorrentFile) error {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return err
	}
    req := api.AnnounceRequest{
        InfoHash:   string(t.InfoHash[:]),
        PeerID:     string(s.PeerID[:]),
        Port:       s.Port,
        Uploaded:   0,
        Downloaded: int(t.Length),
        Left:       0,
        Event:      "started",
    }
	base.RawQuery = req.ToUrlValues().Encode()
	trackerUrl := base.String()

	resp, err := http.Get(trackerUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	slog.Info("Seeding", "url", t.Announce, "response", string(respBody))

	// ...
	return nil
}
