package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{"Basic alphanumeric", "username123", "username123"},
		{"Lowercase conversion", "UserName123", "username123"},
		{"Special characters", "user@name!", "user-name"},
		{"Spaces to hyphens", "user name test", "user-name-test"},
		{"Multiple spaces", "  user  name  ", "user-name"},
		{"Underscores and hyphens", "user_name-test", "user-name-test"},

		// Edge cases
		{"Empty string", "", ""},
		{"Only special characters", "@#$%^&*", ""},
		{"Numbers only", "123456", "123456"},
		{"Mixed case with symbols", "Test@Example!123", "test-example-123"},
		{"Leading and trailing spaces", "  test-user  ", "test-user"},
		{"Repeated non-alphanumeric characters", "user---name", "user-name"},
		{"Accents and diacritics", "naïve café", "naive-cafe"},

		// Complex cases
		{"Long username", strings.Repeat("a", 300), strings.Repeat("a", 300)},
		{"Mixed alphanumeric and symbols", "user123@#$name456!", "user123-name456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GenerateSlug(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
