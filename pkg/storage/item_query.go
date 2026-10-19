package storage

import (
	"fmt"
	"strconv"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/validation"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Filter operators supported on item lists.
const (
	OpEq       = "$eq"
	OpNe       = "$ne"
	OpLt       = "$lt"
	OpLte      = "$lte"
	OpGt       = "$gt"
	OpGte      = "$gte"
	OpContains = "$contains"
)

var sqlOperators = map[string]string{
	OpEq: "=", OpNe: "<>", OpLt: "<", OpLte: "<=", OpGt: ">", OpGte: ">=",
}

// ItemFilter is one condition on an item attribute (or a built-in column).
type ItemFilter struct {
	Field string
	Op    string
	Value string
}

// ItemSort orders results by an attribute or a built-in column.
type ItemSort struct {
	Field string
	Desc  bool
}

// ItemQuery describes a filtered, sorted, paginated item listing.
type ItemQuery struct {
	Page     int
	PageSize int
	Filters  []ItemFilter
	Sort     []ItemSort
	// Status filters draft-and-publish collections: "" (all items),
	// models.StatusPublished or models.StatusDraft. Ignored otherwise.
	Status string
	// Locale makes filters and sorts on localized attributes use that
	// locale's values ("" or the default locale: the main data). With
	// Fallback, a missing value falls back to the default locale's.
	Locale   string
	Fallback bool
}

// withStatus restricts q to published or draft items of a draft-and-publish
// collection.
func withStatus(q *gorm.DB, ct models.Collection, status string) *gorm.DB {
	if !ct.DraftAndPublish {
		return q
	}
	switch status {
	case models.StatusPublished:
		return q.Where("published_at IS NOT NULL")
	case models.StatusDraft:
		return q.Where("published_at IS NULL")
	}
	return q
}

// builtinColumns are real columns on the items table that can be filtered
// and sorted on, mapped to their kind.
var builtinColumns = map[string]string{
	"id":           "int",
	"created_at":   "datetime",
	"updated_at":   "datetime",
	"published_at": "datetime",
}

// builtinAliases maps the field names used in responses to their columns.
var builtinAliases = map[string]string{
	"createdAt":   "created_at",
	"updatedAt":   "updated_at",
	"publishedAt": "published_at",
}

// filterableTypes lists attribute types whose stored JSON value is a scalar
// that can be compared. Relations, components, media, json and password
// attributes are deliberately excluded.
var filterableTypes = map[string]string{
	"string": "text", "text": "text", "richtext": "text", "email": "text",
	"uid": "text", "enum": "text", "enumeration": "text",
	"date": "text", "datetime": "text", "time": "text", // ISO strings sort lexically
	"int": "number", "integer": "number", "float": "number", "decimal": "number",
	"bool": "bool", "boolean": "bool",
}

// QueryFieldError reports a filter or sort the collection cannot satisfy.
type QueryFieldError struct{ Msg string }

func (e *QueryFieldError) Error() string { return e.Msg }

// resolvedField is a field that has been checked against the collection schema.
type resolvedField struct {
	column string // set for built-in columns
	path   string // JSON path for attributes, e.g. "$.title" / "title"
	kind   string // "text", "number", "bool", "int", "datetime"
	// locale: the translation to read (localized attributes in another
	// locale than the default); fallback reads the main data when missing.
	locale   string
	fallback bool
}

// localeOf applies the query's locale to a resolved attribute.
func (q ItemQuery) localeOf(ct models.Collection, f resolvedField) resolvedField {
	if q.Locale == "" || f.column != "" {
		return f
	}
	for _, a := range ct.Attributes {
		if a.Name == f.path && a.Localized {
			f.locale, f.fallback = q.Locale, q.Fallback
		}
	}
	return f
}

func resolveField(ct models.Collection, name string) (resolvedField, error) {
	if col, ok := builtinAliases[name]; ok {
		name = col
	}
	if kind, ok := builtinColumns[name]; ok {
		return resolvedField{column: name, kind: kind}, nil
	}
	for _, attr := range ct.Attributes {
		if attr.Name != name {
			continue
		}
		kind, ok := filterableTypes[attr.Type]
		if !ok {
			return resolvedField{}, &QueryFieldError{fmt.Sprintf("attribute '%s' of type '%s' cannot be filtered or sorted", name, attr.Type)}
		}
		return resolvedField{path: name, kind: kind}, nil
	}
	return resolvedField{}, &QueryFieldError{fmt.Sprintf("unknown field '%s'", name)}
}

// jsonValueExpr returns a dialect-specific SQL expression extracting an
// attribute from the JSON "data" column. The attribute name is always bound
// as a parameter, never interpolated.
func jsonValueExpr(dialect string, f resolvedField) (string, []any) {
	raw, args := rawJSONValue(dialect, f)
	switch dialect {
	case "postgres":
		if f.kind == "number" {
			return "CAST(" + raw + " AS NUMERIC)", args
		}
		return "(" + raw + ")", args
	case "mysql":
		if f.kind == "number" {
			return "CAST(JSON_UNQUOTE(" + raw + ") AS DECIMAL(65,10))", args
		}
		return "JSON_UNQUOTE(" + raw + ")", args
	default: // sqlite
		switch f.kind {
		case "number":
			return "CAST(" + raw + " AS REAL)", args
		case "bool":
			// SQLite stores JSON booleans as 1/0.
			return raw, args
		default:
			return "CAST(" + raw + " AS TEXT)", args
		}
	}
}

// rawJSONValue extracts an attribute: from the main data, or from a
// locale's translation (falling back to the main data when f.fallback).
// On PostgreSQL the value is text (->>), elsewhere a JSON/SQL value.
func rawJSONValue(dialect string, f resolvedField) (string, []any) {
	var main, tr string
	var mainArgs, trArgs []any
	switch dialect {
	case "postgres":
		main, mainArgs = "data->>CAST(? AS TEXT)", []any{f.path}
		tr, trArgs = "(CAST(translations AS JSONB) -> CAST(? AS TEXT) ->> CAST(? AS TEXT))", []any{f.locale, f.path}
	case "mysql":
		main, mainArgs = "JSON_EXTRACT(data, ?)", []any{validation.JSONPath(f.path)}
		tr, trArgs = "JSON_EXTRACT(translations, ?)", []any{jsonPath(f.locale, f.path)}
	default:
		main, mainArgs = "json_extract(data, ?)", []any{validation.JSONPath(f.path)}
		tr, trArgs = "json_extract(translations, ?)", []any{jsonPath(f.locale, f.path)}
	}
	switch {
	case f.locale == "":
		return main, mainArgs
	case f.fallback:
		return "COALESCE(" + tr + ", " + main + ")", append(trArgs, mainArgs...)
	default:
		return tr, trArgs
	}
}

// bindValue converts a raw query-string value to the type the SQL expression
// for that field compares against.
func bindValue(dialect string, f resolvedField, raw string) (any, error) {
	switch f.kind {
	case "number", "int":
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, &QueryFieldError{fmt.Sprintf("value '%s' is not a number", raw)}
		}
		return v, nil
	case "bool":
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, &QueryFieldError{fmt.Sprintf("value '%s' is not a boolean", raw)}
		}
		if dialect == "sqlite" {
			if b {
				return 1, nil
			}
			return 0, nil
		}
		return strconv.FormatBool(b), nil
	default:
		return raw, nil
	}
}

func applyFilter(db *gorm.DB, dialect string, ct models.Collection, q ItemQuery, flt ItemFilter) (*gorm.DB, error) {
	f, err := resolveField(ct, flt.Field)
	if err != nil {
		return nil, err
	}
	f = q.localeOf(ct, f)

	var lhs string
	var vars []any
	if f.column != "" {
		lhs = f.column // whitelisted by builtinColumns
	} else {
		lhs, vars = jsonValueExpr(dialect, f)
	}

	if flt.Op == OpContains {
		if f.kind != "text" {
			return nil, &QueryFieldError{fmt.Sprintf("$contains is only supported on text fields, not '%s'", flt.Field)}
		}
		escape := `'\'`
		if dialect == "mysql" {
			escape = `'\\'` // MySQL treats backslash as an escape inside string literals
		}
		return db.Where(lhs+" LIKE ? ESCAPE "+escape, append(vars, "%"+escapeLike(flt.Value)+"%")...), nil
	}

	op, ok := sqlOperators[flt.Op]
	if !ok {
		return nil, &QueryFieldError{fmt.Sprintf("unsupported operator '%s'", flt.Op)}
	}
	if f.kind == "bool" && op != "=" && op != "<>" {
		return nil, &QueryFieldError{fmt.Sprintf("operator '%s' is not supported on boolean field '%s'", flt.Op, flt.Field)}
	}
	val, err := bindValue(dialect, f, flt.Value)
	if err != nil {
		return nil, err
	}
	return db.Where(fmt.Sprintf("%s %s ?", lhs, op), append(vars, val)...), nil
}

func escapeLike(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '%' || r == '_' || r == '\\' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}

// orderByExpr builds a single ORDER BY expression for all sort keys, so that
// several JSON-attribute sorts can be combined (gorm replaces, rather than
// merges, expression-based OrderBy clauses). Built-in column names come from
// a whitelist; attribute paths are bound as parameters.
func orderByExpr(dialect string, ct models.Collection, q ItemQuery) (clause.Expr, error) {
	sorts := q.Sort
	if len(sorts) == 0 {
		return clause.Expr{SQL: "id ASC", WithoutParentheses: true}, nil
	}
	var sql string
	var vars []any
	for i, s := range sorts {
		f, err := resolveField(ct, s.Field)
		if err != nil {
			return clause.Expr{}, err
		}
		f = q.localeOf(ct, f)
		dir := "ASC"
		if s.Desc {
			dir = "DESC"
		}
		if i > 0 {
			sql += ", "
		}
		if f.column != "" {
			sql += f.column + " " + dir
			continue
		}
		expr, args := jsonValueExpr(dialect, f)
		sql += expr + " " + dir
		vars = append(vars, args...)
	}
	sql += ", id ASC" // deterministic pagination
	return clause.Expr{SQL: sql, Vars: vars, WithoutParentheses: true}, nil
}

// QueryItems lists the items of a collection with optional filters and sort.
// Every field is validated against the collection schema; attribute names
// and values are always bound as SQL parameters.
func QueryItems(ct models.Collection, q ItemQuery) ([]models.Item, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 10
	}
	dialect := database.DB.Dialector.Name()

	base := withStatus(database.DB.Model(&models.Item{}).Where("collection_id = ?", ct.ID), ct, q.Status)
	var err error
	for _, flt := range q.Filters {
		if base, err = applyFilter(base, dialect, ct, q, flt); err != nil {
			return nil, 0, err
		}
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}

	order, err := orderByExpr(dialect, ct, q)
	if err != nil {
		return nil, 0, err
	}
	list := base.Session(&gorm.Session{}).Clauses(clause.OrderBy{Expression: order})

	var items []models.Item
	if err := list.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch items: %w", err)
	}
	return items, int(total), nil
}
