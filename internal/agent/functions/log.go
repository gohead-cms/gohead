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
	argMap, ok := args.(map[string]any)
	if !ok {
		// This case should ideally be handled by the LLM's structured output,
		// but we add a fallback for safety.
		logger.Log.WithField("source", "system.log").Warnf("Invalid arguments format received: %T", args)
		return `{"status": "error", "message": "invalid arguments format"}`, nil
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
