package middleware

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

// RateLimit allows at most limit requests per client IP and window for the
// routes it is attached to; name separates independent budgets. A limit of
// 0 disables it. Counters are shared across replicas when backed by Redis.
//
// The client IP comes from gin's ClientIP, which only trusts forwarding
// headers from configured trusted proxies (see trusted_proxies).
func RateLimit(counter ratelimit.Counter, name string, limit int, window time.Duration) gin.HandlerFunc {
	allow := RateLimitAllow(counter, name, limit, window)
	return func(c *gin.Context) {
		if allow(c) {
			c.Next()
		}
	}
}

// Limiter counts a request and reports whether it may go on; when it may
// not, it has answered 429. Unlike a gin middleware it does not run the
// rest of the chain, so another middleware can call it inline.
type Limiter func(c *gin.Context) bool

// RateLimitAllow is RateLimit as a Limiter.
func RateLimitAllow(counter ratelimit.Counter, name string, limit int, window time.Duration) Limiter {
	return func(c *gin.Context) bool {
		if limit <= 0 || counter == nil {
			return true
		}
		count, reset, err := counter.Hit(c.Request.Context(), name+":ip:"+c.ClientIP(), window)
		if err != nil {
			// Fail open: never lock everyone out because the limiter broke.
			logger.Log.WithError(err).Warn("Rate limiter error; request allowed")
			return true
		}
		c.Header("RateLimit-Limit", strconv.Itoa(limit))
		c.Header("RateLimit-Remaining", strconv.FormatInt(max(int64(limit)-count, 0), 10))
		if count > int64(limit) {
			TooManyRequests(c, reset, "Too many requests, please retry later")
			return false
		}
		return true
	}
}

// TooManyRequests aborts with 429, a Retry-After header and the standard
// error envelope.
func TooManyRequests(c *gin.Context, retryAfter time.Duration, message string) {
	secs := int(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	abortWithError(c, http.StatusTooManyRequests, "", message)
}
