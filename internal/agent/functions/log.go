package functions

import (
	"context"
	"encoding/json"

	"github.com/gohead-cms/gohead/pkg/logger"
)

// init registers the system.log function with the static function map.
func init() {
	StaticFunctionMap["system.log"] = logMessage
}

// logMessage is the implementation for the "system.log" agent function.
// It logs a message and associated data to the worker's console.
func logMessage(ctx context.Context, args any) (string, error) {
	argString, ok := args.(string)
	if !ok {
		// This would happen if something other than a string is passed.
		return `{"status": "error", "message": "invalid arguments type, expected a string"}`, nil
	}

	var argMap map[string]any
	err := json.Unmarshal([]byte(argString), &argMap)
	if err != nil {
		// This handles cases where the LLM sends a malformed JSON string.
		return `{"status": "error", "message": "invalid JSON format in arguments"}`, nil
	}

	message, _ := argMap["message"].(string)
	data, _ := argMap["data"]

	// Log the message with structured data for easy debugging.
	logger.Log.
		WithField("source", "system.log_function").
		WithField("data", data).
		Info(message)

	// Return a success message to the LLM.
	response := map[string]any{
		"status":  "success",
		"message": "Message was logged successfully.",
	}
	responseBytes, _ := json.Marshal(response)
	return string(responseBytes), nil
}
