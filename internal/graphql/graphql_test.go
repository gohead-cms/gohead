package graphql

import (
	"bytes"
	"testing"

	"gohead/internal/models"
	"gohead/pkg/database"
	"gohead/pkg/logger"
	"gohead/pkg/testutils"

	"github.com/graphql-go/graphql"
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

// TestGenerateGraphQLQueries ensures that queries are generated correctly
func TestGenerateGraphQLQueries(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Automigrate the Attribute model
	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Insert test collections with attributes
	collection := models.Collection{
		Name: "authors",
		Attributes: []models.Attribute{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Required: true, Unique: true},
		},
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("Failed to insert test collection: %v", err)
	}

	// Call function to generate GraphQL queries
	queryObject, err := GenerateGraphQLQueries()
	assert.NoError(t, err)
	assert.NotNil(t, queryObject)

	// Check if "authors" query is generated
	field := queryObject.Fields()["authors"]
	assert.NotNil(t, field, "Expected 'authors' query to be generated")

	// Ensure the field type is correctly assigned
	//assert.Equal(t, graphql.TypeKindObject(graphql.Object), field.Type.Kind(), "Expected 'authors' type to be an Object")

	// Test executing the query
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryObject,
	})
	assert.NoError(t, err)

	// Create a mock item for testing
	testItem := models.Item{
		CollectionID: collection.ID,
		Data:         models.JSONMap{"name": "John Doe", "email": "john.doe@example.com"},
	}
	if err := database.DB.Create(&testItem).Error; err != nil {
		t.Fatalf("Failed to insert test item: %v", err)
	}

	// Execute a GraphQL query
	query := `
		{
			authors(id: "1") {
				name
				email
			}
		}`
	params := graphql.Params{Schema: schema, RequestString: query}
	result := graphql.Do(params)

	// Validate response
	assert.Empty(t, result.Errors, "Expected no errors in GraphQL query execution")
	assert.NotNil(t, result.Data, "Expected valid data in GraphQL response")
}
