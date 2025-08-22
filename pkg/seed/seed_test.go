package seed

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func TestSeedRoles(t *testing.T) {
	// Initialize in-memory test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	assert.NoError(t, db.AutoMigrate(&models.UserRole{}))

	logger.Log.Info("Testing role seeding")
	SeedRoles()

	// Verify that roles have been seeded
	var roles []models.UserRole
	err := db.Find(&roles).Error
	assert.NoError(t, err, "Failed to retrieve roles")
	assert.Len(t, roles, 3, "Expected 3 default roles")

	// Check specific roles
	expectedRoles := map[string]string{
		"admin":  "Administrator with full access",
		"editor": "Editor with content management access",
		"viewer": "Viewer with read-only access",
	}

	for _, role := range roles {
		description, exists := expectedRoles[role.Name]
		assert.True(t, exists, "Unexpected role: %s", role.Name)
		assert.Equal(t, description, role.Description)
		logger.Log.Infof("Verified role '%s' with description '%s'", role.Name, role.Description)
	}
}
