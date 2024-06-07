package peer

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/connection"
	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/jackpal/bencode-go"
)

type Peer struct {
	config           *Config
	events           chan Event
	connectingPeers  map[string]net.Conn
	downloadingPeers map[string]*DownloadSession
	uploadingPeers   map[string]*UploadSession
	seedingTorrents  map[string]*torrent.TorrentFile

	PeerID [20]byte
	Port   uint16
	// used to seed to server for other peers to download
	// must have the files
	Torrents   []*torrent.TorrentFile
	StopServer chan struct{}
}

func NewPeer(port uint16) (*Peer, error) {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return nil, err
	}

	slog.Info("Joining the network", "peerID", peerID, "port", port)
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return &Peer{
		config:           cfg,
		events:           make(chan Event),
		connectingPeers:  make(map[string]net.Conn),
		downloadingPeers: make(map[string]*DownloadSession),
		uploadingPeers:   make(map[string]*UploadSession),
		PeerID:           peerID,
		Port:             port,
		Torrents:         make([]*torrent.TorrentFile, 0),
		StopServer:       make(chan struct{}, 1),
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

func (p *Peer) Download(t *torrent.TorrentFile, filepath string) error {
    session, err := p.NewUploadSession(t, filepath)

	// we temporarily save the whole file in memory
	//! TODO: save to disk when each piece is downloaded
	buf, err := session.Download(filepath)
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

func (s *Peer) Seed(t *torrent.TorrentFile) (*api.AnnounceResponse, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer resp.Body.Close()

	var respBody api.AnnounceResponse
	err = bencode.Unmarshal(resp.Body, &respBody)
	if err != nil {
		return nil, err
	}

	slog.Info("Seeding", "url", t.Announce, "response", respBody)

	return &respBody, nil
}

// New event-driven architecture
func (p *Peer) seedTorrent(tf *torrent.TorrentFile) error {
	p.seedingTorrents[tf.InfoHash.String()] = tf
	defer delete(p.seedingTorrents, tf.InfoHash.String())

	resp, err := p.Seed(tf)
	if err != nil {
		return err
	}
	// resp.Interval in minutes
	interval := time.Minute * resp.Interval

	for {
		select {
		case <-p.StopServer:
			return nil
		case <-time.After(interval):
			resp, err = p.Seed(tf)
			if err != nil {
				return err
			}
		}
	}
}

func (p *Peer) RegisterEvent(e Event) {
	p.events <- e
}

func (p *Peer) Run() {
    //! TODO: Initialize server

	// Waiting for events
	for e := range p.events {
		e.Consume(p)
	}
}

// graceful shutdown
func (p *Peer) Stop() {
}

func (p *Peer) NewUploadSession(t *torrent.TorrentFile, path string) (*DownloadSession, error) {
    initialPeers, err := t.RequestPeers(p.PeerID, p.Port)
    if err != nil {
        return nil, err
    }

    if len(initialPeers) == 0 {
        return nil, fmt.Errorf("No peers available")
    }

    //! TODO: load bitfield from cache
    // if not exist then empty bitfield
    bitfield := connection.NewBitField(t.NumPieces())

    session := &DownloadSession{
        Peer: p,
        TorrentFile: t,
        bitfield: bitfield,
        peers: initialPeers,
    }
    return session, nil
}

