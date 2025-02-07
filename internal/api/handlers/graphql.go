package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"

	schema "gohead/internal/graphql"
)

var Schema graphql.Schema

// InitializeGraphQLSchema dynamically generates the GraphQL schema.
func InitializeGraphQLSchema() error {
	rootQuery, err := schema.GenerateGraphQLQueries()
	if err != nil {
		return err
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})
	if err != nil {
		return err
	}

	Schema = schema
	return nil
}

// GraphQLHandler handles GraphQL queries
func GraphQLHandler(c *gin.Context) {
	var request struct {
		Query string `json:"query"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:        Schema,
		RequestString: request.Query,
	})

	if len(result.Errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": result.Errors})
		return
	}

	c.JSON(http.StatusOK, result)
}
