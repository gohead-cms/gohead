package middleware

import (
	"net/http"
	"time"

	"github.com/gohead-cms/gohead/internal/demo"
	"github.com/gohead-cms/gohead/pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

// Limits of the demo account, per visitor IP: it is shared by everyone.
const (
	demoWritesPerHour = 60
	demoTestsPerHour  = 10
	demoMaxBody       = 16 << 10
	demoMaxTestBody   = 8 << 10
)

var demoCounter ratelimit.Counter

// SetDemoLimiter sets the counter of the demo account's rate limits (nil:
// none).
func SetDemoLimiter(c ratelimit.Counter) { demoCounter = c }

// DemoGuard applies to the demo account only: small request bodies, writes
// and test runs rate-limited per visitor, and no email address in what it
// reads (forms of a demo's public site collect some).
func DemoGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !demo.IsDemo(c.GetString("role")) {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			limit, name, maxBody := demoWritesPerHour, "demo-write", int64(demoMaxBody)
			if c.FullPath() == "/admin/agents/:name/test" {
				limit, name, maxBody = demoTestsPerHour, "demo-test", demoMaxTestBody
			}
			if c.Request.ContentLength > maxBody {
				abortWithError(c, http.StatusRequestEntityTooLarge, "DemoError", "This is a demo: requests are limited in size.")
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
			if !RateLimitAllow(demoCounter, name, limit, time.Hour)(c) {
				return
			}
		}
		c.Next()
		if v, ok := c.Get("response"); ok {
			c.Set("response", demo.RedactEmails(v))
		}
	}
}
