package graphql

import (
	"bytes"
	"errors"
	"testing"

	"gohead/internal/models"
	"gohead/pkg/logger"
	"gohead/pkg/testutils"

	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// Configure logger to write logs to a buffer for testing
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

// MockStorage is a mock for the storage layer
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GetCollectionByName(name string) (*models.Collection, error) {
	args := m.Called(name)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Collection), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorage) GetItemByID(collectionID, itemID uint) (*models.Item, error) {
	args := m.Called(collectionID, itemID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Item), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestResolveRelation(t *testing.T) {
	// Setup test database
	defer testutils.CleanupTestDB()

	mockStorage := new(MockStorage)

	// Define a test collection
	testCollection := &models.Collection{
		Name: "authors",
		ID:   1,
		Attributes: []models.Attribute{
			{Name: "name", Type: "text", Required: true},
		},
	}

	// Define a related test item
	testItem := &models.Item{
		ID:           10,
		CollectionID: 1,
		Data:         models.JSONMap{"name": "John Doe"},
	}

	// Register mock expectations for collection retrieval
	mockStorage.On("GetCollectionByName", "authors").Return(testCollection, nil)

	// Register mock expectations for item retrieval
	mockStorage.On("GetItemByID", uint(1), uint(10)).Return(testItem, nil)

	t.Run("Resolve one-to-one relation", func(t *testing.T) {
		attr := models.Attribute{
			Name:     "author",
			Type:     "relation",
			Target:   "authors",
			Relation: "oneToOne",
		}

		item := models.Item{
			ID:           5,
			CollectionID: 2,
			Data:         models.JSONMap{"author": 10}, // Related ID
		}

		p := graphql.ResolveParams{
			Source: item,
		}

		result, err := ResolveRelation(p, 2, attr)

		assert.NoError(t, err, "Unexpected error in one-to-one relation resolution")
		assert.NotNil(t, result, "Expected result for one-to-one relation should not be nil")
		assert.Equal(t, testItem, result, "Resolved item does not match expected item")
	})

	t.Run("Resolve one-to-many relation", func(t *testing.T) {
		attr := models.Attribute{
			Name:     "authors",
			Type:     "relation",
			Target:   "authors",
			Relation: "oneToMany",
		}

		item := models.Item{
			ID:           5,
			CollectionID: 2,
			Data:         models.JSONMap{"authors": []interface{}{10, 11}}, // Related IDs
		}

		// Define additional related test item
		relatedItem2 := &models.Item{
			ID:           11,
			CollectionID: 1,
			Data:         models.JSONMap{"name": "Jane Smith"},
		}

		// Register mock expectations for multiple item retrievals
		mockStorage.On("GetItemByID", uint(1), uint(11)).Return(relatedItem2, nil)

		p := graphql.ResolveParams{
			Source: item,
		}

		result, err := ResolveRelation(p, 2, attr)

		assert.NoError(t, err, "Unexpected error in one-to-many relation resolution")
		assert.NotNil(t, result, "Expected result for one-to-many relation should not be nil")

		relatedItems, ok := result.([]models.Item)
		assert.True(t, ok, "Expected result to be a slice of related items")
		assert.Len(t, relatedItems, 2, "Expected two related items")
	})

	t.Run("Fail to resolve non-existent relation", func(t *testing.T) {
		attr := models.Attribute{
			Name:     "author",
			Type:     "relation",
			Target:   "authors",
			Relation: "oneToOne",
		}

		item := models.Item{
			ID:           5,
			CollectionID: 2,
			Data:         models.JSONMap{"author": 999}, // Non-existent ID
		}

		mockStorage.On("GetItemByID", uint(1), uint(999)).Return(nil, errors.New("item not found"))

		p := graphql.ResolveParams{
			Source: item,
		}

		result, err := ResolveRelation(p, 2, attr)

		assert.Error(t, err, "Expected error when resolving a non-existent relation")
		assert.Nil(t, result, "Result should be nil when relation cannot be resolved")
	})

	t.Run("Fail to resolve with missing data", func(t *testing.T) {
		attr := models.Attribute{
			Name:     "author",
			Type:     "relation",
			Target:   "authors",
			Relation: "oneToOne",
		}

		item := models.Item{
			ID:           5,
			CollectionID: 2,
			Data:         models.JSONMap{}, // No relation data
		}

		p := graphql.ResolveParams{
			Source: item,
		}

		result, err := ResolveRelation(p, 2, attr)

		assert.NoError(t, err, "Expected no error when relation data is missing")
		assert.Nil(t, result, "Result should be nil when relation data is missing")
	})

	mockStorage.AssertExpectations(t)
}
