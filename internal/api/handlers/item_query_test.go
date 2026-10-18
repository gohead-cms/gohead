package handlers

import (
	"net/url"
	"testing"

	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseItemQuery(t *testing.T) {
	v, _ := url.ParseQuery("page=2&pageSize=5&filters[title][$contains]=go&filters[views][$gte]=10&filters[published]=true&sort=views:desc,title")
	q, err := parseItemQuery(v)
	require.NoError(t, err)
	assert.Equal(t, 2, q.Page)
	assert.Equal(t, 5, q.PageSize)
	assert.Equal(t, []storage.ItemFilter{
		{Field: "published", Op: storage.OpEq, Value: "true"},
		{Field: "title", Op: storage.OpContains, Value: "go"},
		{Field: "views", Op: storage.OpGte, Value: "10"},
	}, q.Filters)
	assert.Equal(t, []storage.ItemSort{{Field: "views", Desc: true}, {Field: "title"}}, q.Sort)

	for _, bad := range []string{"page=0", "pageSize=1000", "page=abc", "sort=title:sideways", "sort=,"} {
		v, _ := url.ParseQuery(bad)
		_, err := parseItemQuery(v)
		assert.Error(t, err, bad)
	}

	// Unrelated parameters are ignored.
	v, _ = url.ParseQuery("level=2&filter=x&filters=y")
	q, err = parseItemQuery(v)
	require.NoError(t, err)
	assert.Empty(t, q.Filters)
}
