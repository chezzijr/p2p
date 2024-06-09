package database

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

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
    AddOrUpdateTTL(ctx context.Context, key string, value string, ttl time.Duration) error
    GetAll(ctx context.Context, key string) ([]string, error)
    Remove(ctx context.Context, key string, value string) error
    RemoveExpired(ctx context.Context) error
}

type redisConn struct {
	client *redis.Client
}

func NewRedis() (Redis, error) {
	if redisInstance != nil {
		return redisInstance, nil
	}

    rdb := redis.NewClient(&redis.Options{
        Addr:     net.JoinHostPort(redisHost, redisPort),
        Password: redisPassword,
        DB:       0,
    })
	redisInstance = &redisConn{
		client: rdb,
	}

    // run remove expired keys every 5 minutes
    go func() {
        for {
            time.Sleep(5 * time.Minute)
            ctx := context.Background()
            err := redisInstance.RemoveExpired(ctx)
            if err != nil {
                fmt.Println("Error removing expired keys", err)
            }
        }
    }()

	return redisInstance, nil
}

func (r* redisConn) AddOrUpdateTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
    val, err := r.client.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().Add(ttl).Unix()), Member: value}).Result()
    slog.Info("AddOrUpdateTTL", "val", val, "err", err)
	return err
}

func (r* redisConn) GetAll(ctx context.Context, key string) ([]string, error) {
    result, err := r.client.ZRange(ctx, key, 0, -1).Result()
    return result, err
}

func (r* redisConn) Remove(ctx context.Context, key string, value string) error {
    _, err := r.client.ZRem(ctx, key, value).Result()
    return err
}

func (r* redisConn) RemoveExpired(ctx context.Context) error {
    cmds, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
        keys, _, err := r.client.Scan(ctx, 0, "*", 0).Result()
        if err != nil {
            return err
        }
        for _, key := range keys {
            currentTime := time.Now().Unix()
            pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", currentTime))
        }
        return nil
    })
    slog.Info("RemoveExpired", "cmds", cmds, "err", err)
	return err
}
