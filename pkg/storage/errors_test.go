package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/storage"
)

func TestDuplicateEntryError(t *testing.T) {
	// Create an instance of DuplicateEntryError
	duplicateError := &storage.DuplicateEntryError{
		Field: "username",
	}

	// Check the error message
	expectedMessage := "duplicate entry for field: username"
	assert.Equal(t, expectedMessage, duplicateError.Error(), "DuplicateEntryError message should match expected format")
}

func TestGeneralDatabaseError(t *testing.T) {
	// Create an instance of GeneralDatabaseError
	generalError := &storage.GeneralDatabaseError{
		Message: "connection failed",
	}

	// Check the error message
	expectedMessage := "database error: connection failed"
	assert.Equal(t, expectedMessage, generalError.Error(), "GeneralDatabaseError message should match expected format")
}
