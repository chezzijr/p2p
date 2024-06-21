package peer

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
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
	done             chan struct{}

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

	InitLogger(os.Stderr)

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
		done:             make(chan struct{}, 1),
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
	defer func() {
		session.Close()
		delete(p.downloadingPeers, t.InfoHash.String())
		// rename file
		if session.done {
			os.Rename(filepath+t.Name+".tmp", filepath+t.Name)
		}
		// seed the file
		if session.done && p.config.SeedOnFileDownloaded {
			go p.seedTorrent(ctx, t)
		}
	}()

	err = session.Download(ctx, filepath)
	if err != nil {
		slog.Error("Failed to download", "error", err)
		return err
	}

	return nil
}

func (s *Peer) updateToTracker(t *torrent.TorrentFile, event api.AnnounceEvent, uploadSize, downloadSize int) (*api.AnnounceResponse, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return nil, err
	}
	req := api.AnnounceRequest{
		InfoHash:   string(t.InfoHash[:]),
		PeerID:     string(s.PeerID[:]),
		Port:       s.Port,
		Uploaded:   uploadSize,
		Downloaded: downloadSize,
		Left:       int(t.Length) - downloadSize,
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

	return &respBody, nil
}

// New event-driven architecture
func (p *Peer) seedTorrent(ctx context.Context, tf *torrent.TorrentFile) error {
	p.seedingTorrents[tf.InfoHash.String()] = tf
	defer delete(p.seedingTorrents, tf.InfoHash.String())

	resp, err := p.updateToTracker(tf, api.Started, 0, int(tf.Length))
	if err != nil {
		return err
	}
	// resp.Interval in minutes
	interval := time.Minute * resp.Interval

	for {
		select {
		case <-ctx.Done():
			_, err = p.updateToTracker(tf, api.Stopped, 0, 0)
			return err
		case <-time.After(interval):
			resp, err = p.updateToTracker(tf, api.Started, 0, int(tf.Length))
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
	defer p.Close()

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
				return
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
				logger.Info("Handling event", "event", e.Name())
				err := e.Handle(ctx, p)
				if err != nil {
					logger.Error("Failed to handle event", "event", e.Name(), "error", err)
				}
			}()
		}
	}
}

// graceful shutdown
func (p *Peer) Close() {
	logger.Info("Closing peer")
	for _, session := range p.downloadingPeers {
		session.Close()
	}
	// save cache
	p.cache.SaveCache(p.config.CachePath)

	time.Sleep(time.Second * 3)
	p.done <- struct{}{}
}

func (p *Peer) Done() <-chan struct{} {
	return p.done
}
