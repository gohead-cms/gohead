package handlers

import (
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// actorFrom returns who makes the request, for content revisions: a user
// (JWT) or an API token (username "token:<name>", set by AuthMiddleware).
func actorFrom(c *gin.Context) storage.Actor {
	name := c.GetString("username")
	if tok, ok := strings.CutPrefix(name, "token:"); ok {
		return storage.Actor{Kind: models.ActorToken, Name: tok}
	}
	if name == "" {
		return storage.SystemActor
	}
	return storage.Actor{Kind: models.ActorUser, Name: name}
}
