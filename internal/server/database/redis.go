package database

import (
	"context"
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
    AddOrUpdateTTL(ctx context.Context, key string, value string, ttl uint64) error
    RemoveExpired(ctx context.Context, key string) error
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

func (r* redisConn) AddOrUpdateTTL(ctx context.Context, key string, value string, ttl uint64) error {
	script := redis.NewScript(`
        local key = KEYS[1]
        local element = ARGV[1]
        local ttl = tonumber(ARGV[2])
        local currentTime = tonumber(redis.call('TIME')[1])
        local expireTime = currentTime + ttl
        redis.call('ZADD', key, expireTime, element)
    `)
	_, err := script.Run(ctx, r.client, []string{key}, value, ttl).Result()
	return err
}

func (r* redisConn) RemoveExpired(ctx context.Context, key string) error {
	script := redis.NewScript(`
        local keys = redis.call('KEYS', "*")
        local currentTime = tonumber(redis.call('TIME')[1])
        for i, key in ipairs(keys) do
            redis.call('ZREMRANGEBYSCORE', key, '-inf', currentTime)
        end
   `)
	_, err := script.Run(ctx, redisInstance.client, []string{key}).Result()
	return err
}
