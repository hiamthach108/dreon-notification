package constant

type ContextKey string

const (
	// Request metadata for session (ip, user agent, referer)
	ContextKeyClientIP  ContextKey = "ip"
	ContextKeyUserAgent ContextKey = "user_agent"
	ContextKeyReferer   ContextKey = "referer"
)
