package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/testutils"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

func TestGenerateGraphQLMutations(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Automigrate the Attribute model
	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	setupTestCollection(t, db)

	mutation, err := GenerateGraphQLMutations()
	assert.NoError(t, err)
	assert.NotNil(t, mutation)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Mutation: mutation,
	})
	assert.NoError(t, err)

	// --- Test Create Item Mutation ---
	params := graphql.Params{
		Schema: schema,
		RequestString: fmt.Sprintf(`
			mutation {
				createTestCollection(title: "Sample Title", age: 25) {
					id
					title
					age
				}
			}
		`),
	}

	result := graphql.Do(params)
	assert.Empty(t, result.Errors)
	assert.NotNil(t, result.Data)

	data, _ := json.Marshal(result.Data)
	fmt.Println("Create Mutation Result:", string(data))

	// Verify item is stored in DB
	var item models.Item
	err = db.First(&item).Error
	assert.NoError(t, err)
	assert.Equal(t, "Sample Title", item.Data["title"])
	assert.Equal(t, float64(25), item.Data["age"])

	// --- Test Update Item Mutation ---
	params = graphql.Params{
		Schema: schema,
		RequestString: fmt.Sprintf(`
			mutation {
				updateTestCollection(id: "%d", title: "Updated Title") {
					id
					title
					age
				}
			}
		`, item.ID),
	}

	result = graphql.Do(params)
	assert.Empty(t, result.Errors)
	assert.NotNil(t, result.Data)

	data, _ = json.Marshal(result.Data)
	fmt.Println("Update Mutation Result:", string(data))

	// Verify item is updated
	err = db.First(&item).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", item.Data["title"])

	// --- Test Delete Item Mutation ---
	params = graphql.Params{
		Schema: schema,
		RequestString: fmt.Sprintf(`
			mutation {
				deleteTestCollection(id: "%d")
			}
		`, item.ID),
	}

	result = graphql.Do(params)
	assert.Empty(t, result.Errors)
	assert.NotNil(t, result.Data)

	// Verify item is deleted
	err = db.First(&item).Error
	assert.Error(t, err) // Should return record not found
}
