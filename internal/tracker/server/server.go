package server

import (
	"github.com/gofiber/fiber/v2"

	"p2p/internal/tracker/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "p2p",
			AppName:      "p2p",
		}),

		db: database.New(),
	}

	return server
}
