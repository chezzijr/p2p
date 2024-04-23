package main

import (
	"flag"
	"log/slog"
	"p2p/internal/common/torrent"
	"p2p/internal/peer"
)

var (
    port = flag.Uint("port", 1234, "port to listen on")
    seedingTorrent = flag.String("seed", "", "torrent file to seed")
    downloadTorrent = flag.String("download", "", "torrent file to download")
)

func main() {
    flag.Parse()

    p, err := peer.NewPeer(uint16(*port))
    if err != nil {
        panic(err)
    }

    if *seedingTorrent != "" {
        filename := *seedingTorrent
        ut, err := torrent.GenerateTorrentFromSingleFile(filename, "http://localhost:1234", 1024)
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
        err = p.Download(dt, "tests")
        if err != nil {
            panic(err)
        }
    }

    slog.Info("Listening", "port", *port)

    err = p.RunServer()
    if err != nil {
        panic(err)
    }
}
