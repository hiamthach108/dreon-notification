package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockLogger is a simple mock implementation of logger.ILogger
type MockLogger struct {
	errors []string
}

func (m *MockLogger) Debug(msg string, fields ...any) {}
func (m *MockLogger) Info(msg string, fields ...any)  {}
func (m *MockLogger) Warn(msg string, fields ...any)  {}
func (m *MockLogger) Error(msg string, fields ...any) {
	if m.errors != nil {
		m.errors = append(m.errors, msg)
	}
}
func (m *MockLogger) Fatal(msg string, fields ...any) {}
func (m *MockLogger) With(fields ...any) logger.ILogger {
	return m
}
func (m *MockLogger) GetZapLogger() *zap.Logger {
	return nil
}

// Test helper to create a test cache instance
func createTestCache() *appCache {
	mockLogger := &MockLogger{}

	// Create a test cache with mock dependencies
	cache := &appCache{
		serviceName: "test-service",
		logger:      mockLogger,
		redisClient: nil, // We'll set this in individual tests
	}

	return cache
}

func TestNewAppCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis connection test")
	}

	tests := []struct {
		name    string
		config  *config.AppConfig
		logger  logger.ILogger
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.AppConfig{
				App: struct {
					Name    string `env:"APP_NAME"`
					Version string `env:"APP_VERSION"`
				}{
					Name: "test-service",
				},
				Cache: struct {
					DefaultExpireTimeSec int    `env:"CACHE_DEFAULT_EXPIRE_TIME_SEC"`
					CleanupIntervalHour  int    `env:"CACHE_CLEANUP_INTERVAL_HOUR"`
					RedisHost            string `env:"REDIS_HOST"`
					RedisPort            string `env:"REDIS_PORT"`
					RedisPassword        string `env:"REDIS_PASSWORD"`
					RedisDB              int    `env:"REDIS_DB"`
				}{
					RedisHost:     "localhost",
					RedisPort:     "6379",
					RedisPassword: "",
					RedisDB:       1, // Use test DB
				},
			},
			logger:  &MockLogger{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewAppCache(tt.config, tt.logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cache)
			} else {
				// Redis might not be running in CI/CD
				if err != nil {
					t.Skip("Redis not available, skipping test")
				}
				assert.NoError(t, err)
				assert.NotNil(t, cache)
				assert.IsType(t, &appCache{}, cache)
			}
		})
	}
}

func TestAppCache_prefixedKey(t *testing.T) {
	cache := createTestCache()

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "simple key",
			key:      "test-key",
			expected: "test-service:test-key",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "test-service:",
		},
		{
			name:     "key with special characters",
			key:      "test:key:with:colons",
			expected: "test-service:test:key:with:colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.prefixedKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test with real Redis connection (integration test)
func TestAppCache_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create a real Redis client for integration testing
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use a different DB for testing
	})

	// Test connection
	ctx := context.Background()
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	// Clean up test database
	defer redisClient.FlushDB(ctx)

	// Create cache with real Redis client
	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("Set and Get string", func(t *testing.T) {
		key := "test-key"
		value := "test-value"
		expireTime := 5 * time.Minute

		// Test Set
		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		// Test Get
		var result string
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get struct", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}

		key := "test-struct-key"
		value := TestStruct{Name: "test", Count: 42}
		expireTime := 5 * time.Minute

		// Test Set
		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		// Test Get
		var result TestStruct
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value.Name, result.Name)
		assert.Equal(t, value.Count, result.Count)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "test-delete-key"
		value := "test-value"
		expireTime := 5 * time.Minute

		// Set a value first
		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		// Verify it exists
		var result string
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// Delete it
		err = cache.Delete(key)
		assert.NoError(t, err)

		// Verify it's gone
		var deleted string
		err = cache.Get(key, &deleted)
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("ClearWithPrefix", func(t *testing.T) {
		prefix := "test-prefix"
		expireTime := 5 * time.Minute

		// Set multiple keys with prefix
		keys := []string{"test-prefix:1", "test-prefix:2", "other-key"}
		for _, key := range keys {
			err := cache.Set(key, "value", &expireTime)
			assert.NoError(t, err)
		}

		// Clear with prefix
		err := cache.ClearWithPrefix(prefix)
		assert.NoError(t, err)

		// Verify prefixed keys are gone
		for _, key := range []string{"test-prefix:1", "test-prefix:2"} {
			var result string
			err := cache.Get(key, &result)
			assert.Error(t, err)
			assert.Equal(t, redis.Nil, err)
		}

		// Verify other key still exists
		var result string
		err = cache.Get("other-key", &result)
		assert.NoError(t, err)
		assert.Equal(t, "value", result)
	})

	t.Run("AddScore and GetTopN", func(t *testing.T) {
		boardKey := "test-leaderboard"

		// Add scores
		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)

		err = cache.AddScore(boardKey, "player2", 200.0)
		assert.NoError(t, err)

		err = cache.AddScore(boardKey, "player3", 150.0)
		assert.NoError(t, err)

		// Get top 3
		topN, err := cache.GetTopN(boardKey, 3)
		assert.NoError(t, err)
		assert.Len(t, topN, 3)

		// Verify order (should be descending by score)
		assert.Equal(t, "player2", topN[0].Member)
		assert.Equal(t, 200.0, topN[0].Score)
		assert.Equal(t, "player3", topN[1].Member)
		assert.Equal(t, 150.0, topN[1].Score)
		assert.Equal(t, "player1", topN[2].Member)
		assert.Equal(t, 100.0, topN[2].Score)
	})

	t.Run("GetRank", func(t *testing.T) {
		boardKey := "test-leaderboard"

		// Get rank for player2 (should be 1st)
		rank, score, err := cache.GetRank(boardKey, "player2")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rank)
		assert.Equal(t, 200.0, score)

		// Get rank for player1 (should be 3rd)
		rank, score, err = cache.GetRank(boardKey, "player1")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), rank)
		assert.Equal(t, 100.0, score)
	})

	t.Run("RemoveMember", func(t *testing.T) {
		boardKey := "test-leaderboard"

		// Remove player2
		err := cache.RemoveMember(boardKey, "player2")
		assert.NoError(t, err)

		// Verify player2 is gone
		_, _, err = cache.GetRank(boardKey, "player2")
		assert.Error(t, err)

		// Verify other players are still there
		rank, _, err := cache.GetRank(boardKey, "player1")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), rank) // Should now be 2nd instead of 3rd
	})

	t.Run("GetAroundMember", func(t *testing.T) {
		boardKey := "test-leaderboard"

		// Get around player3 with radius 1
		around, err := cache.GetAroundMember(boardKey, "player3", 1)
		assert.NoError(t, err)
		assert.Len(t, around, 2) // player3 and player1

		// Verify the order
		assert.Equal(t, "player3", around[0].Member)
		assert.Equal(t, "player1", around[1].Member)
	})

	t.Run("Clear", func(t *testing.T) {
		// Clear all data
		err := cache.Clear()
		assert.NoError(t, err)

		// Verify everything is gone
		var result string
		err = cache.Get("other-key", &result)
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})
}

// Test error cases with mock Redis client
func TestAppCache_ErrorCases(t *testing.T) {
	t.Run("Set error", func(t *testing.T) {
		// This would require a more sophisticated mock
		// For now, we'll test the integration test covers the happy path
		t.Skip("Requires sophisticated Redis mocking")
	})
}

// MockRedisClient for error testing (simplified)
type MockRedisClient struct{}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "SET", key, value, "EX", int(expiration.Seconds()))
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "GET", key)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := make([]interface{}, len(keys)+1)
	args[0] = "DEL"
	for i, key := range keys {
		args[i+1] = key
	}
	cmd := redis.NewIntCmd(ctx, args...)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) FlushAll(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "FLUSHALL")
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx, "KEYS", pattern)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "ZADD", key, members)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	cmd := redis.NewZSliceCmd(ctx, "ZREVRANGE", key, start, stop, "WITHSCORES")
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) ZRevRank(ctx context.Context, key, member string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "ZREVRANK", key, member)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) ZScore(ctx context.Context, key, member string) *redis.FloatCmd {
	cmd := redis.NewFloatCmd(ctx, "ZSCORE", key, member)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

func (m *MockRedisClient) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	args := make([]interface{}, len(members)+2)
	args[0] = "ZREM"
	args[1] = key
	for i, member := range members {
		args[i+2] = member
	}
	cmd := redis.NewIntCmd(ctx, args...)
	cmd.SetErr(errors.New("mock redis error"))
	return cmd
}

// =============================
// 🔹 Additional Comprehensive Tests
// =============================

func TestAppCache_Set_DifferentTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("Set and Get int", func(t *testing.T) {
		key := "test-int"
		value := 42
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result int
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get int64", func(t *testing.T) {
		key := "test-int64"
		value := int64(9223372036854775807)
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result int64
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get float64", func(t *testing.T) {
		key := "test-float"
		value := 3.14159
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result float64
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get bool", func(t *testing.T) {
		key := "test-bool"
		value := true
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result bool
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get slice", func(t *testing.T) {
		key := "test-slice"
		value := []string{"apple", "banana", "cherry"}
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result []string
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get map", func(t *testing.T) {
		key := "test-map"
		value := map[string]int{"a": 1, "b": 2, "c": 3}
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result map[string]int
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Set and Get nested struct", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}
		type User struct {
			Name    string  `json:"name"`
			Age     int     `json:"age"`
			Address Address `json:"address"`
		}

		key := "test-nested-struct"
		value := User{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}
		expireTime := 5 * time.Minute

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		var result User
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value.Name, result.Name)
		assert.Equal(t, value.Age, result.Age)
		assert.Equal(t, value.Address.Street, result.Address.Street)
		assert.Equal(t, value.Address.City, result.Address.City)
	})
}

func TestAppCache_Get_NonExistentKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	var result string
	err := cache.Get("non-existent-key", &result)
	assert.Error(t, err)
	assert.Equal(t, redis.Nil, err)
}

func TestAppCache_Leaderboard_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("Update existing member score", func(t *testing.T) {
		boardKey := "test-update-score"

		// Add initial score
		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)

		// Update score
		err = cache.AddScore(boardKey, "player1", 200.0)
		assert.NoError(t, err)

		// Verify updated score
		rank, score, err := cache.GetRank(boardKey, "player1")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rank)
		assert.Equal(t, 200.0, score)
	})

	t.Run("GetTopN with more than available", func(t *testing.T) {
		boardKey := "test-topn-limit"

		// Add only 2 players
		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)
		err = cache.AddScore(boardKey, "player2", 200.0)
		assert.NoError(t, err)

		// Request top 10 (more than available)
		topN, err := cache.GetTopN(boardKey, 10)
		assert.NoError(t, err)
		assert.Len(t, topN, 2)
	})

	t.Run("GetTopN with zero", func(t *testing.T) {
		boardKey := "test-topn-zero"

		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)

		topN, err := cache.GetTopN(boardKey, 0)
		assert.NoError(t, err)
		assert.Len(t, topN, 0)
	})

	t.Run("GetRank for non-existent member", func(t *testing.T) {
		boardKey := "test-rank-missing"

		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)

		_, _, err = cache.GetRank(boardKey, "non-existent-player")
		assert.Error(t, err)
	})

	t.Run("RemoveMember non-existent", func(t *testing.T) {
		boardKey := "test-remove-missing"

		err := cache.RemoveMember(boardKey, "non-existent-player")
		assert.NoError(t, err) // Redis doesn't error on removing non-existent members
	})

	t.Run("GetAroundMember at boundary", func(t *testing.T) {
		boardKey := "test-around-boundary"

		// Add players
		for i := 1; i <= 10; i++ {
			err := cache.AddScore(boardKey, string(rune('A'+i-1)), float64(i*10))
			assert.NoError(t, err)
		}

		// Get around top player with large radius
		around, err := cache.GetAroundMember(boardKey, "J", 100)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(around), 10)
		assert.Equal(t, "J", around[0].Member)
	})

	t.Run("GetAroundMember at bottom", func(t *testing.T) {
		boardKey := "test-around-bottom"

		// Get around bottom player
		around, err := cache.GetAroundMember(boardKey, "A", 2)
		assert.NoError(t, err)
		assert.Greater(t, len(around), 0)
	})

	t.Run("Same score different members", func(t *testing.T) {
		boardKey := "test-same-score"

		// Add multiple players with same score
		err := cache.AddScore(boardKey, "player1", 100.0)
		assert.NoError(t, err)
		err = cache.AddScore(boardKey, "player2", 100.0)
		assert.NoError(t, err)
		err = cache.AddScore(boardKey, "player3", 100.0)
		assert.NoError(t, err)

		topN, err := cache.GetTopN(boardKey, 3)
		assert.NoError(t, err)
		assert.Len(t, topN, 3)

		// All should have score 100
		for _, entry := range topN {
			assert.Equal(t, 100.0, entry.Score)
		}
	})

	t.Run("Negative scores", func(t *testing.T) {
		boardKey := "test-negative-score"

		err := cache.AddScore(boardKey, "player1", -100.0)
		assert.NoError(t, err)
		err = cache.AddScore(boardKey, "player2", 50.0)
		assert.NoError(t, err)

		topN, err := cache.GetTopN(boardKey, 2)
		assert.NoError(t, err)
		assert.Len(t, topN, 2)

		// player2 should be first (higher score)
		assert.Equal(t, "player2", topN[0].Member)
		assert.Equal(t, 50.0, topN[0].Score)
		assert.Equal(t, "player1", topN[1].Member)
		assert.Equal(t, -100.0, topN[1].Score)
	})

	t.Run("Float precision scores", func(t *testing.T) {
		boardKey := "test-float-precision"

		err := cache.AddScore(boardKey, "player1", 123.456789)
		assert.NoError(t, err)

		rank, score, err := cache.GetRank(boardKey, "player1")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rank)
		assert.InDelta(t, 123.456789, score, 0.000001)
	})
}

func TestAppCache_ClearWithPrefix_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("ClearWithPrefix no matches", func(t *testing.T) {
		expireTime := 5 * time.Minute
		err := cache.Set("other-key", "value", &expireTime)
		assert.NoError(t, err)

		err = cache.ClearWithPrefix("non-existent-prefix")
		assert.NoError(t, err)

		// Verify other key still exists
		var result string
		err = cache.Get("other-key", &result)
		assert.NoError(t, err)
	})

	t.Run("ClearWithPrefix empty prefix", func(t *testing.T) {
		expireTime := 5 * time.Minute
		err := cache.Set("key1", "value1", &expireTime)
		assert.NoError(t, err)
		err = cache.Set("key2", "value2", &expireTime)
		assert.NoError(t, err)

		// Empty prefix with * should match all keys with service prefix
		err = cache.ClearWithPrefix("")
		assert.NoError(t, err)

		// All keys should be cleared
		var result string
		err = cache.Get("key1", &result)
		assert.Error(t, err)
		err = cache.Get("key2", &result)
		assert.Error(t, err)
	})
}

func TestAppCache_StreamOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("Publish message", func(t *testing.T) {
		stream := "test-stream"
		message := map[string]interface{}{
			"event": "test",
			"data":  "hello world",
		}

		err := cache.Publish(stream, message)
		assert.NoError(t, err)

		// Verify message was added to stream
		rKey := cache.prefixedKey(stream)
		messages, err := redisClient.XRead(ctx, &redis.XReadArgs{
			Streams: []string{rKey, "0"},
			Count:   1,
		}).Result()
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "test", messages[0].Messages[0].Values["event"])
	})

	t.Run("EnsureGroup creates group", func(t *testing.T) {
		stream := "test-stream-group"
		group := "test-group"

		// Publish a message first to create the stream
		err := cache.Publish(stream, map[string]interface{}{"init": "true"})
		assert.NoError(t, err)

		err = cache.EnsureGroup(stream, group)
		assert.NoError(t, err)

		// Calling again should not error
		err = cache.EnsureGroup(stream, group)
		assert.NoError(t, err)
	})

	t.Run("Subscribe and consume messages", func(t *testing.T) {
		stream := "test-stream-subscribe"
		group := "test-group-subscribe"

		// Publish initial message to create stream
		err := cache.Publish(stream, map[string]interface{}{"init": "true"})
		require.NoError(t, err)

		// Create group
		err = cache.EnsureGroup(stream, group)
		require.NoError(t, err)

		// Set up message handler
		messageReceived := make(chan map[string]interface{}, 1)
		handler := ConsumerHandler{
			Consumer: "consumer-1",
			Handler: func(message any) {
				if msg, ok := message.(map[string]interface{}); ok {
					messageReceived <- msg
				}
			},
		}

		// Subscribe
		err = cache.Subscribe(stream, group, handler)
		require.NoError(t, err)

		// Publish a new message
		testMessage := map[string]interface{}{
			"event": "test-event",
			"value": "123",
		}
		err = cache.Publish(stream, testMessage)
		require.NoError(t, err)

		// Wait for message (with timeout)
		select {
		case msg := <-messageReceived:
			assert.Equal(t, "test-event", msg["event"])
			assert.Equal(t, "123", msg["value"])
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for message")
		}
	})

	t.Run("Multiple consumers in same group", func(t *testing.T) {
		stream := "test-stream-multi"
		group := "test-group-multi"

		// Publish initial message to create stream
		err := cache.Publish(stream, map[string]interface{}{"init": "true"})
		require.NoError(t, err)

		// Create group
		err = cache.EnsureGroup(stream, group)
		require.NoError(t, err)

		// Set up two consumers
		consumer1Received := make(chan bool, 1)
		consumer2Received := make(chan bool, 1)

		handler1 := ConsumerHandler{
			Consumer: "consumer-1",
			Handler: func(message any) {
				consumer1Received <- true
			},
		}

		handler2 := ConsumerHandler{
			Consumer: "consumer-2",
			Handler: func(message any) {
				consumer2Received <- true
			},
		}

		// Subscribe both consumers
		err = cache.Subscribe(stream, group, handler1)
		require.NoError(t, err)
		err = cache.Subscribe(stream, group, handler2)
		require.NoError(t, err)

		// Publish messages
		for i := 0; i < 2; i++ {
			err = cache.Publish(stream, map[string]interface{}{"count": i})
			require.NoError(t, err)
		}

		// At least one consumer should receive a message
		// (Redis streams distribute messages among consumers in a group)
		select {
		case <-consumer1Received:
			// OK
		case <-consumer2Received:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for message from any consumer")
		}
	})

	t.Run("Subscribe error handling", func(t *testing.T) {
		mockLogger := &MockLogger{errors: []string{}}
		cacheWithMock := &appCache{
			serviceName: "test-service",
			logger:      mockLogger,
			redisClient: redisClient,
		}

		stream := "non-existent-stream"
		group := "non-existent-group"

		handler := ConsumerHandler{
			Consumer: "consumer-1",
			Handler: func(message any) {
				// This shouldn't be called
			},
		}

		// Subscribe without creating group first
		err := cacheWithMock.Subscribe(stream, group, handler)
		require.NoError(t, err) // Subscribe itself doesn't error

		// Give it a moment to try reading
		time.Sleep(100 * time.Millisecond)

		// Logger should have recorded errors
		assert.Greater(t, len(mockLogger.errors), 0)
	})
}

func TestAppCache_Expiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer redisClient.FlushDB(ctx)

	cache := &appCache{
		serviceName: "test-service",
		logger:      &MockLogger{},
		redisClient: redisClient,
	}

	t.Run("Key expires after TTL", func(t *testing.T) {
		key := "test-expiring-key"
		value := "test-value"
		expireTime := 1 * time.Second

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		// Immediately should exist
		var result string
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// Wait for expiration
		time.Sleep(2 * time.Second)

		// Should be gone
		err = cache.Get(key, &result)
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("Zero expiration means no expiry", func(t *testing.T) {
		key := "test-no-expiry"
		value := "test-value"
		expireTime := 0 * time.Second

		err := cache.Set(key, value, &expireTime)
		assert.NoError(t, err)

		// Should exist immediately
		var result string
		err = cache.Get(key, &result)
		assert.NoError(t, err)

		// Should still exist after waiting
		time.Sleep(1 * time.Second)
		err = cache.Get(key, &result)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})
}

func TestLeaderboardEntry(t *testing.T) {
	t.Run("LeaderboardEntry structure", func(t *testing.T) {
		entry := LeaderboardEntry{
			Member: "player1",
			Score:  100.5,
		}

		assert.Equal(t, "player1", entry.Member)
		assert.Equal(t, 100.5, entry.Score)
	})

	t.Run("LeaderboardEntry with different member types", func(t *testing.T) {
		// String member
		entry1 := LeaderboardEntry{
			Member: "player1",
			Score:  100.0,
		}
		assert.IsType(t, "", entry1.Member)

		// Int member
		entry2 := LeaderboardEntry{
			Member: 12345,
			Score:  200.0,
		}
		assert.IsType(t, 0, entry2.Member)

		// Struct member
		type Player struct {
			ID   int
			Name string
		}
		entry3 := LeaderboardEntry{
			Member: Player{ID: 1, Name: "John"},
			Score:  300.0,
		}
		assert.IsType(t, Player{}, entry3.Member)
	})
}

func TestConsumerHandler(t *testing.T) {
	t.Run("ConsumerHandler structure", func(t *testing.T) {
		called := false
		handler := ConsumerHandler{
			Consumer: "test-consumer",
			Handler: func(message any) {
				called = true
			},
		}

		assert.Equal(t, "test-consumer", handler.Consumer)
		assert.NotNil(t, handler.Handler)

		// Call handler
		handler.Handler(nil)
		assert.True(t, called)
	})
}

func TestDefaultTTL(t *testing.T) {
	t.Run("DefaultTTL is set correctly", func(t *testing.T) {
		assert.Equal(t, time.Duration(1*time.Hour), DefaultTTL)
	})
}
