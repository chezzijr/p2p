package database

import (
	"fmt"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/redis/go-redis/v9"
)

var (
	redisHost     = os.Getenv("REDIS_HOST")
	redisPort     = os.Getenv("REDIS_PORT")
	redisPassword = os.Getenv("REDIS_PASSWORD")
	redisDatabase = os.Getenv("REDIS_DATABASE")
	redisUsername = os.Getenv("REDIS_USERNAME")
	redisInstance *redisConn
)

type Redis interface {
}

type redisConn struct {
    client *redis.Client
}

func NewRedis() (Redis, error) {
	if redisInstance != nil {
		return redisInstance, nil
	}

    connStr := fmt.Sprintf("redis://%s:%s@%s:%s/%s", redisUsername, redisPassword, redisHost, redisPort, redisDatabase)
    opt, err := redis.ParseURL(connStr)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)
    redisInstance = &redisConn{
        client: client,
    }

	return redisInstance, nil
}
