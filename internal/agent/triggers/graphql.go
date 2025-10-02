package triggers

import (
	"github.com/gohead-cms/gohead/internal/graphql"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// TriggerSchemaReload re-initializes the GraphQL schema.
// It's designed to be called in a goroutine so it doesn't block API responses.
func TriggerSchemaReload() {
	logger.Log.Info("🔄 Schema modification detected, triggering hot reload...")
	if err := graphql.InitializeGraphQLSchema(); err != nil {
		logger.Log.WithError(err).Error("Failed to hot reload GraphQL schema")
	}
}
