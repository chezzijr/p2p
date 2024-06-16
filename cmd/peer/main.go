package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/peer"
)

var (
	port            = flag.Uint("port", 1234, "port to listen on")
	seedingTorrent  = flag.String("seed", "", "torrent file to seed")
	downloadTorrent = flag.String("download", "", "torrent file to leech")
	trackerURL      = flag.String("tracker", "http://localhost:8080/announce", "tracker URL")
	file            = flag.String("file", "", "file to create torrent from")
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	// get first argument
	cmd := flag.Arg(0)
	switch cmd {
	case "list":
		slog.Info("Searching for torrents")
	case "create":
		if *file == "" {
			slog.Error("No file provided")
			return
		}
		slog.Info("Creating torrent")
		ut, err := torrent.GenerateTorrentFromSingleFile(*file, *trackerURL, 1024)
		if err != nil {
			panic(err)
		}
		ut.Save(ut.Name + ".torrent")

	case "run":
		p, err := peer.NewPeer(uint16(*port))
		if err != nil {
			panic(err)
		}

		if *seedingTorrent != "" {
			filename := *seedingTorrent
			ut, err := torrent.GenerateTorrentFromSingleFile(filename, *trackerURL, 1024)
			if err != nil {
				panic(err)
			}

			err = ut.Save(ut.Name + ".torrent")
			if err != nil {
				panic(err)
			}

			p.RegisterEvent(&peer.EventUpload{
				FilePath:    filename,
				TorrentPath: ut.Name + ".torrent",
			})
		}

		if *downloadTorrent != "" {
			dt, err := torrent.Open(*downloadTorrent)
			if err != nil {
				panic(err)
			}

			slog.Info("Downloading", "torrent", dt.Name)
            p.RegisterEvent(&peer.EventDownload{
                DownloadPath: "tests/",
                TorrentPath: *downloadTorrent,
            })
		}

		slog.Info("Listening", "port", *port)

        ctx, cancel := context.WithCancel(context.Background())
		err = p.Run(ctx)
		if err != nil {
            panic(err)
		}
        defer cancel()
	default:
		slog.Error("Unknown command")
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
