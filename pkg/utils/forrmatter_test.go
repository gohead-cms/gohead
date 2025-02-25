package utils

import (
	"testing"

	"gohead/internal/models"
)

// TestFormatCollectionSchema tests the FormatCollectionSchema function.
func TestFormatCollectionSchema(t *testing.T) {
	collection := &models.Collection{
		ID:   1,
		Name: "test_collection",
		Attributes: []models.Attribute{
			{
				Name:     "title",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "author",
				Type:     "relation",
				Required: false,
				Target:   "user",
				Relation: "oneToOne",
			},
		},
	}

	result := FormatCollectionSchema(collection)

	// Check top-level keys
	if result["id"] != 1 {
		t.Errorf("expected id to be 1, got %v", result["id"])
	}

	if result["uid"] != "api::test_collection.test_collection" {
		t.Errorf("expected uid to be api::test_collection.test_collection, got %v", result["uid"])
	}

	schema, ok := result["schema"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected schema to be a map, got %T", result["schema"])
	}

	// Check schema content
	if schema["collectionName"] != "test_collection" {
		t.Errorf("expected collectionName to be 'test_collection', got %v", schema["collectionName"])
	}

	info, ok := schema["info"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected info to be a map, got %T", schema["info"])
	}
	if info["displayName"] != "Test_collection" {
		t.Errorf("expected displayName to be 'Test_collection', got %v", info["displayName"])
	}

	attrs, ok := schema["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attributes to be a map, got %T", schema["attributes"])
	}

	// Check individual attributes
	titleAttr, exists := attrs["title"].(map[string]interface{})
	if !exists {
		t.Fatalf("expected 'title' attribute to exist")
	}
	if titleAttr["type"] != "string" {
		t.Errorf("expected title.type to be 'string', got %v", titleAttr["type"])
	}
	if titleAttr["required"] != true {
		t.Errorf("expected title.required to be true, got %v", titleAttr["required"])
	}

	authorAttr, exists := attrs["author"].(map[string]interface{})
	if !exists {
		t.Fatalf("expected 'author' attribute to exist")
	}
	if authorAttr["type"] != "relation" {
		t.Errorf("expected author.type to be 'relation', got %v", authorAttr["type"])
	}
	if authorAttr["relation"] != "oneToOne" {
		t.Errorf("expected author.relation to be 'oneToOne', got %v", authorAttr["relation"])
	}
	if authorAttr["target"] != "api::user.user" {
		t.Errorf("expected author.target to be 'api::user.user', got %v", authorAttr["target"])
	}
}

// TestFormatCollectionItem tests the FormatCollectionItem function.
func TestFormatCollectionItem(t *testing.T) {
	collection := &models.Collection{
		ID:   1,
		Name: "articles",
		Attributes: []models.Attribute{
			{
				Name: "title",
				Type: "string",
			},
			{
				Name:     "author",
				Type:     "relation",
				Relation: "oneToOne",
				Target:   "users",
			},
			{
				Name:     "tags",
				Type:     "relation",
				Relation: "manyToMany",
				Target:   "tags",
			},
		},
	}

	item := &models.Item{
		ID: 101,
		Data: map[string]interface{}{
			"title":  "Hello World",
			"author": 202,                    // single relation
			"tags":   []interface{}{1, 2, 3}, // many-to-many
		},
	}

	result := FormatCollectionItem(item, collection)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result["id"] != 101 {
		t.Errorf("expected id to be 101, got %v", result["id"])
	}

	attributes, ok := result["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attributes to be a map, got %T", result["attributes"])
	}

	if attributes["title"] != "Hello World" {
		t.Errorf("expected title to be 'Hello World', got %v", attributes["title"])
	}

	// Check oneToOne relation
	author, ok := attributes["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'author' to be a map, got %T", attributes["author"])
	}
	authorData, ok := author["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'author.data' to be a map, got %T", author["data"])
	}
	if authorData["id"] != 202 {
		t.Errorf("expected 'author.data.id' to be 202, got %v", authorData["id"])
	}

	// Check manyToMany relation
	tags, ok := attributes["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'tags' to be a map, got %T", attributes["tags"])
	}
	tagsData, ok := tags["data"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected 'tags.data' to be a slice of maps, got %T", tags["data"])
	}
	expectedTagIDs := []interface{}{1, 2, 3}
	for i, tagMap := range tagsData {
		if tagMap["id"] != expectedTagIDs[i] {
			t.Errorf("expected tag id %v, got %v", expectedTagIDs[i], tagMap["id"])
		}
	}
}

// TestFormatCollectionItems tests the FormatCollectionItems function.
func TestFormatCollectionItems(t *testing.T) {
	collection := &models.Collection{
		ID:   2,
		Name: "posts",
		Attributes: []models.Attribute{
			{
				Name: "title",
				Type: "string",
			},
		},
	}

	items := []models.Item{
		{
			ID: 1,
			Data: map[string]interface{}{
				"title": "First Post",
			},
		},
		{
			ID: 2,
			Data: map[string]interface{}{
				"title": "Second Post",
			},
		},
	}

	result := FormatCollectionItems(items, collection)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	// Check first item
	if result[0]["id"] != 1 {
		t.Errorf("expected first item ID to be 1, got %v", result[0]["id"])
	}
	attrs1, ok := result[0]["attributes"].(map[string]interface{})
	if !ok {
		t.Errorf("expected attributes to be a map for first item")
	} else {
		if attrs1["title"] != "First Post" {
			t.Errorf("expected title to be 'First Post', got %v", attrs1["title"])
		}
	}

	// Check second item
	if result[1]["id"] != 2 {
		t.Errorf("expected second item ID to be 2, got %v", result[1]["id"])
	}
	attrs2, ok := result[1]["attributes"].(map[string]interface{})
	if !ok {
		t.Errorf("expected attributes to be a map for second item")
	} else {
		if attrs2["title"] != "Second Post" {
			t.Errorf("expected title to be 'Second Post', got %v", attrs2["title"])
		}
	}
}

// TestCapitalize tests the capitalize function.
func TestCapitalize(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "A"},
		{"abc", "Abc"},
		{"test", "Test"},
	}

	for _, c := range cases {
		result := capitalize(c.input)
		if result != c.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", c.input, c.expected, result)
		}
	}
}
