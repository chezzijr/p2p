package server

import (
	"github.com/gofiber/fiber/v2"
)

type FiberServer struct {
	*fiber.App
	trackerServer  Tracker
	explorerServer Explorer
}

func New(trackerServer Tracker, explorerServer Explorer) *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "p2p",
			AppName:      "p2p",
		}),
		trackerServer:  trackerServer,
		explorerServer: explorerServer,
	}

	return server
}
