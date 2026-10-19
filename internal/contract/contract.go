// Package contract builds a fixed content model and compares the public API
// generated from it (OpenAPI document, GraphQL introspection) with the
// snapshots in api/, so changes to the API surface are visible in review and
// checked for breaking changes in CI.
//
// After an intended change, regenerate the snapshots:
//
//	UPDATE_API_CONTRACT=1 go test ./internal/openapi ./internal/graphql -run TestAPIContract
package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Seed creates a content model that uses every attribute type, a
// component, a singleton and relations.
func Seed() error {
	min, max := 1, 5
	if err := storage.CreateComponent(&models.Component{
		Name: "seo",
		Attributes: []models.ComponentAttribute{
			{BaseAttribute: models.BaseAttribute{Name: "metaTitle", Type: "string", Required: true}},
			{BaseAttribute: models.BaseAttribute{Name: "metaDescription", Type: "text"}},
		},
	}); err != nil {
		return fmt.Errorf("component: %w", err)
	}
	authors := models.Collection{Name: "authors", Kind: "collection", Attributes: []models.Attribute{
		{Name: "name", Type: "string", Required: true, Unique: true},
		{Name: "bio", Type: "text"},
	}}
	if err := storage.SaveCollection(&authors); err != nil {
		return fmt.Errorf("authors: %w", err)
	}
	articles := models.Collection{Name: "articles", Kind: "collection", DraftAndPublish: true, Attributes: []models.Attribute{
		{Name: "title", Type: "string", Required: true},
		{Name: "body", Type: "richtext"},
		{Name: "views", Type: "int", Min: &min},
		{Name: "rating", Type: "float"},
		{Name: "featured", Type: "bool"},
		{Name: "publishedOn", Type: "date"},
		{Name: "updatedOn", Type: "datetime"},
		{Name: "category", Type: "enum", Options: []string{"news", "tutorial"}},
		{Name: "stars", Type: "int", Max: &max},
		{Name: "cover", Type: "media", AllowedTypes: []string{"image/*"}},
		{Name: "gallery", Type: "media", Multiple: true},
		{Name: "seo", Type: "component", ComponentRef: "seo"},
		{Name: "faq", Type: "component", ComponentRef: "seo", Repeatable: true},
		{Name: "author", Type: "relation", Relation: "oneToOne", Target: "authors"},
		{Name: "coauthors", Type: "relation", Relation: "manyToMany", Target: "authors"},
	}}
	if err := storage.SaveCollection(&articles); err != nil {
		return fmt.Errorf("articles: %w", err)
	}
	if err := storage.SaveOrUpdateSingleton(&models.Singleton{Name: "site", Attributes: []models.Attribute{
		{Name: "title", Type: "string", Required: true},
		{Name: "tagline", Type: "text"},
	}}); err != nil {
		return fmt.Errorf("singleton: %w", err)
	}
	return nil
}

// Path is the snapshot file name under api/ in the repository.
func Path(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "api", name)
}

// Encode returns a stable, indented JSON encoding: object keys are sorted,
// and lists of named objects (GraphQL types, fields, arguments, enum values)
// are sorted by name, since their order is not part of the contract.
func Encode(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(sortNamed(generic)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sortNamed(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			t[k] = sortNamed(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = sortNamed(x)
		}
		named := len(t) > 0
		for _, x := range t {
			m, ok := x.(map[string]any)
			if !ok {
				named = false
				break
			}
			if _, ok := m["name"].(string); !ok {
				named = false
				break
			}
		}
		if named {
			sort.SliceStable(t, func(i, j int) bool {
				return t[i].(map[string]any)["name"].(string) < t[j].(map[string]any)["name"].(string)
			})
		}
		return t
	default:
		return v
	}
}

// Compare checks got against the snapshot, or rewrites the snapshot when
// UPDATE_API_CONTRACT is set. It returns a readable error on a mismatch.
func Compare(name string, got []byte) error {
	path := Path(name)
	if os.Getenv("UPDATE_API_CONTRACT") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, got, 0o644)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("missing API snapshot %s (run with UPDATE_API_CONTRACT=1): %w", path, err)
	}
	if !bytes.Equal(want, got) {
		return fmt.Errorf("the public API changed: api/%s differs from the generated one.\n"+
			"If the change is intended, regenerate the snapshot with\n"+
			"  UPDATE_API_CONTRACT=1 go test ./internal/openapi ./internal/graphql -run TestAPIContract\n"+
			"and commit it (CI reports breaking changes against the last release)", name)
	}
	return nil
}

// IntrospectionQuery is the standard GraphQL introspection query.
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types { ...FullType }
    directives { name description locations args { ...InputValue } }
  }
}
fragment FullType on __Type {
  kind name description
  fields(includeDeprecated: true) {
    name description
    args { ...InputValue }
    type { ...TypeRef }
    isDeprecated deprecationReason
  }
  inputFields { ...InputValue }
  interfaces { ...TypeRef }
  enumValues(includeDeprecated: true) { name description isDeprecated deprecationReason }
  possibleTypes { ...TypeRef }
}
fragment InputValue on __InputValue {
  name description
  type { ...TypeRef }
  defaultValue
}
fragment TypeRef on __Type {
  kind name
  ofType { kind name ofType { kind name ofType { kind name ofType { kind name ofType { kind name ofType { kind name ofType { kind name } } } } } } }
}
`
