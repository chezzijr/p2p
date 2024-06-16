package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/chezzijr/p2p/internal/server"
	"github.com/chezzijr/p2p/internal/server/database"
	"go.uber.org/fx"

	_ "github.com/joho/godotenv/autoload"
)

func main() {

	fx.New(
		fx.Provide(database.NewPostgres),
		fx.Provide(database.NewRedis),
		fx.Provide(server.NewTrackerServer),
		fx.Provide(server.NewExplorerServer),
		fx.Provide(server.New),
		fx.Invoke(func(server *server.FiberServer) error {
			server.RegisterFiberRoutes()

			port, _ := strconv.Atoi(os.Getenv("PORT"))
			return server.Listen(fmt.Sprintf(":%d", port))
		}),
	).Run()
}
