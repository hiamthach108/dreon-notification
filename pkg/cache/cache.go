package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	ErrCacheNil = redis.Nil
)

type appCache struct {
	serviceName string
	logger      logger.ILogger
	redisClient *redis.Client
}

func NewAppCache(config *config.AppConfig, logger logger.ILogger) (ICache, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Cache.RedisHost + ":" + config.Cache.RedisPort,
		Password: config.Cache.RedisPassword,
		DB:       config.Cache.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis successfully")

	return &appCache{
		serviceName: config.App.Name,
		logger:      logger,
		redisClient: redisClient,
	}, nil
}

// =============================
// 🔹 Basic Cache Operations
// =============================

func (c *appCache) Set(key string, value any, expireTime *time.Duration) error {
	rKey := c.prefixedKey(key)

	// Serialize value to JSON for complex types
	var data any
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		// Primitive types can be stored directly
		data = v
	default:
		// Serialize complex types to JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		data = jsonData
	}

	return c.redisClient.Set(context.Background(), rKey, data, *expireTime).Err()
}

func (c *appCache) Get(key string, data any) error {
	rKey := c.prefixedKey(key)
	val, err := c.redisClient.Get(context.Background(), rKey).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), data); err != nil {
		return err
	}

	return nil
}

func (c *appCache) Delete(key string) error {
	rKey := c.prefixedKey(key)
	return c.redisClient.Del(context.Background(), rKey).Err()
}

func (c *appCache) Clear() error {
	return c.redisClient.FlushAll(context.Background()).Err()
}

func (c *appCache) ClearWithPrefix(prefix string) error {
	ctx := context.Background()
	pattern := c.prefixedKey(fmt.Sprintf("%s*", prefix))
	keys, err := c.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.redisClient.Del(ctx, keys...).Err()
	}
	return nil
}

// =============================
// 🔹 Leaderboard (Sorted Set)
// =============================

// AddScore adds or updates a member’s score in a leaderboard.
func (c *appCache) AddScore(boardKey, member string, score float64) error {
	rKey := c.prefixedKey(boardKey)
	return c.redisClient.ZAdd(context.Background(), rKey, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// GetTopN retrieves top N members with their scores in descending order.
func (c *appCache) GetTopN(boardKey string, n int64) ([]LeaderboardEntry, error) {
	rKey := c.prefixedKey(boardKey)
	zResult, err := c.redisClient.ZRevRangeWithScores(context.Background(), rKey, 0, n-1).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(zResult))
	for i, z := range zResult {
		entries[i] = LeaderboardEntry{
			Member: z.Member,
			Score:  z.Score,
		}
	}

	return entries, nil
}

// GetRank retrieves the rank (1-based) and score of a specific member.
func (c *appCache) GetRank(boardKey, member string) (rank int64, score float64, err error) {
	rKey := c.prefixedKey(boardKey)
	rank, err = c.redisClient.ZRevRank(context.Background(), rKey, member).Result()
	if err != nil {
		return 0, 0, err
	}

	score, err = c.redisClient.ZScore(context.Background(), rKey, member).Result()
	if err != nil {
		return 0, 0, err
	}

	return rank + 1, score, nil // rank is 0-based in Redis
}

// RemoveMember removes a player from the leaderboard.
func (c *appCache) RemoveMember(boardKey, member string) error {
	rKey := c.prefixedKey(boardKey)
	return c.redisClient.ZRem(context.Background(), rKey, member).Err()
}

// GetAroundMember gets a window of players around a given member (for user’s local rank view)
func (c *appCache) GetAroundMember(boardKey, member string, radius int64) ([]LeaderboardEntry, error) {
	rKey := c.prefixedKey(boardKey)
	rank, err := c.redisClient.ZRevRank(context.Background(), rKey, member).Result()
	if err != nil {
		return nil, err
	}

	start := rank - radius
	if start < 0 {
		start = 0
	}
	end := rank + radius

	zResult, err := c.redisClient.ZRevRangeWithScores(context.Background(), rKey, start, end).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, len(zResult))
	for i, z := range zResult {
		entries[i] = LeaderboardEntry{
			Member: z.Member,
			Score:  z.Score,
		}
	}

	return entries, nil
}

// =============================
// 🔹 Stream Operations
// =============================

func (c *appCache) Publish(stream string, message any) error {
	rKey := c.prefixedKey(stream)

	// Encode to binary using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(message); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Store as binary data field
	return c.redisClient.XAdd(context.Background(), &redis.XAddArgs{
		Stream: rKey,
		Values: map[string]any{
			"data": buf.Bytes(),
		},
	}).Err()
}

func (c *appCache) EnsureGroup(stream, group string) error {
	rKey := c.prefixedKey(stream)

	err := c.redisClient.
		XGroupCreateMkStream(context.Background(), rKey, group, "$").
		Err()

	// If group already exists → ignore
	if err != nil {
		if strings.Contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return err
	}

	return nil
}

func (c *appCache) Subscribe(stream string, group string, handler ConsumerHandler) error {
	rKey := c.prefixedKey(stream)
	go func() {
		for {
			streams, err := c.redisClient.XReadGroup(context.Background(), &redis.XReadGroupArgs{
				Group:    group,
				Consumer: handler.Consumer,
				Streams:  []string{rKey, ">"},
				Count:    1,
				Block:    0,
			}).Result()
			if err != nil {
				c.logger.Error("Failed to read from stream", "stream", stream, "group", group, "error", err)
				continue
			}

			for _, strm := range streams {
				for _, message := range strm.Messages {
					handler.Handler(message.Values)
					// Acknowledge message
					c.redisClient.XAck(context.Background(), rKey, group, message.ID)
				}
			}
		}
	}()

	return nil
}

func (c *appCache) prefixedKey(key string) string {
	return fmt.Sprintf("%s:%s", c.serviceName, key)
}
