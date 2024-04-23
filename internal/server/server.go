package server

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/chezzijr/p2p/internal/server/database"
    "github.com/chezzijr/p2p/internal/server/tracker"
)

type FiberServer struct {
	*fiber.App

    interval time.Duration
    tracker  *tracker.Tracker

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "p2p",
			AppName:      "p2p",
		}),
        interval: 15 * time.Minute,
        tracker:  tracker.New(),

		db: database.New(),
	}

	return server
}
