package agents

import (
	"context"
	"encoding/json"
	"fmt"
)

// The generic data structure for a workflow.
type Workflow struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Nodes       map[string]Node `json:"nodes"`
	StartNodeID string          `json:"startNodeId"`
}

// A generic node that can be any type of action.
type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Use json.RawMessage to hold the specific configuration for each node type.
	Config json.RawMessage `json:"config"`
	// Connections define the output paths based on the result.
	Connections map[string]string `json:"connections"` // e.g., {"success": "next_node_id", "failure": "error_node_id"}
}

// ExecutionContext stores the data that flows through the workflow.
type ExecutionContext map[string]interface{}

// Engine is the runtime for the workflow.
type Engine struct {
	// You can add more fields here like a logger, database connections, etc.
}

// NewEngine creates a new workflow engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Run executes a given workflow definition.
func (e *Engine) Run(ctx context.Context, wf *Workflow) (ExecutionContext, error) {
	// Create an empty context for this run.
	flowContext := make(ExecutionContext)
	currentNodeID := wf.StartNodeID

	for currentNodeID != "" {
		node, ok := wf.Nodes[currentNodeID]
		if !ok {
			return nil, fmt.Errorf("node with ID '%s' not found in workflow", currentNodeID)
		}

		// A switch statement is a clean way to handle different node types.
		var nextNodeID string
		var err error

		switch node.Type {
		case "start":
			// The start node just moves to the next one.
			nextNodeID, err = e.executeStartNode(ctx, &node, flowContext)
		case "http-request":
			nextNodeID, err = e.executeHTTPRequestNode(ctx, &node, flowContext)
		case "conditional":
			nextNodeID, err = e.executeConditionalNode(ctx, &node, flowContext)
		case "database-query":
			nextNodeID, err = e.executeDatabaseQueryNode(ctx, &node, flowContext)
		case "end":
			return flowContext, nil // End of workflow.
		default:
			return nil, fmt.Errorf("unsupported node type: %s", node.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("error executing node '%s': %w", node.ID, err)
		}

		// Move to the next node based on the result.
		currentNodeID = nextNodeID
	}

	return flowContext, nil
}

// Private methods to handle specific node types.
// These methods contain your actual business logic.
func (e *Engine) executeStartNode(ctx context.Context, node *Node, flowContext ExecutionContext) (string, error) {
	// The start node simply returns its "next" connection ID.
	return node.Connections["next"], nil
}

func (e *Engine) executeHTTPRequestNode(ctx context.Context, node *Node, flowContext ExecutionContext) (string, error) {
	// Logic to make an HTTP request based on node.Config.
	// Update the flowContext with the response data.
	// Return the "success" or "failure" connection ID.
	return node.Connections["success"], nil
}

func (e *Engine) executeConditionalNode(ctx context.Context, node *Node, flowContext ExecutionContext) (string, error) {
	// Logic to evaluate a condition based on node.Config and flowContext.
	// For example: `flowContext["user_input"] == "yes"`.
	// Return the "true" or "false" connection ID.
	return node.Connections["true"], nil
}

func (e *Engine) executeDatabaseQueryNode(ctx context.Context, node *Node, flowContext ExecutionContext) (string, error) {
	// Logic to query the database.
	// Update flowContext with the query result.
	// Return the "success" or "failure" connection ID.
	return node.Connections["success"], nil
}
