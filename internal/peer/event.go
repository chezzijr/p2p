package peer

import (
	"context"

	"github.com/chezzijr/p2p/internal/common/torrent"
)

type Event interface {
	Handle(ctx context.Context, p *Peer) error
}

type EventDownload struct {
	DownloadPath string
	TorrentPath  string
}

func (e *EventDownload) Handle(ctx context.Context, p *Peer) error {
	tf, err := torrent.Open(e.TorrentPath)
	if err != nil {
		return err
	}
	return p.Download(ctx, tf, e.DownloadPath)
}

type EventUpload struct {
	FilePath    string
	TorrentPath string
	// Used for announcing the file
	Interval int
}

func (e *EventUpload) Handle(ctx context.Context, p *Peer) error {
	tf, err := torrent.Open(e.TorrentPath)
	if err != nil {
		return err
	}
    return p.seedTorrent(ctx, tf)
}
