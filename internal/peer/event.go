package peer

import "github.com/chezzijr/p2p/internal/common/torrent"

type Event interface {
	Consume(p *Peer) error
}

type EventDownload struct {
	DownloadPath string
	TorrentPath  string
}

func (e *EventDownload) Consume(p *Peer) error {
	tf, err := torrent.Open(e.TorrentPath)
	if err != nil {
		return err
	}
	return p.Download(tf, e.DownloadPath)
}

type EventUpload struct {
	FilePath    string
	TorrentPath string
	// Used for announcing the file
	Interval int
}

func (e *EventUpload) Consume(p *Peer) error {
	tf, err := torrent.Open(e.TorrentPath)
	if err != nil {
		return err
	}
	p.AddTorrent(tf)
	return nil
}
