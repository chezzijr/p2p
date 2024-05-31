package database

import (
	"context"
	"fmt"
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

	connStr := fmt.Sprintf("redis://%s:%s@%s:%s/%s", redisUsername, redisPassword, redisHost, redisPort, redisDatabase)
	opt, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	redisInstance = &redisConn{
		client: client,
	}

    // run remove expired keys every 5 minutes
    go func() {
        for {
            ctx := context.Background()
            err := redisInstance.RemoveExpired(ctx)
            if err != nil {
                fmt.Println("Error removing expired keys", err)
            }
            time.Sleep(5 * time.Minute)
        }
    }()

	return redisInstance, nil
}

func (r* redisConn) AddOrUpdateTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
	script := redis.NewScript(`
        local key = KEYS[1]
        local element = ARGV[1]
        local ttl = tonumber(ARGV[2])
        local currentTime = tonumber(redis.call('TIME')[1])
        local expireTime = currentTime + ttl
        redis.call('ZADD', key, expireTime, element)
    `)
	_, err := script.Run(ctx, r.client, []string{key}, value, uint64(ttl)).Result()
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
	script := redis.NewScript(`
        local keys = redis.call('KEYS', "*")
        local currentTime = tonumber(redis.call('TIME')[1])
        for i, key in ipairs(keys) do
            redis.call('ZREMRANGEBYSCORE', key, '-inf', currentTime)
        end
   `)
	_, err := script.Run(ctx, redisInstance.client, []string{}).Result()
	return err
}
