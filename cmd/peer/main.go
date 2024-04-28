package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"

	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/peer"
)

var (
	port            = flag.Uint("port", 1234, "port to listen on")
	seedingTorrent  = flag.String("seed", "", "torrent file to seed")
	downloadTorrent = flag.String("download", "", "torrent file to leech")
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
	case "search":
		slog.Info("Searching for torrents")

	case "run":
		p, err := peer.NewPeer(uint16(*port))
		if err != nil {
			panic(err)
		}

		if *seedingTorrent != "" {
			filename := *seedingTorrent
			ut, err := torrent.GenerateTorrentFromSingleFile(filename, "http://localhost:8081/announce", 1024)
			if err != nil {
				panic(err)
			}

			err = ut.Save(ut.Name + ".torrent")
			if err != nil {
				panic(err)
			}

			slog.Info("Seeding", "torrent", ut.Name)
			p.AddTorrent(ut)
		}

		if *downloadTorrent != "" {
			dt, err := torrent.Open(*downloadTorrent)
			if err != nil {
				panic(err)
			}

			slog.Info("Downloading", "torrent", dt.Name)
			go func() {
				err = p.Download(dt, "tests")
				if err != nil {
					panic(err)
				}
			}()
		}

		slog.Info("Listening", "port", *port)

		err = p.RunServer()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("Connection closed unexpectedly")
			} else {
				slog.Error("Error", "error", err)
			}
		}
	}
}
