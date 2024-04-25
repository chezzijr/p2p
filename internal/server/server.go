package server

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/chezzijr/p2p/internal/server/database"
)

type FiberServer struct {
	*fiber.App

    interval time.Duration
    tracker  Tracker

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "p2p",
			AppName:      "p2p",
		}),
        interval: 15 * time.Minute,
        tracker:  NewTracker(),

		db: database.New(),
	}

	return server
}
