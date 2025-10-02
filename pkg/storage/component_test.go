package storage

import (
	"bytes"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- Test Suite Setup ---

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

// setupTestDatabase initializes an in-memory SQLite database for testing.
func setupTestDatabase(t *testing.T) (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to in-memory database: %v", err)
	}
	database.DB = db

	// Apply migrations for the correct, new models
	err = db.AutoMigrate(&models.Component{}, &models.ComponentAttribute{})
	assert.NoError(t, err, "Failed to apply migrations")

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return db, cleanup
}

// clearDB deletes all records to ensure a clean state between tests.
func clearDB(db *gorm.DB) {
	db.Exec("DELETE FROM component_attributes")
	db.Exec("DELETE FROM components")
}

// --- Test Cases ---

func TestCreateComponent(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	seoComponent := &models.Component{
		Name:        "SeoBlock",
		Description: "A reusable block for SEO metadata.",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "meta_title", Type: "text", Required: true}},
			{BaseAttribute: models.BaseAttribute{Name: "meta_description", Type: "text"}},
		},
	}

	err := CreateComponent(seoComponent)
	assert.NoError(t, err)

	// Verification
	var fetched models.Component
	err = db.Preload("Attributes").Where("name = ?", "SeoBlock").First(&fetched).Error
	assert.NoError(t, err)
	assert.Equal(t, "SeoBlock", fetched.Name)
	assert.Len(t, fetched.Attributes, 2)
	assert.Equal(t, "meta_title", fetched.Attributes[0].Name)
	assert.Equal(t, fetched.ID, fetched.Attributes[0].ComponentID)
}

func TestCreateComponent_NameConflict(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	// Pre-populate a component
	CreateComponent(&models.Component{
		Name:       "ExistingHero",
		Attributes: []models.ComponentAttribute{{BaseAttribute: models.BaseAttribute{Name: "title", Type: "text"}}},
	})

	// Attempt to create another with the same name
	err := CreateComponent(&models.Component{
		Name:       "ExistingHero",
		Attributes: []models.ComponentAttribute{{BaseAttribute: models.BaseAttribute{Name: "subtitle", Type: "text"}}},
	})

	assert.Error(t, err)
	assert.Equal(t, "component with name 'ExistingHero' already exists", err.Error())
}

func TestCreateComponent_InvalidSchema(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	component := &models.Component{
		Name: "InvalidComponent",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "", Type: "string"}}, // Invalid: empty name
		},
	}

	err := CreateComponent(component)
	assert.Error(t, err, "Should fail validation due to invalid schema")
}

func TestGetComponentByName(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	CreateComponent(&models.Component{
		Name: "CallToAction",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "button_text", Type: "string"}},
		},
	})

	fetched, err := GetComponentByName("CallToAction")
	assert.NoError(t, err)
	assert.NotNil(t, fetched)
	assert.Equal(t, "CallToAction", fetched.Name)
	assert.Len(t, fetched.Attributes, 1)
}

func TestGetComponentByName_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	_, err := GetComponentByName("NonExistentComponent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateComponent_Attributes(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	CreateComponent(&models.Component{
		Name: "FeatureCard",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "title", Type: "string"}},
			{BaseAttribute: models.BaseAttribute{Name: "icon", Type: "media"}}, // To be removed
		},
	})

	updated := &models.Component{
		Name: "FeatureCard",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "title", Type: "string", Required: true}}, // Update
			{BaseAttribute: models.BaseAttribute{Name: "description", Type: "text"}},             // Add
		},
	}

	err := UpdateComponent("FeatureCard", updated)
	assert.NoError(t, err)

	fetched, _ := GetComponentByName("FeatureCard")
	assert.Len(t, fetched.Attributes, 2)
	attrMap := make(map[string]models.ComponentAttribute)
	for _, attr := range fetched.Attributes {
		attrMap[attr.Name] = attr
	}
	assert.True(t, attrMap["title"].Required)
	assert.NotContains(t, attrMap, "icon")
}

func TestUpdateComponent_Rename(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	CreateComponent(&models.Component{
		Name:       "OldCardName",
		Attributes: []models.ComponentAttribute{{BaseAttribute: models.BaseAttribute{Name: "field", Type: "text"}}},
	})

	updatedComponent := &models.Component{
		Name:       "NewCardName",
		Attributes: []models.ComponentAttribute{{BaseAttribute: models.BaseAttribute{Name: "field", Type: "text"}}},
	}

	err := UpdateComponent("OldCardName", updatedComponent)
	assert.NoError(t, err)

	_, err = GetComponentByName("OldCardName")
	assert.Error(t, err, "Old component name should not be found")
	_, err = GetComponentByName("NewCardName")
	assert.NoError(t, err, "New component name should be found")
}

func TestDeleteComponent(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()
	clearDB(db)

	CreateComponent(&models.Component{
		Name:       "ComponentToDelete",
		Attributes: []models.ComponentAttribute{{BaseAttribute: models.BaseAttribute{Name: "attr1", Type: "string"}}},
	})

	err := DeleteComponent("ComponentToDelete")
	assert.NoError(t, err)

	_, err = GetComponentByName("ComponentToDelete")
	assert.Error(t, err, "Component should not be found after deletion")

	var attrCount int64
	db.Model(&models.ComponentAttribute{}).Count(&attrCount)
	assert.Equal(t, int64(0), attrCount, "Attributes should also be deleted via cascade")
}
