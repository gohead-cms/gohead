package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug creates a URL-friendly slug from the provided username.
func GenerateSlug(username string) string {
	// Convert to lowercase
	slug := strings.ToLower(username)

	// Replace non-alphanumeric characters with a hyphen
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}
