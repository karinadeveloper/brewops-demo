package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// aiRequestTimeout bounds every call to the external AI provider —
// CLAUDE.md requires a clear error instead of a hung request when OpenAI is
// slow or unresponsive.
const aiRequestTimeout = 15 * time.Second

// productMatchThreshold is the minimum similarity score (0-1) a product
// name must reach to be offered as a fuzzy match. Below this, the
// suggestion is left for the user to resolve manually in the frontend.
const productMatchThreshold = 0.6

// maxProductsForMatching bounds how many active products are loaded for
// fuzzy matching against AI-mentioned product names. A small juice/beverage
// business's catalog is realistically a few dozen items, so reusing
// ProductRepository.List with a generous fixed limit avoids adding a new
// unbounded "list everything" repository method just for this.
const maxProductsForMatching = 1000

// InventorySuggestionService implements POST /inventory/suggest: it asks
// the AI provider to extract structured movements from natural language,
// then fuzzy-matches each mentioned product name against the real catalog.
// It never writes anything — see domain.SuggestedMovement.
type InventorySuggestionService struct {
	ai       domain.InventorySuggestionClient
	products domain.ProductRepository
}

func NewInventorySuggestionService(ai domain.InventorySuggestionClient, products domain.ProductRepository) *InventorySuggestionService {
	return &InventorySuggestionService{ai: ai, products: products}
}

type aiSuggestResponse struct {
	Movements []struct {
		ProductName   string `json:"product_name"`
		Type          string `json:"type"`
		Quantity      int32  `json:"quantity"`
		UnitCostCents int64  `json:"unit_cost_cents"`
	} `json:"movements"`
}

// normalizeSuggestedMovementType keeps only the manual movement types
// POST /inventory/movements actually accepts. Anything else the model
// returns (including, deliberately, "SALE") is discarded to "" so the
// frontend must have the user pick a type manually rather than silently
// forwarding a value that would just be rejected — or worse, misapplied —
// downstream.
func normalizeSuggestedMovementType(t string) string {
	switch t {
	case domain.MovementTypePurchase, domain.MovementTypeAdjustmentIn, domain.MovementTypeAdjustmentOut:
		return t
	default:
		return ""
	}
}

// normalizeForMatch lowercases and trims for case/whitespace-insensitive
// comparison. Pure and tested.
func normalizeForMatch(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// levenshtein computes the classic edit distance between two strings.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// similarity returns a 0-1 score (1 = identical) derived from Levenshtein
// distance normalized by the longer string's length.
func similarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1
	}
	return 1 - float64(levenshtein(a, b))/float64(maxLen)
}

// stripTrailingS is a deliberately crude Spanish singularizer — good enough
// to turn "naranjas" into "naranja" for matching purposes, not a real
// linguistic tool. Short words are left alone so it doesn't mangle
// unrelated 2-3 letter tokens.
func stripTrailingS(s string) string {
	if len(s) > 3 && strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

func containsEitherDirection(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// bestProductMatch finds the product whose name is most similar to
// mentioned, requiring at least productMatchThreshold similarity. Two
// boosts handle the common real-world cases plain edit-distance scores
// poorly despite being obvious matches to a person:
//   - A substring match (either direction, singular or plural) — a short
//     generic mention ("naranjas") against a longer specific product name
//     ("Jugo de naranja 1L").
//   - Any single word of the product name matching (or singularizing to)
//     the mention — handles the same case when word order or extra words
//     defeat a plain substring check.
//
// Pure and exhaustively tested — this is the core matching logic the AI
// feature depends on.
func bestProductMatch(mentioned string, products []domain.Product) *domain.Product {
	normMentioned := normalizeForMatch(mentioned)
	if normMentioned == "" {
		return nil
	}
	mentionedSingular := stripTrailingS(normMentioned)

	var best *domain.Product
	bestScore := 0.0
	for i := range products {
		normName := normalizeForMatch(products[i].Name)
		score := similarity(normMentioned, normName)

		if containsEitherDirection(normName, normMentioned) || containsEitherDirection(normName, mentionedSingular) {
			score = math.Max(score, 0.75)
		} else {
			for _, word := range strings.Fields(normName) {
				if word == normMentioned || word == mentionedSingular || stripTrailingS(word) == mentionedSingular {
					score = math.Max(score, 0.85)
					break
				}
			}
		}

		if score > bestScore {
			bestScore = score
			best = &products[i]
		}
	}
	if bestScore < productMatchThreshold {
		return nil
	}
	return best
}

// Suggest asks the AI provider to extract movements from text and
// fuzzy-matches each one against the active product catalog. Nothing here
// is ever persisted — the caller must submit each confirmed suggestion
// through the existing POST /inventory/movements.
func (s *InventorySuggestionService) Suggest(ctx context.Context, text string) ([]domain.SuggestedMovement, error) {
	ctx, cancel := context.WithTimeout(ctx, aiRequestTimeout)
	defer cancel()

	raw, err := s.ai.Suggest(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAISuggestionFailed, err)
	}

	var parsed aiSuggestResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("%w: could not parse AI response", domain.ErrAISuggestionFailed)
	}

	products, err := s.products.List(ctx, domain.ListProductsParams{Limit: maxProductsForMatching})
	if err != nil {
		return nil, fmt.Errorf("inventory suggestion: list products: %w", err)
	}

	suggestions := make([]domain.SuggestedMovement, len(parsed.Movements))
	for i, m := range parsed.Movements {
		var productMatch *uuid.UUID
		if match := bestProductMatch(m.ProductName, products); match != nil {
			id := match.ID
			productMatch = &id
		}
		suggestions[i] = domain.SuggestedMovement{
			ProductMentioned: m.ProductName,
			ProductMatch:     productMatch,
			Type:             normalizeSuggestedMovementType(m.Type),
			Quantity:         m.Quantity,
			UnitCostCents:    m.UnitCostCents,
		}
	}
	return suggestions, nil
}
