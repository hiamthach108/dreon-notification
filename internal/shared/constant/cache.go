package constant

import "time"

const (
	// Add cache key prefixes here as needed
	CacheDefaultTTL time.Duration = 1 * time.Hour

	// Cache key prefixes
	CacheKeyPrefixNotification    = "notifications:"
	CacheKeyScheduledEnqueueDedup = "scheduled_enqueue:"
)
