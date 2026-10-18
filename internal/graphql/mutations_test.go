package graphql

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

// setupTestCollection creates a mock collection with attributes and a test item.
func setupTestCollection(t *testing.T, db *gorm.DB) models.Collection {
	collection := models.Collection{
		Name: "TestCollection",
		Attributes: []models.Attribute{
			{Name: "title", Type: "text"},
			{Name: "age", Type: "int"},
		},
	}

	err := db.Create(&collection).Error
	if err != nil {
		t.Fatalf("Failed to create test collection: %v", err)
	}

	// Insert a test item into the collection
	testItem := models.Item{
		CollectionID: collection.ID,
		Data: models.JSONMap{
			"title": "Mr. John Doe",
			"age":   30,
		},
	}

	err = db.Create(&testItem).Error
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	return collection
}

func runGQL(t *testing.T, role, query string) *graphql.Result {
	t.Helper()
	return graphql.Do(graphql.Params{
		Schema:        GetSchema(),
		RequestString: query,
		Context:       auth.WithRole(context.Background(), role),
	})
}

func TestGenerateGraphQLMutations(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	require.NoError(t, err)

	setupTestCollection(t, db)
	require.NoError(t, InitializeGraphQLSchema())

	// --- Create ---
	result := runGQL(t, "admin", `
		mutation {
			createTestCollection(input: {title: "Sample Title", age: 25}) {
				id
				title
				age
			}
		}`)
	require.Empty(t, result.Errors)
	created := result.Data.(map[string]any)["createTestCollection"].(map[string]any)
	assert.Equal(t, "Sample Title", created["title"])
	assert.EqualValues(t, 25, created["age"])
	createdID := fmt.Sprint(created["id"])

	var item models.Item
	require.NoError(t, db.First(&item, createdID).Error)
	assert.Equal(t, "Sample Title", item.Data["title"])

	// --- Update (partial: required fields may be omitted) ---
	result = runGQL(t, "editor", fmt.Sprintf(`
		mutation {
			updateTestCollection(id: "%s", input: {title: "Updated Title"}) {
				id
				title
				age
			}
		}`, createdID))
	require.Empty(t, result.Errors)
	updated := result.Data.(map[string]any)["updateTestCollection"].(map[string]any)
	assert.Equal(t, "Updated Title", updated["title"])
	assert.EqualValues(t, 25, updated["age"], "untouched fields are preserved")

	require.NoError(t, db.First(&item, createdID).Error)
	assert.Equal(t, "Updated Title", item.Data["title"])

	// --- Viewers may read but not write ---
	result = runGQL(t, "viewer", `{ TestCollection { id title } }`)
	assert.Empty(t, result.Errors)

	result = runGQL(t, "viewer", `mutation { createTestCollection(input: {title: "nope"}) { id } }`)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "access denied")

	result = runGQL(t, "viewer", fmt.Sprintf(`mutation { deleteTestCollection(id: "%s") }`, createdID))
	require.NotEmpty(t, result.Errors)

	// Unauthenticated context (no role) cannot even read.
	result = runGQL(t, "", `{ TestCollection { id } }`)
	require.NotEmpty(t, result.Errors)

	// --- Validation goes through the same rules as REST ---
	result = runGQL(t, "admin", `mutation { createTestCollection(input: {title: "x", age: "not-a-number"}) { id } }`)
	assert.NotEmpty(t, result.Errors)

	// --- Delete ---
	result = runGQL(t, "admin", fmt.Sprintf(`mutation { deleteTestCollection(id: "%s") }`, createdID))
	require.Empty(t, result.Errors)
	assert.Equal(t, true, result.Data.(map[string]any)["deleteTestCollection"])
	assert.Error(t, db.First(&item, createdID).Error)
}
