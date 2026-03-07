package cache

import (
	"time"
)

var (
	DefaultTTL = time.Duration(1 * time.Hour)
)

type LeaderboardEntry struct {
	Member any     `json:"member"`
	Score  float64 `json:"score"`
}

type ConsumerHandler struct {
	Consumer string
	Handler  func(message any)
}

type ICache interface {
	Set(key string, value any, expireTime *time.Duration) error
	Get(key string, data any) error
	Delete(key string) error
	Clear() error
	ClearWithPrefix(prefix string) error
	// Leaderboard (Sorted Set) methods
	AddScore(boardKey, member string, score float64) error
	GetTopN(boardKey string, n int64) ([]LeaderboardEntry, error)
	GetRank(boardKey, member string) (rank int64, score float64, err error)
	RemoveMember(boardKey, member string) error
	GetAroundMember(boardKey, member string, radius int64) ([]LeaderboardEntry, error)

	// Stream methods
	Publish(stream string, message any) error
	EnsureGroup(stream string, group string) error
	Subscribe(stream string, group string, handler ConsumerHandler) error
}
