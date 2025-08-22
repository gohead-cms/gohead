package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"

	schema "github.com/gohead-cms/gohead/internal/graphql"
)

// GraphQLHandler handles GraphQL queries
func GraphQLHandler(c *gin.Context) {
	var request struct {
		Query string `json:"query"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.Set("response", "Invalid input format")
		c.Set("status", http.StatusBadRequest)
		return
	}

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:        schema.Schema,
		RequestString: request.Query,
	})

	if len(result.Errors) > 0 {
		c.Set("response", "Failed to fetch items")
		c.Set("details", result.Errors[0].Message)
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", result.Data)
	c.Set("status", http.StatusOK)
}
