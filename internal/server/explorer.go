package server

import (
	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/server/database"
	"github.com/gofiber/fiber/v2"
)

var (
    explorerServer *explorerSrv
)

type Explorer interface {
    UploadHandler(ctx *fiber.Ctx) error
}

// Explorer is a struct that represents the explorerSrv server
// Stores the metainfo files
type explorerSrv struct {
    pg database.Postgres
}

func NewExplorerServer(pg database.Postgres) Explorer {
    if explorerServer != nil {
        return explorerServer
    }
    explorerServer = &explorerSrv{
        pg: pg,
    }
    return explorerServer
}


func (e *explorerSrv) UploadHandler(ctx *fiber.Ctx) error {
    form, err := ctx.MultipartForm()
    if err != nil {
        return err
    }

    fileHeaders, ok := form.File["metainfo_files"]
    if !ok {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "no file found",
        })
    }

    if len(fileHeaders) == 0 {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "no file found",
        })
    }

    torrents := make([]*torrent.TorrentFile, 0, len(fileHeaders))

    for _, fileHeader := range fileHeaders {
        file, err := fileHeader.Open()
        if err != nil {
            return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Could not open file",
            })
        }
        defer file.Close()

        torrent, err := torrent.OpenFromReader(file)
        if err != nil {
            return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "File is not a valid torrent file",
            })
        }

        torrents = append(torrents, torrent)
    }

    e.pg.AddBulkTorrents(torrents)

    return nil
}


