package storage

import "fmt"

// Custom error types
type DuplicateEntryError struct {
	Field string
}

func (e *DuplicateEntryError) Error() string {
	return fmt.Sprintf("duplicate entry for field: %s", e.Field)
}

type GeneralDatabaseError struct {
	Message string
}

func (e *GeneralDatabaseError) Error() string {
	return fmt.Sprintf("database error: %s", e.Message)
}
