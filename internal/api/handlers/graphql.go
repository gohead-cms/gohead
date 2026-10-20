package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"

	schema "github.com/gohead-cms/gohead/internal/graphql"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// GraphQLHandler executes a GraphQL request. The caller's role is passed to
// resolvers through the context so that queries and mutations enforce the
// same permissions as the REST API.
func GraphQLHandler(c *gin.Context) {
	var request struct {
		Query         string         `json:"query"`
		Variables     map[string]any `json:"variables"`
		OperationName string         `json:"operationName"`
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxGraphQLBody)
	if err := c.ShouldBindJSON(&request); err != nil || request.Query == "" {
		c.Set("response", "Invalid input format")
		c.Set("status", http.StatusBadRequest)
		return
	}

	// The schema lists every type: anonymous callers query what the public
	// role lets them read, without exploring the model, within limits.
	anonymous := c.GetString("role") == auth.RolePublic
	if anonymous {
		if introspection.MatchString(request.Query) {
			c.Set("response", "Authentication required")
			c.Set("details", "introspection needs a token")
			c.Set("status", http.StatusUnauthorized)
			return
		}
		depth, fields, queriesOnly, err := queryShape(request.Query)
		switch {
		case err != nil:
			c.Set("response", "GraphQL request failed")
			c.Set("details", "the query could not be parsed")
			c.Set("status", http.StatusBadRequest)
			return
		case !queriesOnly:
			c.Set("response", "Authentication required")
			c.Set("details", errNotAQuery.Error())
			c.Set("status", http.StatusUnauthorized)
			return
		case depth > anonymousGraphQLDepth || fields > anonymousGraphQLFields:
			c.Set("response", "GraphQL request failed")
			c.Set("details", fmt.Sprintf("anonymous queries are limited to %d levels and %d fields: send a token for larger ones", anonymousGraphQLDepth, anonymousGraphQLFields))
			c.Set("status", http.StatusBadRequest)
			return
		}
	}

	result := graphql.Do(graphql.Params{
		Schema:         schema.GetSchema(),
		RequestString:  request.Query,
		VariableValues: request.Variables,
		OperationName:  request.OperationName,
		Context:        storage.WithActor(auth.WithRole(c.Request.Context(), c.GetString("role")), actorFrom(c)),
	})

	if len(result.Errors) > 0 {
		c.Set("response", "GraphQL request failed")
		c.Set("details", result.Errors[0].Message)
		if anonymous {
			// Validation messages suggest names ("Did you mean …?") of
			// types visitors cannot read.
			c.Set("details", "the query is invalid or asks for content that is not public")
		}
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", result.Data)
	c.Set("status", http.StatusOK)
}

// introspection matches the introspection fields (__typename is fine).
var introspection = regexp.MustCompile(`__(schema|type)\b`)
