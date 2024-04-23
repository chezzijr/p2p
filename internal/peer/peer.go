package peer

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"p2p/internal/common/torrent"
	"path"
	"strconv"
	"time"
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

	return &Peer{
		PeerID:     peerID,
		Port:       port,
		Torrents:   make([]*torrent.TorrentFile, 0),
		StopServer: make(chan struct{}, 1),
	}, nil
}

func (s *Peer) AddTorrent(t... *torrent.TorrentFile) {
    s.Torrents = append(s.Torrents, t...)
}

func (s *Peer) RunServer() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	defer lis.Close()

	// seed to server every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				for _, t := range s.Torrents {
					s.Seed(t)
				}
			case <-s.StopServer:
				ticker.Stop()
				return
			}
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
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(s.PeerID[:])},
		"port":       []string{strconv.Itoa(int(s.Port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{"0"},
	}
	base.RawQuery = params.Encode()
	trackerUrl := base.String()
	resp, err := http.Get(trackerUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ...
	return nil
}
