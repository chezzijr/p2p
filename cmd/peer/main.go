package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	// "path"

	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/peer"
	"github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name: "chezzijr-p2p",
        Usage: "File sharing application",
        Commands: []*cli.Command{
            {
                Name: "start",
                Usage: "Start the peer",
                Flags: []cli.Flag{
                    &cli.UintFlag{
                        Name: "port",
                        Value: 6881,
                        Aliases: []string{"p"},
                        Usage: "Port to listen on",
                    },
                    &cli.StringFlag{
                        Name: "tracker",
                        Value: "http://localhost:8080/announce",
                        Aliases: []string{"t"},
                        Usage: "Tracker URL",
                    },
                    &cli.StringSliceFlag{
                        Name: "seed",
                        Value: cli.NewStringSlice(),
                        Aliases: []string{"s"},
                        Usage: "Files to seed, from which the metainfo files will be created",
                    },
                    &cli.StringSliceFlag{
                        Name: "leech",
                        Value: cli.NewStringSlice(),
                        Aliases: []string{"l"},
                        Usage: "Torrent files to leech",
                    },
                },
                Action: func(c *cli.Context) error {
                    port := c.Uint("port")
                    trackerUrl := c.String("tracker")
                    seedingFiles := c.StringSlice("seed")
                    leechingFiles := c.StringSlice("leech")

                    p, err := peer.NewPeer(uint16(port))
                    if err != nil {
                        return err
                    }
                    defer p.Close()

                    for _, seedingFile := range seedingFiles {
                        t, err := torrent.GenerateTorrentFromSingleFile(seedingFile, trackerUrl, 1024)
                        if err != nil {
                            return err
                        }

                        torrentFile := t.Name + ".torrent"
                        err = t.Save(torrentFile)
                        if err != nil {
                            return err
                        }

                        p.RegisterEvent(&peer.EventUpload{
                            FilePath: seedingFile,
                            TorrentPath: torrentFile,
                        })
                    }

                    for _, leechingFile := range leechingFiles {
                        p.RegisterEvent(&peer.EventDownload{
                            DownloadPath: "tests",
                            TorrentPath: leechingFile,
                        })
                    }

                    err = p.Run(c.Context)

                    return err
                },
            },
            {
                Name: "list",
                Usage: "List the torrents on the website",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name: "url",
                        Value: "http://localhost:8080/api/list",
                        Usage: "Website URL",
                    },
                },
                Action: func(c *cli.Context) error {
                    url := c.String("url")

                    resp, err := http.Get(url)
                    if err != nil {
                        return err
                    }
                    defer resp.Body.Close()

                    io.Copy(os.Stdout, resp.Body)

                    return nil
                },
            },
        },
    }

    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        signalChan := make(chan os.Signal, 1)
        signal.Notify(signalChan, os.Interrupt)
        <-signalChan
        cancel()
    }()

    if err := app.RunContext(ctx, os.Args); err != nil {
        panic(err)
    }
}
