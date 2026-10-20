package handlers

import (
	"errors"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
)

// Limits of anonymous GraphQL requests (public access): each level of a
// selection can multiply the work, so depth and size are bounded.
const (
	anonymousGraphQLDepth  = 4
	anonymousGraphQLFields = 200
	maxGraphQLBody         = 1 << 20 // bytes, for every caller
)

var errNotAQuery = errors.New("anonymous visitors can only send queries")

// queryShape parses a GraphQL document and returns its deepest selection
// (fragments expanded), how many fields it selects, and whether it only
// holds queries.
func queryShape(source string) (depth, fields int, queriesOnly bool, err error) {
	doc, err := parser.Parse(parser.ParseParams{Source: source})
	if err != nil {
		return 0, 0, false, err
	}
	fragments := map[string]*ast.FragmentDefinition{}
	var ops []*ast.OperationDefinition
	for _, def := range doc.Definitions {
		switch d := def.(type) {
		case *ast.FragmentDefinition:
			if d.Name != nil {
				fragments[d.Name.Value] = d
			}
		case *ast.OperationDefinition:
			ops = append(ops, d)
		}
	}
	queriesOnly = true
	var walk func(set *ast.SelectionSet, level int, seen map[string]bool)
	walk = func(set *ast.SelectionSet, level int, seen map[string]bool) {
		if set == nil || fields > anonymousGraphQLFields*10 {
			return
		}
		for _, sel := range set.Selections {
			switch s := sel.(type) {
			case *ast.Field:
				fields++
				if level > depth {
					depth = level
				}
				walk(s.SelectionSet, level+1, seen)
			case *ast.InlineFragment:
				walk(s.SelectionSet, level, seen)
			case *ast.FragmentSpread:
				if s.Name == nil || seen[s.Name.Value] {
					continue
				}
				if f := fragments[s.Name.Value]; f != nil {
					seen[s.Name.Value] = true
					walk(f.SelectionSet, level, seen)
					delete(seen, s.Name.Value)
				}
			}
		}
	}
	for _, op := range ops {
		if op.Operation != ast.OperationTypeQuery {
			queriesOnly = false
		}
		walk(op.SelectionSet, 1, map[string]bool{})
	}
	return depth, fields, queriesOnly, nil
}
