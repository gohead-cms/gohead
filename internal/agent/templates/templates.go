// Package templates holds ready-made agents (summarizer, tagger,
// moderator, translator). A template renders an agent definition from a
// few parameters; the definition is then reviewed and saved like any agent.
package templates

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/gohead-cms/gohead/internal/agent/functions"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Param is one input of a template.
type Param struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Type: collection, field (an attribute of the collection param),
	// fields (several), text, number or bool.
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Default  any      `json:"default,omitempty"`
	Help     string   `json:"help,omitempty"`
	Types    []string `json:"types,omitempty"` // field/fields: allowed attribute types
	// Localized (field/fields): only localized attributes.
	Localized bool `json:"localized,omitempty"`
}

// Template is a ready-made agent.
type Template struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Params      []Param `json:"params"`
	build       func(v values) (map[string]any, error)
}

var textTypes = []string{"string", "text", "richtext"}

var all = []Template{
	{
		ID: "summarizer", Name: "Summarizer",
		Description: "Writes a short summary of a text field into another field when an item is created or updated.",
		Params: []Param{
			{Key: "collection", Label: "Collection", Type: "collection", Required: true},
			{Key: "source", Label: "Text to summarize", Type: "field", Required: true, Types: textTypes},
			{Key: "target", Label: "Field for the summary", Type: "field", Required: true, Types: textTypes},
			{Key: "words", Label: "Maximum words", Type: "number", Default: 40},
			approvalParam(false),
		},
		build: buildSummarizer,
	},
	{
		ID: "tagger", Name: "Tagger",
		Description: "Picks tags for an item from its text, optionally from a fixed list, and stores them in a field.",
		Params: []Param{
			{Key: "collection", Label: "Collection", Type: "collection", Required: true},
			{Key: "sources", Label: "Text to read", Type: "fields", Required: true, Types: textTypes},
			{Key: "target", Label: "Field for the tags", Type: "field", Required: true, Types: []string{"string", "text", "json"},
				Help: "A JSON field gets a list; a text field a comma-separated list."},
			{Key: "tags", Label: "Allowed tags", Type: "text", Help: "Comma-separated. Empty: the agent chooses."},
			{Key: "max", Label: "Maximum tags", Type: "number", Default: 5},
			approvalParam(false),
		},
		build: buildTagger,
	},
	{
		ID: "moderator", Name: "Moderator",
		Description: "Checks new and changed items for abuse, spam or unsafe content and flags them in a field for an editor.",
		Params: []Param{
			{Key: "collection", Label: "Collection", Type: "collection", Required: true},
			{Key: "sources", Label: "Text to check", Type: "fields", Required: true, Types: textTypes},
			{Key: "flag", Label: "Flag field", Type: "field", Required: true, Types: []string{"bool", "boolean", "enum", "enumeration", "string"},
				Help: "A boolean is set to true when flagged; an enum or text field to the values below."},
			{Key: "flagged_value", Label: "Value when flagged", Type: "text", Default: "flagged"},
			{Key: "ok_value", Label: "Value when fine", Type: "text", Default: "ok"},
			{Key: "reason", Label: "Field for the reason", Type: "field", Types: textTypes, Help: "Optional."},
			approvalParam(false),
		},
		build: buildModerator,
	},
	{
		ID: "translator", Name: "Translator",
		Description: "Translates localized fields of new items into the other locales.",
		Params: []Param{
			{Key: "collection", Label: "Collection", Type: "collection", Required: true},
			{Key: "fields", Label: "Fields to translate", Type: "fields", Required: true, Localized: true},
			{Key: "locales", Label: "Locales", Type: "text", Help: "Comma-separated codes. Empty: every locale."},
			{Key: "on_update", Label: "Also when an item is updated", Type: "bool", Default: false,
				Help: "Updates overwrite translations people edited."},
			approvalParam(true),
		},
		build: buildTranslator,
	},
}

func approvalParam(def bool) Param {
	return Param{Key: "approval", Label: "Changes need an editor's approval", Type: "bool", Default: def}
}

// List returns the templates.
func List() []Template { return slices.Clone(all) }

// ErrUnknown is returned for an unknown template.
var ErrUnknown = fmt.Errorf("unknown template")

// Render builds an agent definition (the body of POST /admin/agents) from
// a template and parameters. Errors are about the parameters.
func Render(id string, params map[string]any) (map[string]any, error) {
	for _, t := range all {
		if t.ID != id {
			continue
		}
		v, err := t.read(params)
		if err != nil {
			return nil, err
		}
		def, err := t.build(v)
		if err != nil {
			return nil, err
		}
		name, _ := params["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = t.ID + "-" + v.collection.Name
		}
		def["name"] = name
		provider, model := llm.DefaultProvider()
		if provider == "" {
			provider = "openai"
		}
		def["llm_config"] = map[string]any{"provider": provider, "model": model}
		def["memory"] = map[string]any{"type": "in-memory", "session_scope": "user"}
		def["require_approval"] = v.bool("approval")
		def["limits"] = map[string]any{"max_runs_per_hour": 60}
		return def, nil
	}
	return nil, ErrUnknown
}

// values are validated parameters.
type values struct {
	raw        map[string]any
	collection *models.Collection
}

func (v values) str(key string) string {
	s, _ := v.raw[key].(string)
	return strings.TrimSpace(s)
}

func (v values) strs(key string) []string {
	switch x := v.raw[key].(type) {
	case []string:
		return x
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func (v values) num(key string) int {
	switch x := v.raw[key].(type) {
	case float64:
		return int(x)
	case int:
		return x
	}
	return 0
}

func (v values) bool(key string) bool { b, _ := v.raw[key].(bool); return b }

func (v values) attr(name string) *models.Attribute {
	for i := range v.collection.Attributes {
		if v.collection.Attributes[i].Name == name {
			return &v.collection.Attributes[i]
		}
	}
	return nil
}

// read checks the parameters against the template and the content model,
// and fills in defaults.
func (t Template) read(params map[string]any) (values, error) {
	v := values{raw: map[string]any{}}
	for k, val := range params {
		v.raw[k] = val
	}
	for _, p := range t.Params {
		if _, set := v.raw[p.Key]; !set && p.Default != nil {
			v.raw[p.Key] = p.Default
		}
	}
	name := v.str("collection")
	if name == "" {
		return v, fmt.Errorf("collection is required")
	}
	ct, err := storage.GetCollectionByName(name)
	if err != nil {
		return v, fmt.Errorf("collection '%s' does not exist", name)
	}
	v.collection = ct
	for _, p := range t.Params {
		var names []string
		switch p.Type {
		case "field":
			if s := v.str(p.Key); s != "" {
				names = []string{s}
			}
		case "fields":
			names = v.strs(p.Key)
		default:
			continue
		}
		if p.Required && len(names) == 0 {
			return v, fmt.Errorf("%s is required", strings.ToLower(p.Label))
		}
		for _, n := range names {
			a := v.attr(n)
			if a == nil {
				return v, fmt.Errorf("%s: '%s' is not an attribute of %s", strings.ToLower(p.Label), n, ct.Name)
			}
			if len(p.Types) > 0 && !slices.Contains(p.Types, a.Type) {
				return v, fmt.Errorf("%s: '%s' is a %s attribute; expected %s", strings.ToLower(p.Label), n, a.Type, strings.Join(p.Types, ", "))
			}
			if p.Localized && !a.Localized {
				return v, fmt.Errorf("%s: '%s' is not localized", strings.ToLower(p.Label), n)
			}
		}
	}
	return v, nil
}

// fn is a function spec using a built-in implementation, scoped to the
// template's collection.
func fn(name, description, implKey string, collection string) map[string]any {
	params := map[string]any{"type": "object", "properties": map[string]any{}}
	for _, info := range functions.Catalog() {
		if info.ImplKey == implKey {
			params = info.Parameters
		}
	}
	spec := map[string]any{"name": name, "description": description, "impl_key": implKey, "parameters": params}
	if collection != "" {
		spec["collections"] = []string{collection}
	}
	return spec
}

func eventTrigger(collection string, events ...string) map[string]any {
	return map[string]any{"type": "collection_event", "event_trigger": map[string]any{"collection": collection, "events": events}}
}

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = `"` + n + `"`
	}
	return strings.Join(q, ", ")
}

func buildSummarizer(v values) (map[string]any, error) {
	ct, source, target := v.collection.Name, v.str("source"), v.str("target")
	if source == target {
		return nil, fmt.Errorf("the summary needs its own field")
	}
	words := v.num("words")
	if words <= 0 {
		words = 40
	}
	prompt := fmt.Sprintf(`You write summaries for the "%[1]s" collection.
The message describes an event on an item and gives its data.
1. Read the item's "%[2]s" field. If it is empty, stop without calling any function.
2. If "%[3]s" already summarizes the current "%[2]s", stop.
3. Otherwise write a summary of at most %[4]d words, in the language of the text, as plain text without Markdown or quotes.
4. Save it with save_summary: collection_name "%[1]s", item_id the item's ID, data {"%[3]s": "<the summary>"}.
Never change any other field.`, ct, source, target, words)
	return map[string]any{
		"system_prompt": prompt, "max_turns": 3,
		"trigger":   eventTrigger(ct, "item:created", "item:updated"),
		"functions": []any{fn("save_summary", "Save the summary into the item", "collections.patch_item", ct)},
	}, nil
}

func buildTagger(v values) (map[string]any, error) {
	ct, sources, target := v.collection.Name, v.strs("sources"), v.str("target")
	if slices.Contains(sources, target) {
		return nil, fmt.Errorf("the tags need their own field")
	}
	max := v.num("max")
	if max <= 0 {
		max = 5
	}
	format := `a single string of comma-separated tags, e.g. "travel, food"`
	if a := v.attr(target); a != nil && a.Type == "json" {
		format = `a JSON array of strings, e.g. ["travel", "food"]`
	}
	choice := "Choose short, lowercase tags that describe the topic."
	if tags := v.str("tags"); tags != "" {
		var list []string
		for _, t := range strings.Split(tags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				list = append(list, t)
			}
		}
		sort.Strings(list)
		choice = "Only use tags from this list, spelled exactly: " + strings.Join(list, ", ") + ". If none fits, use no tag."
	}
	prompt := fmt.Sprintf(`You tag items of the "%[1]s" collection.
The message describes an event on an item and gives its data.
1. Read the item's %[2]s fields. If they are all empty, stop without calling any function.
2. %[3]s Pick at most %[4]d tags.
3. If "%[5]s" already holds the same tags, stop.
4. Otherwise save them with save_tags: collection_name "%[1]s", item_id the item's ID, data {"%[5]s": <tags>} where <tags> is %[6]s.
Never change any other field.`, ct, quoted(sources), choice, max, target, format)
	return map[string]any{
		"system_prompt": prompt, "max_turns": 3,
		"trigger":   eventTrigger(ct, "item:created", "item:updated"),
		"functions": []any{fn("save_tags", "Save the tags into the item", "collections.patch_item", ct)},
	}, nil
}

func buildModerator(v values) (map[string]any, error) {
	ct, sources, flag, reason := v.collection.Name, v.strs("sources"), v.str("flag"), v.str("reason")
	a := v.attr(flag)
	flagged, ok := `"`+v.str("flagged_value")+`"`, `"`+v.str("ok_value")+`"`
	switch a.Type {
	case "bool", "boolean":
		flagged, ok = "true", "false"
	case "enum", "enumeration":
		for _, val := range []string{v.str("flagged_value"), v.str("ok_value")} {
			if !slices.Contains(a.Options, val) {
				return nil, fmt.Errorf("'%s' is not an option of %s (options: %s)", val, flag, strings.Join(a.Options, ", "))
			}
		}
	}
	reasonStep := ""
	if reason != "" {
		reasonStep = fmt.Sprintf(` and "%s": a one-sentence reason (empty when fine)`, reason)
	}
	prompt := fmt.Sprintf(`You moderate items of the "%[1]s" collection.
The message describes an event on an item and gives its data.
1. Read the item's %[2]s fields.
2. Decide whether they contain abuse, harassment, hate, spam, scams, sexual content involving minors, or other unsafe content. Strong opinions, criticism and mild language are fine.
3. Save your decision with save_moderation: collection_name "%[1]s", item_id the item's ID, data {"%[3]s": %[4]s when it must be reviewed, %[5]s when it is fine%[6]s}.
4. If "%[3]s" already holds that decision, do not save it again.
Never change any other field. Do not rewrite or delete content.`, ct, quoted(sources), flag, flagged, ok, reasonStep)
	return map[string]any{
		"system_prompt": prompt, "max_turns": 3,
		"trigger":   eventTrigger(ct, "item:created", "item:updated"),
		"functions": []any{fn("save_moderation", "Save the moderation decision into the item", "collections.patch_item", ct)},
	}, nil
}

func buildTranslator(v values) (map[string]any, error) {
	ct, fields := v.collection.Name, v.strs("fields")
	target := "every locale other than the default (see list_locales)"
	if l := v.str("locales"); l != "" {
		var codes []string
		for _, c := range strings.Split(l, ",") {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if !storage.LocaleExists(c) {
				return nil, fmt.Errorf("locale '%s' is not configured", c)
			}
			codes = append(codes, c)
		}
		target = "these locales: " + strings.Join(codes, ", ")
	}
	events := []string{"item:created"}
	if v.bool("on_update") {
		events = append(events, "item:updated")
	}
	prompt := fmt.Sprintf(`You translate items of the "%[1]s" collection.
The message describes an event on an item and gives its data, in the default locale.
1. Call list_locales to know the locales.
2. Translate the item's %[2]s fields into %[3]s. Skip empty fields.
3. For each locale, save the translations with save_translation: collection_name "%[1]s", item_id the item's ID, locale the locale code, data with the translated fields only.
Keep the meaning, tone and formatting (Markdown, HTML, line breaks). Do not translate names of people, brands or products, URLs, code or e-mail addresses.`, ct, quoted(fields), target)
	return map[string]any{
		"system_prompt": prompt, "max_turns": 6,
		"trigger": eventTrigger(ct, events...),
		"functions": []any{
			fn("list_locales", "List the content locales", "collections.list_locales", ""),
			fn("save_translation", "Save an item's values in one locale", "collections.patch_item", ct),
		},
	}, nil
}
