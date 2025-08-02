package ratelimiter

import (
	"time"
)

const (
	privateChatRate = time.Second
	groupChatRate   = 3 * time.Second
	queueSize       = 1000
)
