// models/attribute_test.go
package models

import (
	"bytes"
	"encoding/json"
	"testing"

	"gohead/pkg/database"
	"gohead/pkg/logger"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Logger init for tests
func init() {
	var buffer bytes.Buffer
	logger.InitLogger("debug")
	logger.Log.SetOutput(&buffer)
	logger.Log.SetFormatter(&logrus.TextFormatter{})
}

func setupDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to in-memory database")
	database.DB = db
	err = db.AutoMigrate(&Collection{}, &Item{}, &SingleType{}, &SingleItem{}, &Attribute{})
	require.NoError(t, err, "failed to auto-migrate schema")
	return db
}

func TestAttributeModelInitialization(t *testing.T) {
	id := uint(1)
	collectionID := &id
	attr := Attribute{
		Name:         "Age",
		Type:         "int",
		Required:     true,
		Unique:       false,
		Options:      []string{"18", "25", "30"},
		Min:          intPtr(18),
		Max:          intPtr(99),
		Pattern:      "^\\d+$",
		CustomErrors: JSONMap{"required": "Age is required."},
		Target:       "User",
		Relation:     "oneToOne",
		CollectionID: collectionID,
	}

	assert.Equal(t, "Age", attr.Name)
	assert.Equal(t, "int", attr.Type)
	assert.True(t, attr.Required)
	assert.False(t, attr.Unique)
	assert.Equal(t, []string{"18", "25", "30"}, attr.Options)
	assert.Equal(t, 18, *attr.Min)
	assert.Equal(t, 99, *attr.Max)
	assert.Equal(t, "^\\d+$", attr.Pattern)
	assert.Equal(t, JSONMap{"required": "Age is required."}, attr.CustomErrors)
	assert.Equal(t, "User", attr.Target)
	assert.Equal(t, "oneToOne", attr.Relation)
	assert.Equal(t, uint(1), *attr.CollectionID)
}

func TestAttributeJSONMarshalling(t *testing.T) {
	id := uint(2)
	collectionID := &id
	attr := Attribute{
		Name:     "Username",
		Type:     "string",
		Required: true,
		Unique:   true,
		Options:  []string{"admin", "user"},
		Min:      intPtr(3),
		Max:      intPtr(20),
		Pattern:  "^[a-zA-Z0-9_]+$",
		CustomErrors: JSONMap{
			"required": "Username cannot be empty.",
			"unique":   "Username already exists.",
		},
		Target:       "",
		Relation:     "",
		CollectionID: collectionID,
	}

	jsonData, err := json.Marshal(attr)
	assert.NoError(t, err)

	var unmarshalledAttr Attribute
	err = json.Unmarshal(jsonData, &unmarshalledAttr)
	assert.NoError(t, err)

	// Compare original and unmarshalled Attribute
	assert.Equal(t, attr.Name, unmarshalledAttr.Name)
	assert.Equal(t, attr.Type, unmarshalledAttr.Type)
	assert.Equal(t, attr.Required, unmarshalledAttr.Required)
	assert.Equal(t, attr.Unique, unmarshalledAttr.Unique)
	assert.Equal(t, attr.Options, unmarshalledAttr.Options)
	if attr.Min != nil && unmarshalledAttr.Min != nil {
		assert.Equal(t, *attr.Min, *unmarshalledAttr.Min)
	} else {
		assert.Equal(t, attr.Min, unmarshalledAttr.Min)
	}
	if attr.Max != nil && unmarshalledAttr.Max != nil {
		assert.Equal(t, *attr.Max, *unmarshalledAttr.Max)
	} else {
		assert.Equal(t, attr.Max, unmarshalledAttr.Max)
	}
	assert.Equal(t, attr.Pattern, unmarshalledAttr.Pattern)
	assert.Equal(t, attr.CustomErrors, unmarshalledAttr.CustomErrors)
	assert.Equal(t, attr.Target, unmarshalledAttr.Target)
	assert.Equal(t, attr.Relation, unmarshalledAttr.Relation)
}

func TestAttributeCRUD(t *testing.T) {
	db := setupDatabase(t)

	id := uint(3)
	collectionID := &id

	// Create
	attr := Attribute{
		Name:         "Email",
		Type:         "string",
		Required:     true,
		Unique:       true,
		Options:      nil,
		Min:          nil,
		Max:          nil,
		Pattern:      "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$",
		CustomErrors: JSONMap{"required": "Email is required.", "pattern": "Invalid email format."},
		Target:       "",
		Relation:     "",
		CollectionID: collectionID,
	}

	result := db.Create(&attr)
	assert.NoError(t, result.Error)
	assert.NotZero(t, attr.ID) // gorm.Model includes ID

	// Read
	var fetchedAttr Attribute
	result = db.First(&fetchedAttr, attr.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, attr.Name, fetchedAttr.Name)
	assert.Equal(t, attr.Type, fetchedAttr.Type)
	assert.Equal(t, attr.Required, fetchedAttr.Required)
	assert.Equal(t, attr.Unique, fetchedAttr.Unique)
	assert.Equal(t, attr.Pattern, fetchedAttr.Pattern)
	assert.Equal(t, attr.CustomErrors, fetchedAttr.CustomErrors)
	assert.Equal(t, attr.CollectionID, fetchedAttr.CollectionID)

	// Update
	newName := "User Email"
	result = db.Model(&fetchedAttr).Update("Name", newName)
	assert.NoError(t, result.Error)

	var updatedAttr Attribute
	result = db.First(&updatedAttr, attr.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, newName, updatedAttr.Name)

	// Delete
	result = db.Delete(&updatedAttr)
	assert.NoError(t, result.Error)

	var deletedAttr Attribute
	result = db.First(&deletedAttr, attr.ID)
	assert.Error(t, result.Error)
	assert.Equal(t, "record not found", result.Error.Error())
}
