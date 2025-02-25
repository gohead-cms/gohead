package types

import (
	"fmt"

	"github.com/graphql-go/graphql"
)

// TypeRegistry centralizes the mapping of CMS types to GraphQL types.
var TypeRegistry = map[string]graphql.Output{
	"string":    graphql.String,
	"text":      graphql.String,
	"richtext":  graphql.String,
	"int":       graphql.Int,
	"float":     graphql.Float,
	"bool":      graphql.Boolean,
	"date":      graphql.String,
	"json":      graphql.String,
	"enum":      graphql.String,
	"component": nil,
	"relation":  nil,
}

// GetGraphQLType retrieves the corresponding GraphQL type for a CMS type.
func GetGraphQLType(cmsType string) (graphql.Output, error) {
	if gqlType, exists := TypeRegistry[cmsType]; exists {
		if gqlType == nil {
			return nil, fmt.Errorf("type '%s' requires special handling", cmsType)
		}
		return gqlType, nil
	}
	return nil, fmt.Errorf("registry: unsupported attribute type: %s", cmsType)
}
