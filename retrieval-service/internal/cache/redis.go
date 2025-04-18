package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"retrieval-service/internal/domain/paste"
	"retrieval-service/shared"
)

type PasteCache interface {
	Get(url string) (*paste.Paste, error)
	Set(p *paste.Paste) error
	Delete(url string) error
}

// RedisPasteCache implements PasteCache with Redis
type RedisPasteCache struct {
	client *redis.Client
}

// NewRedisPasteCache creates a new RedisPasteCache
func NewRedisPasteCache(client *redis.Client) *RedisPasteCache {
	return &RedisPasteCache{client: client}
}

// NewRedisClient creates a Redis client from URI
func NewRedisClient(uri string) (*redis.Client, error) {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	_, err = client.Ping(context.Background()).Result()
	return client, err
}

func (c *RedisPasteCache) Get(url string) (*paste.Paste, error) {
	ctx := context.Background()
	val, err := c.client.Get(ctx, "paste:"+url).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}

	var p paste.Paste
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *RedisPasteCache) Set(p *paste.Paste) error {
	ctx := context.Background()
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	ttl := 24 * time.Hour
	if p.ExpirationPolicy.Type == paste.TimedExpiration {
		if duration, ok := shared.DurationMap[p.ExpirationPolicy.Duration]; ok {
			remaining := duration - time.Since(p.CreatedAt)
			if remaining > 0 {
				ttl = remaining
			}
		}
	}

	return c.client.Set(ctx, "paste:"+p.URL, data, ttl).Err()
}

func (c *RedisPasteCache) Delete(url string) error {
	ctx := context.Background()
	return c.client.Del(ctx, "paste:"+url).Err()
}
