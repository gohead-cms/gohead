package types

import (
	"fmt"

	"github.com/graphql-go/graphql"
)

// TypeRegistry centralizes the mapping of CMS types to GraphQL types.
var TypeRegistry = map[string]graphql.Output{
	"string":      graphql.String,
	"text":        graphql.String,
	"richtext":    graphql.String,
	"email":       graphql.String,
	"password":    graphql.String, // not queryable
	"int":         graphql.Int,
	"integer":     graphql.Int,
	"float":       graphql.Float,
	"decimal":     graphql.Float,
	"bool":        graphql.Boolean,
	"boolean":     graphql.Boolean,
	"date":        graphql.DateTime,
	"datetime":    graphql.DateTime,
	"time":        graphql.String,
	"json":        graphql.String,
	"enum":        graphql.String,
	"enumeration": graphql.String,
	"media":       graphql.String,
	"uid":         graphql.String,
	"component":   nil,
	"dynamiczone": nil,
	"relation":    nil,
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
