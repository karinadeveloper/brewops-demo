// Package ai contains the concrete client for the external LLM provider
// used by POST /inventory/suggest. Only OpenAIClient lives here — parsing
// its response into domain.SuggestedMovement values, and the fuzzy product
// matching, are business logic and belong in
// service.InventorySuggestionService instead.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// model is gpt-4o-mini per CLAUDE.md's AI feature section: cheap, more
// than enough reasoning for parsing short Spanish inventory phrases.
const model = "gpt-4o-mini"

const chatCompletionsURL = "https://api.openai.com/v1/chat/completions"

// systemPrompt instructs the model to extract structured inventory
// movements as JSON — CLAUDE.md asks for OpenAI's structured JSON output
// mode rather than parsing free-form text.
const systemPrompt = `You are an inventory assistant for a small Mexican juice/beverage business. Extract inventory movements from the user's Spanish text and respond with ONLY a JSON object of this exact shape:

{"movements": [{"product_name": "string, the product as mentioned in the text", "type": "PURCHASE | ADJUSTMENT_IN | ADJUSTMENT_OUT", "quantity": integer > 0, "unit_cost_cents": integer >= 0, the price per unit in Mexican peso CENTS (multiply pesos by 100)}]}

Use PURCHASE for buying new stock, ADJUSTMENT_IN for stock found/corrected upward, ADJUSTMENT_OUT for stock lost/damaged/corrected downward. Never use any other "type" value. If a price isn't mentioned, use 0 for unit_cost_cents. Respond with ONLY the JSON object, no other text.`

// OpenAIClient implements domain.InventorySuggestionClient against
// OpenAI's Chat Completions API using net/http directly — this call is
// simple enough (one request, one response) that pulling in the full
// OpenAI SDK isn't warranted, consistent with this project's preference
// for hand-rolled fundamentals over framework dependencies (see CLAUDE.md's
// "manual JWT" rationale).
type OpenAIClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{apiKey: apiKey, httpClient: &http.Client{}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Suggest sends naturalLanguageText to gpt-4o-mini and returns the raw JSON
// string it replied with (expected to match systemPrompt's schema — parsing
// that is the caller's responsibility). ctx's deadline governs the whole
// call; the caller (service) is expected to apply CLAUDE.md's ~15s AI-call
// timeout via context.WithTimeout.
//
// Never logs the request or response body, and never logs c.apiKey — every
// error returned here is a generic domain.ErrAISuggestionFailed wrap with
// no provider response content attached.
func (c *OpenAIClient) Suggest(ctx context.Context, naturalLanguageText string) (string, error) {
	reqBody, err := json.Marshal(struct {
		Model          string            `json:"model"`
		Messages       []chatMessage     `json:"messages"`
		ResponseFormat map[string]string `json:"response_format"`
	}{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: naturalLanguageText},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return "", fmt.Errorf("%w: encode request", domain.ErrAISuggestionFailed)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("%w: build request", domain.ErrAISuggestionFailed)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: request failed", domain.ErrAISuggestionFailed)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read response", domain.ErrAISuggestionFailed)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: unexpected status %d", domain.ErrAISuggestionFailed, resp.StatusCode)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%w: malformed response", domain.ErrAISuggestionFailed)
	}

	return parsed.Choices[0].Message.Content, nil
}
