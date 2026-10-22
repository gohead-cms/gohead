package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gohead-cms/gohead/internal/demo"
	gqlschema "github.com/gohead-cms/gohead/internal/graphql"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
)

// DemoInfo tells the login page how to sign in to a public demo (404 when
// the instance is not a demo).
func DemoInfo(c *gin.Context) {
	if !demo.Enabled() {
		c.Set("response", "Not a demo")
		c.Set("status", http.StatusNotFound)
		return
	}
	c.Set("response", demo.PublicInfo(time.Now()))
	c.Set("status", http.StatusOK)
}

// DemoStatus returns the demo's settings and its snapshot.
func DemoStatus(c *gin.Context) {
	if !demo.Enabled() {
		c.Set("response", "Not a demo")
		c.Set("status", http.StatusNotFound)
		return
	}
	out := gin.H{"demo": demo.PublicInfo(time.Now())}
	if at, ok := demo.SnapshotTime(); ok {
		out["snapshotAt"] = at
	}
	c.Set("response", out)
	c.Set("status", http.StatusOK)
}

// TakeDemoSnapshot saves the database as it is now: the nightly reset goes
// back to it.
func TakeDemoSnapshot(c *gin.Context) {
	if !demo.Enabled() {
		c.Set("response", "Not a demo: set demo.enabled")
		c.Set("status", http.StatusNotFound)
		return
	}
	// The demo role writes the draft-and-publish collections of the
	// content model as it is now.
	if err := demo.EnsureAccount(); err != nil {
		demoError(c, err)
		return
	}
	at, err := demo.Snapshot(c.Request.Context())
	if err != nil {
		demoError(c, err)
		return
	}
	c.Set("response", gin.H{"snapshotAt": at})
	c.Set("status", http.StatusOK)
}

// ResetDemo restores the snapshot now.
func ResetDemo(c *gin.Context) {
	if !demo.Enabled() {
		c.Set("response", "Not a demo: set demo.enabled")
		c.Set("status", http.StatusNotFound)
		return
	}
	if err := demo.Reset(c.Request.Context(), AfterDemoReset); err != nil {
		demoError(c, err)
		return
	}
	c.Set("response", gin.H{"reset": true})
	c.Set("status", http.StatusOK)
}

// AfterDemoReset rebuilds what depends on the content model.
func AfterDemoReset() {
	if err := gqlschema.InitializeGraphQLSchema(); err != nil {
		logger.Log.WithError(err).Error("Demo reset: could not rebuild the GraphQL schema")
	}
}

func demoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, demo.ErrNotSQLite), errors.Is(err, demo.ErrNoSnapshot), errors.Is(err, demo.ErrSnapshotOutdated):
		c.Set("response", err.Error())
		c.Set("status", http.StatusConflict)
	default:
		logger.Log.WithError(err).Error("Demo snapshot or reset failed")
		c.Set("response", "Failed: "+err.Error())
		c.Set("status", http.StatusInternalServerError)
	}
}

// demoURL shows the demo account only the scheme and host of a webhook's
// or connection's URL: paths and query strings can hold secrets (Slack
// webhook URLs do).
func demoURL[M ~map[string]any](c *gin.Context, m M) M {
	if !demo.IsDemo(c.GetString("role")) {
		return m
	}
	if raw, ok := m["url"].(string); ok {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			m["url"] = u.Scheme + "://" + u.Host + "/…"
		} else {
			m["url"] = "hidden"
		}
	}
	return m
}
