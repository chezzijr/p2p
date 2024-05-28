package server

import (
	"strconv"

	"github.com/chezzijr/p2p/internal/common/api"
	"github.com/chezzijr/p2p/internal/common/torrent"
	"github.com/chezzijr/p2p/internal/server/database"
	"github.com/gofiber/fiber/v2"
)

var (
    explorerServer *explorerSrv
)

type Explorer interface {
    UploadHandler(ctx *fiber.Ctx) error
    ListHandler(ctx *fiber.Ctx) error
    DownloadHandler(ctx *fiber.Ctx) error
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

func (e *explorerSrv) ListHandler(ctx *fiber.Ctx) error {
    var req api.ExploreRequest
    if err := ctx.BodyParser(&req); err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    if req.Limit == 0 {
        req.Limit = 10
    }

    torrents, err := e.pg.GetRecentTorrents(req.Offset, req.Limit)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return ctx.JSON(torrents)
}

func (e *explorerSrv) DownloadHandler(ctx *fiber.Ctx) error {
    id := ctx.Params("id")
    if id == "" {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "id is required",
        })
    }

    idInt, err := strconv.Atoi(id)
    if err != nil {
        return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "id must be an integer",
        })
    }

    t, err := e.pg.GetTorrentByID(idInt)
    if err != nil {
        return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    // download torrent file
    ctx.Set(fiber.HeaderContentType, "application/x-bittorrent")
    ctx.Set(fiber.HeaderContentDisposition, "attachment; filename=torrent.torrent")
    return t.Write(ctx)
}
