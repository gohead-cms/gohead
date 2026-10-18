package runner

import (
	"testing"

	"github.com/gohead-cms/gohead/pkg/llm"

	"github.com/stretchr/testify/assert"
)

func TestTrimHistory(t *testing.T) {
	u := llm.Message{Role: llm.RoleUser}
	a := llm.Message{Role: llm.RoleAssistant}
	tool := llm.Message{Role: llm.RoleTool}

	short := []llm.Message{u, a}
	assert.Equal(t, short, trimHistory(short, 10))

	// The window must not start with a dangling tool result.
	h := []llm.Message{u, a, tool, a, u, a, tool, a}
	got := trimHistory(h, 5)
	assert.Equal(t, []llm.Message{u, a, tool, a}, got)
	assert.Equal(t, llm.RoleUser, got[0].Role)

	assert.Empty(t, trimHistory([]llm.Message{a, tool, a, tool}, 2))
}
