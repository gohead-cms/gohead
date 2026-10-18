package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/ratelimit"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loginRouter(t *testing.T, guard *LoginAttemptGuard, authLimit gin.HandlerFunc, trusted []string) *gin.Engine {
	t.Helper()
	r, db := testutils.SetupTestServer()
	t.Cleanup(testutils.CleanupTestDB)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	require.NoError(t, r.SetTrustedProxies(trusted))
	auth.InitializeJWT("test-secret-0123456789abcdef0123456789")

	prev := LoginGuard
	LoginGuard = guard
	t.Cleanup(func() { LoginGuard = prev })

	r.Use(middleware.ResponseWrapper())
	g := r.Group("/auth")
	if authLimit != nil {
		g.Use(authLimit)
	}
	g.POST("/register", Register)
	g.POST("/login", Login)
	return r
}

func post(r *gin.Engine, path, ip string, body any, headers ...string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":40000"
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLoginGuard(t *testing.T) {
	guard := &LoginAttemptGuard{Counter: ratelimit.NewMemory(), MaxFailures: 3, Window: 15 * time.Minute}
	r := loginRouter(t, guard, nil, nil)

	require.Equal(t, http.StatusCreated, post(r, "/auth/register", "10.0.0.1",
		map[string]string{"username": "alice", "email": "alice@example.com", "password": "correct-horse"}).Code)

	bad := map[string]string{"username": "alice", "password": "wrong"}
	good := map[string]string{"username": "alice", "password": "correct-horse"}

	for i := 0; i < 3; i++ {
		assert.Equal(t, http.StatusUnauthorized, post(r, "/auth/login", "10.0.0.1", bad).Code)
	}

	// Locked for this client, even with the right password.
	w := post(r, "/auth/login", "10.0.0.1", good)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), "TooManyRequests")

	// Another client can still log in: an attacker cannot lock alice out.
	assert.Equal(t, http.StatusOK, post(r, "/auth/login", "10.0.0.2", good).Code)

	// Case and whitespace do not give extra attempts.
	assert.Equal(t, http.StatusTooManyRequests,
		post(r, "/auth/login", "10.0.0.1", map[string]string{"username": " ALICE ", "password": "x"}).Code)

	// Unknown usernames are counted too, and answer like wrong passwords.
	for i := 0; i < 3; i++ {
		w = post(r, "/auth/login", "10.0.0.3", map[string]string{"username": "nobody", "password": "x"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid username or password")
	}
	assert.Equal(t, http.StatusTooManyRequests,
		post(r, "/auth/login", "10.0.0.3", map[string]string{"username": "nobody", "password": "x"}).Code)
}

func TestLoginGuardResetsOnSuccess(t *testing.T) {
	guard := &LoginAttemptGuard{Counter: ratelimit.NewMemory(), MaxFailures: 3, Window: 15 * time.Minute}
	r := loginRouter(t, guard, nil, nil)
	require.Equal(t, http.StatusCreated, post(r, "/auth/register", "10.0.0.1",
		map[string]string{"username": "bob", "email": "bob@example.com", "password": "correct-horse"}).Code)

	bad := map[string]string{"username": "bob", "password": "wrong"}
	good := map[string]string{"username": "bob", "password": "correct-horse"}
	post(r, "/auth/login", "10.0.0.1", bad)
	post(r, "/auth/login", "10.0.0.1", bad)
	assert.Equal(t, http.StatusOK, post(r, "/auth/login", "10.0.0.1", good).Code)
	// The counter was reset, so two more mistakes are still allowed.
	post(r, "/auth/login", "10.0.0.1", bad)
	post(r, "/auth/login", "10.0.0.1", bad)
	assert.Equal(t, http.StatusOK, post(r, "/auth/login", "10.0.0.1", good).Code)
}

func TestAuthRateLimitPerIP(t *testing.T) {
	limit := middleware.RateLimit(ratelimit.NewMemory(), "auth", 2, time.Minute)
	r := loginRouter(t, nil, limit, nil)
	body := map[string]string{"username": "x", "password": "y"}

	assert.Equal(t, http.StatusUnauthorized, post(r, "/auth/login", "10.0.0.1", body).Code)
	w := post(r, "/auth/login", "10.0.0.1", body)
	assert.Equal(t, "0", w.Header().Get("RateLimit-Remaining"))
	w = post(r, "/auth/login", "10.0.0.1", body)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))

	// Registration shares the /auth budget for that IP.
	assert.Equal(t, http.StatusTooManyRequests, post(r, "/auth/register", "10.0.0.1",
		map[string]string{"username": "z", "email": "z@example.com", "password": "123456"}).Code)

	// Other clients are unaffected.
	assert.Equal(t, http.StatusUnauthorized, post(r, "/auth/login", "10.0.0.9", body).Code)
}

func TestForwardedForOnlyFromTrustedProxies(t *testing.T) {
	body := map[string]string{"username": "x", "password": "y"}

	// No trusted proxies: X-Forwarded-For is ignored, so rotating it does
	// not escape the limit.
	r := loginRouter(t, nil, middleware.RateLimit(ratelimit.NewMemory(), "auth", 1, time.Minute), nil)
	post(r, "/auth/login", "10.0.0.1", body, "X-Forwarded-For", "1.1.1.1")
	assert.Equal(t, http.StatusTooManyRequests,
		post(r, "/auth/login", "10.0.0.1", body, "X-Forwarded-For", "2.2.2.2").Code)

	// Behind a trusted ingress, each real client has its own budget.
	r = loginRouter(t, nil, middleware.RateLimit(ratelimit.NewMemory(), "auth", 1, time.Minute), []string{"10.0.0.0/8"})
	assert.Equal(t, http.StatusUnauthorized, post(r, "/auth/login", "10.0.0.1", body, "X-Forwarded-For", "1.1.1.1").Code)
	assert.Equal(t, http.StatusUnauthorized, post(r, "/auth/login", "10.0.0.1", body, "X-Forwarded-For", "2.2.2.2").Code)
	assert.Equal(t, http.StatusTooManyRequests, post(r, "/auth/login", "10.0.0.1", body, "X-Forwarded-For", "1.1.1.1").Code)
}
