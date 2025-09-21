package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// Initialize the LLM primitives in the function map
func init() {
	// Text Analysis
	StaticFunctionMap["llm.analyze_sentiment"] = analyzeSentiment
	StaticFunctionMap["llm.classify_text"] = classifyText
	StaticFunctionMap["llm.moderate_content"] = moderateContent
	StaticFunctionMap["llm.score_quality"] = scoreQuality

	// Text Generation
	StaticFunctionMap["llm.summarize_text"] = summarizeText
	StaticFunctionMap["llm.rewrite_text"] = rewriteText
	StaticFunctionMap["llm.generate_tags"] = generateTags
	StaticFunctionMap["llm.generate_title"] = generateTitle

	// Decision Making
	StaticFunctionMap["llm.evaluate_condition"] = evaluateCondition
	StaticFunctionMap["llm.rank_items"] = rankItems
	StaticFunctionMap["llm.recommend_action"] = recommendAction

	// Data Extraction
	StaticFunctionMap["llm.extract_entities"] = extractEntities
	StaticFunctionMap["llm.extract_structured_data"] = extractStructuredData
}

// getLLMClient creates an LLM client instance
// You might want to cache this or pass it through context for better performance
func getLLMClient() (llm.Client, error) {
	// Use default config or get from environment/config
	// You may need to adjust this based on your config management
	cfg := llm.Config{
		Provider: "openai", // or get from config
		Model:    "gpt-4o", // or get from config
	}
	return llm.NewAdapter(cfg)
}

// parseArgs handles both string and map arguments
func parseArgs(args any) (map[string]any, error) {
	switch v := args.(type) {
	case string:
		var argMap map[string]any
		if err := json.Unmarshal([]byte(v), &argMap); err != nil {
			return nil, fmt.Errorf("invalid JSON format: %w", err)
		}
		return argMap, nil
	case map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("invalid arguments type")
	}
}

// TEXT ANALYSIS FUNCTIONS

// analyzeSentiment returns positive/negative/neutral with confidence score
func analyzeSentiment(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, ok := argMap["text"].(string)
	if !ok || text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: `You are a sentiment analyzer. Analyze the sentiment of the given text and respond with ONLY a JSON object in this exact format:
{"sentiment": "positive|negative|neutral", "confidence": 0.0-1.0, "explanation": "brief reason"}`,
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Analyze the sentiment of: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	// Parse and validate the response
	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		// If parsing fails, wrap the response
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// classifyText categorizes text into predefined categories
func classifyText(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	categories, _ := argMap["categories"].([]any)

	if text == "" || len(categories) == 0 {
		return `{"status": "error", "message": "missing required parameters: text and categories"}`, nil
	}

	// Convert categories to strings
	catStrings := make([]string, len(categories))
	for i, cat := range categories {
		catStrings[i] = fmt.Sprintf("%v", cat)
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a text classifier. Classify the given text into one of these categories: %s. 
Respond with ONLY a JSON object: {"category": "chosen_category", "confidence": 0.0-1.0, "reasoning": "brief explanation"}`,
				strings.Join(catStrings, ", ")),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Classify this text: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// moderateContent checks for inappropriate content, spam, harassment
func moderateContent(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: `You are a content moderator. Check the text for inappropriate content, spam, harassment, hate speech, or policy violations.
Respond with ONLY a JSON object: {"safe": true/false, "issues": ["list", "of", "issues"], "severity": "none|low|medium|high", "explanation": "brief description"}`,
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Moderate this content: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// scoreQuality rates text quality on a scale (0-100)
func scoreQuality(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	criteria, _ := argMap["criteria"].([]any) // optional criteria

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	criteriaStr := "clarity, coherence, grammar, relevance, and completeness"
	if len(criteria) > 0 {
		critStrings := make([]string, len(criteria))
		for i, crit := range criteria {
			critStrings[i] = fmt.Sprintf("%v", crit)
		}
		criteriaStr = strings.Join(critStrings, ", ")
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a text quality evaluator. Rate the text quality based on: %s.
Respond with ONLY a JSON object: {"score": 0-100, "breakdown": {"criterion": score}, "feedback": "improvement suggestions"}`, criteriaStr),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Rate the quality of: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// TEXT GENERATION FUNCTIONS

// summarizeText creates summaries of varying lengths
func summarizeText(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	maxLength, _ := argMap["max_length"].(float64) // optional, in words
	style, _ := argMap["style"].(string)           // optional: bullet_points, paragraph, etc.

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	lengthInstruction := ""
	if maxLength > 0 {
		lengthInstruction = fmt.Sprintf(" Keep it under %d words.", int(maxLength))
	}

	styleInstruction := "paragraph"
	if style != "" {
		styleInstruction = style
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a text summarizer. Create a concise summary in %s format.%s
Respond with ONLY a JSON object: {"summary": "the summary text", "key_points": ["main", "points"], "word_count": number}`, styleInstruction, lengthInstruction),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Summarize this text: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// rewriteText rewrites text for tone, style, or clarity
func rewriteText(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	tone, _ := argMap["tone"].(string)   // formal, casual, professional, etc.
	style, _ := argMap["style"].(string) // concise, elaborate, simple, etc.
	targetAudience, _ := argMap["audience"].(string)

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	instructions := []string{}
	if tone != "" {
		instructions = append(instructions, fmt.Sprintf("tone: %s", tone))
	}
	if style != "" {
		instructions = append(instructions, fmt.Sprintf("style: %s", style))
	}
	if targetAudience != "" {
		instructions = append(instructions, fmt.Sprintf("target audience: %s", targetAudience))
	}

	instructionStr := "maintaining the original meaning"
	if len(instructions) > 0 {
		instructionStr = strings.Join(instructions, ", ")
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a text rewriter. Rewrite the text with: %s.
Respond with ONLY a JSON object: {"rewritten_text": "the new text", "changes_made": ["list", "of", "changes"], "improvement_score": 0-10}`, instructionStr),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Rewrite this text: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// generateTags extracts relevant tags/keywords from content
func generateTags(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	maxTags, _ := argMap["max_tags"].(float64) // optional

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	tagLimit := 10
	if maxTags > 0 {
		tagLimit = int(maxTags)
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a tag generator. Extract up to %d relevant tags/keywords from the text.
Respond with ONLY a JSON object: {"tags": ["tag1", "tag2"], "categories": ["main", "categories"], "relevance_scores": {"tag": 0.0-1.0}}`, tagLimit),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Generate tags for: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// generateTitle creates titles from content
func generateTitle(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	style, _ := argMap["style"].(string)           // clickbait, academic, news, etc.
	maxLength, _ := argMap["max_length"].(float64) // in characters

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	styleInstruction := "engaging and informative"
	if style != "" {
		styleInstruction = style
	}

	lengthInstruction := ""
	if maxLength > 0 {
		lengthInstruction = fmt.Sprintf(" Maximum %d characters.", int(maxLength))
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a title generator. Create a %s title.%s
Respond with ONLY a JSON object: {"title": "main title", "alternatives": ["alt1", "alt2"], "subtitle": "optional subtitle"}`, styleInstruction, lengthInstruction),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Generate a title for: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// DECISION MAKING FUNCTIONS

// evaluateCondition returns true/false based on criteria
func evaluateCondition(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	condition, _ := argMap["condition"].(string)

	if text == "" || condition == "" {
		return `{"status": "error", "message": "missing required parameters: text and condition"}`, nil
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: `You are a condition evaluator. Evaluate if the given text meets the specified condition.
Respond with ONLY a JSON object: {"result": true/false, "confidence": 0.0-1.0, "reasoning": "explanation", "evidence": ["supporting", "facts"]}`,
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Text: %s\n\nCondition to evaluate: %s", text, condition),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// rankItems orders items by relevance or quality
func rankItems(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	items, _ := argMap["items"].([]any)
	criteria, _ := argMap["criteria"].(string)

	if len(items) == 0 || criteria == "" {
		return `{"status": "error", "message": "missing required parameters: items and criteria"}`, nil
	}

	// Convert items to JSON string for the prompt
	itemsJSON, _ := json.Marshal(items)

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: `You are a ranking system. Rank the given items based on the specified criteria.
Respond with ONLY a JSON object: {"ranked_items": [ordered_list], "scores": {"item": score}, "reasoning": {"item": "why ranked here"}}`,
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Items to rank: %s\n\nRanking criteria: %s", string(itemsJSON), criteria),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// recommendAction suggests next steps from predefined options
func recommendAction(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	context, _ := argMap["context"].(string)
	actions, _ := argMap["actions"].([]any)
	goal, _ := argMap["goal"].(string)

	if context == "" || len(actions) == 0 {
		return `{"status": "error", "message": "missing required parameters: context and actions"}`, nil
	}

	// Convert actions to strings
	actionStrings := make([]string, len(actions))
	for i, action := range actions {
		actionStrings[i] = fmt.Sprintf("%v", action)
	}

	goalInstruction := ""
	if goal != "" {
		goalInstruction = fmt.Sprintf(" Goal: %s", goal)
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: `You are an action recommender. Based on the context, recommend the best action from the available options.
Respond with ONLY a JSON object: {"recommended_action": "chosen_action", "confidence": 0.0-1.0, "reasoning": "why this action", "risks": ["potential", "issues"], "alternatives": ["other", "viable", "options"]}`,
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Context: %s\n\nAvailable actions: %s%s", context, strings.Join(actionStrings, ", "), goalInstruction),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// DATA EXTRACTION FUNCTIONS

// extractEntities pulls out names, dates, locations, etc.
func extractEntities(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	entityTypes, _ := argMap["entity_types"].([]any) // optional: person, location, date, organization, etc.

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	typesInstruction := "all entity types (persons, locations, dates, organizations, products, etc.)"
	if len(entityTypes) > 0 {
		typeStrings := make([]string, len(entityTypes))
		for i, et := range entityTypes {
			typeStrings[i] = fmt.Sprintf("%v", et)
		}
		typesInstruction = strings.Join(typeStrings, ", ")
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are an entity extractor. Extract %s from the text.
Respond with ONLY a JSON object: {"entities": {"type": ["entity1", "entity2"]}, "relationships": [{"from": "entity1", "to": "entity2", "type": "relation"}], "count": number_of_entities}`, typesInstruction),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Extract entities from: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return fmt.Sprintf(`{"status": "success", "raw_response": "%s"}`, response.Content), nil
	}

	result["status"] = "success"
	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// extractStructuredData converts unstructured text to JSON
func extractStructuredData(ctx context.Context, args any) (string, error) {
	argMap, err := parseArgs(args)
	if err != nil {
		return `{"status": "error", "message": "invalid arguments format"}`, nil
	}

	text, _ := argMap["text"].(string)
	schema, _ := argMap["schema"].(map[string]any) // Expected structure

	if text == "" {
		return `{"status": "error", "message": "missing required parameter: text"}`, nil
	}

	schemaInstruction := "appropriate structured format"
	if schema != nil {
		schemaJSON, _ := json.Marshal(schema)
		schemaInstruction = fmt.Sprintf("this exact schema: %s", string(schemaJSON))
	}

	client, err := getLLMClient()
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	messages := []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf(`You are a data extractor. Convert the unstructured text into %s.
Respond with ONLY a JSON object containing the extracted structured data. Include a "_metadata" field with extraction confidence and any issues.`, schemaInstruction),
		},
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Extract structured data from: %s", text),
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Sprintf(`{"status": "error", "message": "%s"}`, err.Error()), nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		// Try to wrap it if it's not proper JSON
		return fmt.Sprintf(`{"status": "success", "data": %s}`, response.Content), nil
	}

	// Add status and wrap in a consistent format
	wrappedResult := map[string]any{
		"status":         "success",
		"extracted_data": result,
	}

	resultBytes, _ := json.Marshal(wrappedResult)
	return string(resultBytes), nil
}

// Helper function to log errors for debugging
func logLLMError(functionName string, err error) {
	if logger.Log != nil {
		logger.Log.WithFields(map[string]any{
			"function": functionName,
			"error":    err.Error(),
		}).Error("LLM primitive function error")
	}
}
