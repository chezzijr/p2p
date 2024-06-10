package peer

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/jackpal/bencode-go"
)

type Peer struct {
	config           *Config
    cache            CachedFilesMap
	events           chan Event
	connectingPeers  map[string]net.Conn
	downloadingPeers map[string]*DownloadSession
	uploadingPeers   map[string]*UploadSession
	seedingTorrents  map[string]*torrent.TorrentFile

	PeerID [20]byte
	Port   uint16
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

    cache, err := LoadCache(cfg.CachePath)
    if err != nil {
        return nil, err
    }

	return &Peer{
		config:           cfg,
        cache:            cache,
		events:           make(chan Event, 10),
		connectingPeers:  make(map[string]net.Conn),
		downloadingPeers: make(map[string]*DownloadSession),
		uploadingPeers:   make(map[string]*UploadSession),
        seedingTorrents:  make(map[string]*torrent.TorrentFile),
		PeerID:           peerID,
		Port:             port,
	}, nil
}


// Write downloaded data to file.ext.tmp, where file.ext is the file name
// When finished, rename file.ext.tmp to file.ext
// This function is a goroutine
func (p *Peer) download(ctx context.Context, t *torrent.TorrentFile, filepath string) error {
    session, err := p.NewDownloadSession(t, filepath)
    if err != nil {
        return err
    }
    defer session.Close()

	// we temporarily save the whole file in memory
	//! TODO: save to disk when each piece is downloaded
    // temporary context
	err = session.Download(ctx, filepath)
	if err != nil {
		slog.Error("Failed to download", "error", err)
		return err
	}

	return nil
}

func (s *Peer) seed(t *torrent.TorrentFile, event api.AnnounceEvent) (*api.AnnounceResponse, error) {
    slog.Info("Seeding", "url", t.Announce)
	base, err := url.Parse(t.Announce)
	if err != nil {
		return nil, err
	}
	req := api.AnnounceRequest{
		InfoHash:   string(t.InfoHash[:]),
		PeerID:     string(s.PeerID[:]),
		Port:       s.Port,
		Uploaded:   0,
		Downloaded: int(t.Length), // seed the whole file
		Left:       0,
		Event:      event,
	}
	base.RawQuery = req.ToUrlValues().Encode()
	trackerUrl := base.String()

	resp, err := http.Get(trackerUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Failed to seed: %s", resp.Status)
    }

	var respBody api.AnnounceResponse
	err = bencode.Unmarshal(resp.Body, &respBody)
	if err != nil {
		return nil, err
	}

	slog.Info("Seeding", "url", t.Announce, "response", respBody)

	return &respBody, nil
}

// New event-driven architecture
func (p *Peer) seedTorrent(ctx context.Context, tf *torrent.TorrentFile) error {
	p.seedingTorrents[tf.InfoHash.String()] = tf
	defer delete(p.seedingTorrents, tf.InfoHash.String())

	resp, err := p.seed(tf, api.Started)
	if err != nil {
		return err
	}
	// resp.Interval in minutes
	interval := time.Minute * resp.Interval

	for {
		select {
        case <-ctx.Done():
            _, err = p.seed(tf, api.Stopped)
            return err
		case <-time.After(interval):
			resp, err = p.seed(tf, api.Started)
			if err != nil {
				return err
			}
		}
	}
}

// Control the peer
func (p *Peer) RegisterEvent(e Event) {
	p.events <- e
}

func (p *Peer) Run(ctx context.Context) error {
    //! TODO: Initialize server
    var lc net.ListenConfig
    lis, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", p.Port))
    if err != nil {
        return err
    }
    defer lis.Close()

    go func(listener net.Listener) {
        for {
            conn, err := listener.Accept()
            if err != nil {
                slog.Error("Failed to accept connection", "error", err)
                continue
            }
            go p.handleConn(conn)
        }
    }(lis)

	// Waiting for events
    for {
        select {
        case <-ctx.Done():
            return nil
        case e := <-p.events:
            go func() {
                err := e.Handle(ctx, p)
                if err != nil {
                    slog.Error("Failed to handle event", "error", err)
                }
            }()
        }
    }
}

// graceful shutdown
func (p *Peer) Close() {
    for _, session := range p.downloadingPeers {
        session.Close()
    }
    // save cache
    p.cache.SaveCache(p.config.CachePath)
}
