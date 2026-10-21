package apps

import "github.com/gohead-cms/gohead/internal/models"

// Preset is a service GoHead knows how to connect to: the admin UI fills
// in its URL and asks for its token.
type Preset struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	URL         string `json:"url"`
	AuthType    string `json:"authType"`
	HeaderName  string `json:"headerName,omitempty"`
	// TokenLabel names the token people paste, TokenHelp says where to get
	// one, TokenURL links to that page.
	TokenLabel string `json:"tokenLabel"`
	TokenHelp  string `json:"tokenHelp"`
	TokenURL   string `json:"tokenUrl,omitempty"`
	DocsURL    string `json:"docsUrl,omitempty"`
}

// Presets are the services of the catalog, besides custom servers.
var Presets = []Preset{
	{
		ID: "github", Label: "GitHub",
		Description: "Issues, pull requests, repositories and code search, with GitHub's MCP server.",
		URL:         "https://api.githubcopilot.com/mcp/", AuthType: models.ConnectionAuthBearer,
		TokenLabel: "Personal access token",
		TokenHelp:  "A fine-grained token limited to the repositories and permissions the agents need.",
		TokenURL:   "https://github.com/settings/personal-access-tokens/new",
		DocsURL:    "https://github.com/github/github-mcp-server",
	},
	{
		ID: "zapier", Label: "Zapier",
		Description: "Gmail, Google Drive, Sheets, Slack, Notion and thousands of other apps, through the actions you enable in Zapier MCP.",
		URL:         "https://mcp.zapier.com/api/v1/connect", AuthType: models.ConnectionAuthBearer,
		TokenLabel: "Zapier MCP token",
		TokenHelp:  "At mcp.zapier.com, click Add MCP server and choose \"Other\" as the AI agent, add the actions agents may use, then in Settings → API token click Generate token. It is shown once.",
		TokenURL:   "https://mcp.zapier.com/",
		DocsURL:    "https://docs.zapier.com/mcp/get-started/connect/other",
	},
}

// PresetByID returns a preset.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}
