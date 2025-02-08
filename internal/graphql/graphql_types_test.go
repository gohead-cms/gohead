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

// TestConvertCollectionToGraphQLType ensures the function correctly generates GraphQL types.
func TestConvertCollectionToGraphQLType(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Automigrate the Attribute model
	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Insert a test collection with attributes
	collection := models.Collection{
		Name: "authors",
		Attributes: []models.Attribute{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Required: true, Unique: true},
			{Name: "age", Type: "int"},
			{Name: "isVerified", Type: "bool"},
		},
	}
	if err := database.DB.Create(&collection).Error; err != nil {
		t.Fatalf("Failed to insert test collection: %v", err)
	}

	// Convert collection schema to GraphQL type
	gqlType, err := ConvertCollectionToGraphQLType(collection)
	assert.NoError(t, err)
	assert.NotNil(t, gqlType)

	// Verify fields exist
	assert.NotNil(t, gqlType.Fields()["id"], "Expected 'id' field to exist")
	assert.NotNil(t, gqlType.Fields()["name"], "Expected 'name' field to exist")
	assert.NotNil(t, gqlType.Fields()["email"], "Expected 'email' field to exist")
	assert.NotNil(t, gqlType.Fields()["age"], "Expected 'age' field to exist")
	assert.NotNil(t, gqlType.Fields()["isVerified"], "Expected 'isVerified' field to exist")
}

// TestGetOrCreateGraphQLType ensures GraphQL types are correctly retrieved or generated.
func TestGetOrCreateGraphQLType(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Automigrate the Attribute model
	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// First, create the related "authors" collection
	authorCollection := models.Collection{
		Name: "authors",
		Attributes: []models.Attribute{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Required: true, Unique: true},
		},
	}
	if err := db.Create(&authorCollection).Error; err != nil {
		t.Fatalf("Failed to insert authors collection: %v", err)
	}

	// Insert a test collection with attributes
	collection := models.Collection{
		Name: "posts",
		Attributes: []models.Attribute{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "string"},
			{Name: "author_id", Type: "relation", Target: "authors", Relation: "oneToOne"},
		},
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("Failed to insert test collection: %v", err)
	}

	// Convert collection schema to GraphQL type
	gqlType, err := GetOrCreateGraphQLType("posts")
	assert.NoError(t, err)
	assert.NotNil(t, gqlType)

	// Ensure fields are correctly mapped
	assert.NotNil(t, gqlType.Fields()["id"], "Expected 'id' field to exist")
	assert.NotNil(t, gqlType.Fields()["title"], "Expected 'title' field to exist")
	assert.NotNil(t, gqlType.Fields()["content"], "Expected 'content' field to exist")
	assert.NotNil(t, gqlType.Fields()["author_id"], "Expected 'author_id' field to exist")
	authorField := gqlType.Fields()["author_id"].Type
	_, isObject := authorField.(*graphql.Object)
	assert.True(t, isObject, "Expected 'author_id' to be a GraphQL Object type")
}

// TestTypeRegistryCache ensures that repeated calls return cached types.
func TestTypeRegistryCache(t *testing.T) {
	// Set up the test database
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	// Automigrate the Attribute model
	err := db.AutoMigrate(&models.Attribute{}, &models.Collection{}, &models.Item{}, &models.User{}, &models.UserRole{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	collection := models.Collection{
		Name: "categories",
		Attributes: []models.Attribute{
			{Name: "name", Type: "string", Required: true},
		},
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("Failed to insert test collection: %v", err)
	}

	// First call should create the type
	gqlType1, err := ConvertCollectionToGraphQLType(collection)
	assert.NoError(t, err)

	// Second call should return from cache
	gqlType2, err := ConvertCollectionToGraphQLType(collection)
	assert.NoError(t, err)

	// Ensure they point to the same object (cached)
	assert.Equal(t, gqlType1, gqlType2, "Expected cached GraphQL type to be returned")
}
