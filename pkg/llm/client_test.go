package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	config "github.com/gohead-cms/gohead/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/tools"
)

// TestNewAdapter validates the NewAdapter factory function for all providers.
func TestNewAdapter(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.LLMConfig
		expectedErr bool
		errText     string
	}{
		{
			name: "success with openai",
			cfg: config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "fake-key",
			},
			expectedErr: false,
		},
		{
			name: "success with ollama",
			cfg: config.LLMConfig{
				Provider: "ollama",
				Model:    "llama3",
			},
			expectedErr: false,
		},
		{
			name: "success with anthropic",
			cfg: config.LLMConfig{
				Provider: "anthropic",
				Model:    "claude-3-opus",
				APIKey:   "fake-key",
			},
			expectedErr: false,
		},
		{
			name: "failure with unsupported provider",
			cfg: config.LLMConfig{
				Provider: "unsupported_provider",
				Model:    "some-model",
			},
			expectedErr: true,
			errText:     "unsupported LLM provider: unsupported_provider",
		},
		{
			name: "failure with missing openai api key",
			cfg: config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "",
			},
			expectedErr: true,
			errText:     "OPENAI_API_KEY is not set",
		},
		{
			name: "failure with missing anthropic api key",
			cfg: config.LLMConfig{
				Provider: "anthropic",
				Model:    "claude-3-opus",
				APIKey:   "",
			},
			expectedErr: true,
			errText:     "ANTHROPIC_API_KEY is not set",
		},
		{
			name: "failure with missing ollama model",
			cfg: config.LLMConfig{
				Provider: "ollama",
				Model:    "",
			},
			expectedErr: true,
			errText:     "Ollama model name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAdapter(tt.cfg)
			if tt.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				// Assert that it implements the interface (don’t bind to concretes)
				assert.Implements(t, (*Client)(nil), client, "adapter must implement Client interface")
			}
		})
	}
}

// TestChatOpenAI tests the Chat method for the OpenAI provider.
func TestChatOpenAI(t *testing.T) {
	tests := []struct {
		name              string
		inputMessages     []Message
		inputOptions      []Option
		mockApiResponse   string
		mockApiStatusCode int
		requestAsserter   func(t *testing.T, r *http.Request)
		expectedResponse  *Response
		expectErr         bool
		expectedErrText   string
	}{
		{
			name: "simple text response",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "id": "chatcmpl-123",
                "object": "chat.completion",
                "created": 1677652288,
                "model": "gpt-4",
                "choices": [{
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": "Hi there! How can I help you?"
                    },
                    "finish_reason": "stop"
                }]
            }`,
			mockApiStatusCode: http.StatusOK,
			expectedResponse: &Response{
				Type:    ResponseTypeText,
				Content: "Hi there! How can I help you?",
			},
			expectErr: false,
		},
		{
			name: "tool call response",
			inputMessages: []Message{
				{Role: RoleUser, Content: "What's the weather in Boston?"},
			},
			mockApiResponse: `{
                "id": "chatcmpl-123",
                "object": "chat.completion",
                "created": 1677652288,
                "model": "gpt-4",
                "choices": [{
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": null,
                        "tool_calls": [{
                            "id": "call_abc",
                            "type": "function",
                            "function": {
                                "name": "get_weather",
                                "arguments": "{\"location\": \"Boston, MA\"}"
                            }
                        }]
                    },
                    "finish_reason": "tool_calls"
                }]
            }`,
			mockApiStatusCode: http.StatusOK,
			expectedResponse: &Response{
				Type: ResponseTypeToolCall,
				ToolCall: &ToolCall{
					Name: "get_weather",
					Arguments: map[string]any{
						"location": "Boston, MA",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "request with tools option",
			inputMessages: []Message{
				{Role: RoleUser, Content: "What is 2+2?"},
			},
			inputOptions: []Option{
				WithTools([]tools.Tool{
					tools.Calculator{},
				}),
			},
			mockApiResponse: `{
                "id": "chatcmpl-123",
                "object": "chat.completion",
                "created": 1677652288,
                "model": "gpt-4",
                "choices": [{"index": 0, "message": {"role": "assistant", "content": "The answer is 4."}, "finish_reason": "stop"}]
            }`,
			mockApiStatusCode: http.StatusOK,
			requestAsserter: func(t *testing.T, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var payload map[string]any
				err = json.Unmarshal(body, &payload)
				require.NoError(t, err)
				toolsPayload, ok := payload["tools"]
				require.True(t, ok, "expected 'tools' key in request payload")
				toolsList, ok := toolsPayload.([]any)
				require.True(t, ok, "'tools' should be a list")
				require.Len(t, toolsList, 1)
				tool, ok := toolsList[0].(map[string]any)
				require.True(t, ok)
				function, ok := tool["function"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "calculator", function["name"])
			},
			expectedResponse: &Response{
				Type:    ResponseTypeText,
				Content: "The answer is 4.",
			},
			expectErr: false,
		},
		{
			name: "API returns an error",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "error": {
                    "message": "Internal server error",
                    "type": "server_error",
                    "code": "500"
                }
            }`,
			mockApiStatusCode: http.StatusInternalServerError,
			expectedResponse:  nil,
			expectErr:         true,
			expectedErrText:   "status code 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.requestAsserter != nil {
					bodyBytes, _ := io.ReadAll(r.Body)
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
					tt.requestAsserter(t, r)
				}
				w.WriteHeader(tt.mockApiStatusCode)
				_, _ = w.Write([]byte(tt.mockApiResponse))
			}))
			defer server.Close()

			cfg := config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "fake-api-key",
			}
			client, err := NewAdapter(cfg)
			require.NoError(t, err)

			resp, err := client.Chat(context.Background(), tt.inputMessages, tt.inputOptions...)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrText)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, resp)
			}
		})
	}
}

// TestChatAnthropic tests the Chat method for the Anthropic provider.
func TestChatAnthropic(t *testing.T) {
	tests := []struct {
		name              string
		inputMessages     []Message
		inputOptions      []Option
		mockApiResponse   string
		mockApiStatusCode int
		expectedResponse  *Response
		expectErr         bool
		expectedErrText   string
	}{
		{
			name: "simple text response",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "id": "msg_01",
                "type": "message",
                "role": "assistant",
                "model": "claude-3-opus-20240229",
                "stop_sequence": null,
                "usage": {
                    "input_tokens": 10,
                    "output_tokens": 20
                },
                "content": [
                    {
                        "type": "text",
                        "text": "Hello there! How can I help?"
                    }
                ]
            }`,
			mockApiStatusCode: http.StatusOK,
			expectedResponse: &Response{
				Type:    ResponseTypeText,
				Content: "Hello there! How can I help?",
			},
			expectErr: false,
		},
		{
			name: "tool call response",
			inputMessages: []Message{
				{Role: RoleUser, Content: "What's the weather in Boston?"},
			},
			mockApiResponse: `{
                "id": "msg_02",
                "type": "message",
                "role": "assistant",
                "model": "claude-3-opus-20240229",
                "stop_sequence": null,
                "usage": {
                    "input_tokens": 15,
                    "output_tokens": 30
                },
                "content": [
                    {
                        "type": "tool_use",
                        "id": "toolu_01",
                        "name": "get_weather",
                        "input": {
                            "location": "Boston, MA"
                        }
                    }
                ]
            }`,
			mockApiStatusCode: http.StatusOK,
			expectedResponse: &Response{
				Type: ResponseTypeToolCall,
				ToolCall: &ToolCall{
					Name: "get_weather",
					Arguments: map[string]any{
						"location": "Boston, MA",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "API returns an error",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "type": "error",
                "error": {
                    "type": "internal_server_error",
                    "message": "An unexpected error occurred."
                }
            }`,
			mockApiStatusCode: http.StatusInternalServerError,
			expectedResponse:  nil,
			expectErr:         true,
			expectedErrText:   "API returned error: An unexpected error occurred.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockApiStatusCode)
				_, _ = w.Write([]byte(tt.mockApiResponse))
			}))
			defer server.Close()

			cfg := config.LLMConfig{
				Provider: "anthropic",
				Model:    "claude-3-opus",
				APIKey:   "fake-api-key",
			}
			client, err := NewAdapter(cfg)
			require.NoError(t, err)

			resp, err := client.Chat(context.Background(), tt.inputMessages, tt.inputOptions...)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrText)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, resp)
			}
		})
	}
}

// TestChatOllama tests the Chat method for the Ollama provider.
func TestChatOllama(t *testing.T) {
	tests := []struct {
		name              string
		inputMessages     []Message
		mockApiResponse   string
		mockApiStatusCode int
		expectedResponse  *Response
		expectErr         bool
		expectedErrText   string
	}{
		{
			name: "simple text response",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "model": "llama3",
                "created_at": "2024-05-23T12:00:00Z",
                "message": {
                    "role": "assistant",
                    "content": "Hi there! How can I help?"
                },
                "done": true,
                "total_duration": 100000000,
                "load_duration": 50000000,
                "prompt_eval_duration": 1000000,
                "eval_count": 10,
                "eval_duration": 1000000
            }`,
			mockApiStatusCode: http.StatusOK,
			expectedResponse: &Response{
				Type:    ResponseTypeText,
				Content: "Hi there! How can I help?",
			},
			expectErr: false,
		},
		{
			name: "API returns an error",
			inputMessages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			mockApiResponse: `{
                "error": "The model 'llama3' was not found."
            }`,
			mockApiStatusCode: http.StatusNotFound,
			expectedResponse:  nil,
			expectErr:         true,
			expectedErrText:   "status code 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockApiStatusCode)
				_, _ = w.Write([]byte(tt.mockApiResponse))
			}))
			defer server.Close()

			cfg := config.LLMConfig{
				Provider: "ollama",
				Model:    "llama3",
			}
			client, err := NewAdapter(cfg)
			require.NoError(t, err)

			resp, err := client.Chat(context.Background(), tt.inputMessages)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrText)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResponse, resp)
			}
		})
	}
}
