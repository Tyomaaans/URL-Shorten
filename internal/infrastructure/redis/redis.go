package redis

import (
    "context"
    "log"

    "github.com/redis/go-redis/v9"
)

func NewRedisClient(addr, password string) *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       0,
    })

    if err := client.Ping(context.Background()).Err(); err != nil {
        log.Fatalf("failed to connect to redis: %v!", err)
    }
    
    log.Println("Redis Connected On!", addr)
    return client
}