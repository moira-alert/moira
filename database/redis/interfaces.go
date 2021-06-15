package redis

import (
	"context"
	"time"

	goredis "github.com/go-redis/redis/v8"
)

// redisClient is an interface that covers all methods of Client and ClusterClient in redis library
type redisClient interface {
	Set(ctx context.Context, key string, value interface{}, expire time.Duration) *goredis.StatusCmd
	Ping(ctx context.Context) *goredis.StatusCmd
}
