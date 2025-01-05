package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// removeAccents removes diacritics (accents) from a string.
func removeAccents(input string) string {
	var sb strings.Builder
	for _, r := range norm.NFD.String(input) {
		if unicode.IsMark(r) {
			continue // Skip diacritical marks
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// GenerateSlug creates a URL-friendly slug from the provided username.
func GenerateSlug(username string) string {
	// Normalize and remove accents
	slug := removeAccents(username)

	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Replace non-alphanumeric characters with a hyphen
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}
