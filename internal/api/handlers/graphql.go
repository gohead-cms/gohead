package handlers

import (
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

	if err := c.ShouldBindJSON(&request); err != nil || request.Query == "" {
		c.Set("response", "Invalid input format")
		c.Set("status", http.StatusBadRequest)
		return
	}

	// The schema lists every type: anonymous callers query what the public
	// role lets them read, without exploring the model.
	if c.GetString("role") == auth.RolePublic && introspection.MatchString(request.Query) {
		c.Set("response", "Authentication required")
		c.Set("details", "introspection needs a token")
		c.Set("status", http.StatusUnauthorized)
		return
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
		c.Set("status", http.StatusBadRequest)
		return
	}

	c.Set("response", result.Data)
	c.Set("status", http.StatusOK)
}

// introspection matches the introspection fields (__typename is fine).
var introspection = regexp.MustCompile(`__(schema|type)\b`)
