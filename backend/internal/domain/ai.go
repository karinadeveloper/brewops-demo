package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrAISuggestionFailed wraps any failure talking to the external AI
// provider (timeout, non-2xx response, malformed output) so handlers can
// report one clear, generic error to the client without leaking transport
// details or provider response bodies.
var ErrAISuggestionFailed = errors.New("failed to get AI suggestions")

// InventorySuggestionClient is the boundary to the external LLM used by
// POST /inventory/suggest. The concrete implementation (ai.OpenAIClient)
// calls OpenAI's API; tests substitute a mock so no test ever makes a real
// network call or spends real money.
type InventorySuggestionClient interface {
	// Suggest sends naturalLanguageText to the model and returns the raw
	// JSON string it replied with. The caller (service) owns
	// parsing/validating that JSON — this boundary only owns the network
	// call and its timeout.
	Suggest(ctx context.Context, naturalLanguageText string) (string, error)
}

// SuggestedMovement is one AI-suggested inventory movement, always subject
// to explicit user confirmation via POST /inventory/movements before
// anything is persisted — the AI itself never writes to the database.
type SuggestedMovement struct {
	// ProductMentioned is the product name/phrase as the AI extracted it
	// from the input text (e.g. "naranjas").
	ProductMentioned string
	// ProductMatch is the best fuzzy-matched existing product, or nil when
	// no sufficiently similar product exists — the frontend must then let
	// the user resolve it manually.
	ProductMatch *uuid.UUID
	// Type is one of PURCHASE, ADJUSTMENT_IN, ADJUSTMENT_OUT — mirrors the
	// manual movement types POST /inventory/movements accepts. Never SALE.
	Type          string
	Quantity      int32
	UnitCostCents int64
}
