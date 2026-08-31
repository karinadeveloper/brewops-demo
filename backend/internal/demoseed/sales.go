package demoseed

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// businessHoursLocation loads once; falls back to UTC if the tzdata
// database isn't available rather than failing the whole reset over a
// cosmetic detail of when, within a day, a demo sale looks like it happened.
var businessHoursLocation = loadBusinessHoursLocation()

func loadBusinessHoursLocation() *time.Location {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		return time.UTC
	}
	return loc
}

type saleItemSeed struct {
	ProductIndex   int
	Quantity       int32
	UnitPriceCents int64
}

type saleSeed struct {
	CreatedAt     time.Time
	PaymentMethod string
	Items         []saleItemSeed
	TotalCents    int64
}

// buildSales generates 30-50 sales (count itself derived from rng, so it's
// stable across runs given the same seed) spread across
// [windowStart, now), timestamped within plausible business hours
// (08:00-20:59 America/Mexico_City) so the dashboard's revenue-over-time
// chart has real day-to-day shape rather than a flat line. Each sale has
// 1-3 line items from distinct products at 1-4 units each, priced at that
// product's current sale price, with payment method skewed 60% CASH / 40%
// TRANSFER — plausible for a small storefront that takes some transfers but
// mostly cash.
func buildSales(rng *rand.Rand, products []productSeed, windowStart, now time.Time) []saleSeed {
	const minSales, maxSales = 30, 50
	count := minSales + rng.Intn(maxSales-minSales+1)

	windowDays := int(now.Sub(windowStart).Hours() / 24)
	if windowDays < 1 {
		windowDays = 1
	}

	sales := make([]saleSeed, 0, count)
	for i := 0; i < count; i++ {
		localDay := windowStart.In(businessHoursLocation).AddDate(0, 0, rng.Intn(windowDays))
		hour := 8 + rng.Intn(13) // 08:00-20:59
		minute := rng.Intn(60)
		second := rng.Intn(60)
		createdAt := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), hour, minute, second, 0, businessHoursLocation).UTC()

		itemCount := rng.Intn(3) + 1
		if itemCount > len(products) {
			itemCount = len(products)
		}
		chosen := make(map[int]bool, itemCount)
		items := make([]saleItemSeed, 0, itemCount)
		var total int64
		for len(items) < itemCount {
			idx := rng.Intn(len(products))
			if chosen[idx] {
				continue
			}
			chosen[idx] = true
			qty := int32(rng.Intn(4) + 1)
			unitPrice := products[idx].SalePriceCents
			items = append(items, saleItemSeed{ProductIndex: idx, Quantity: qty, UnitPriceCents: unitPrice})
			total += unitPrice * int64(qty)
		}

		paymentMethod := "CASH"
		if rng.Float64() >= 0.6 {
			paymentMethod = "TRANSFER"
		}

		sales = append(sales, saleSeed{CreatedAt: createdAt, PaymentMethod: paymentMethod, Items: items, TotalCents: total})
	}

	// Chronological order is cosmetic only (no correctness depends on it)
	// but makes the inserted history read naturally in any tool that lists
	// rows in insertion order.
	sort.Slice(sales, func(i, j int) bool { return sales[i].CreatedAt.Before(sales[j].CreatedAt) })

	return sales
}

func insertSales(ctx context.Context, tx pgx.Tx, sales []saleSeed, productIDs []uuid.UUID, adminID uuid.UUID) error {
	const saleQuery = `
		INSERT INTO sales (total_cents, payment_method, created_at, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	const itemQuery = `
		INSERT INTO sale_items (sale_id, product_id, quantity, unit_price_cents, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	for _, s := range sales {
		var saleID pgtype.UUID
		if err := tx.QueryRow(ctx, saleQuery, s.TotalCents, s.PaymentMethod, s.CreatedAt, toPgUUID(adminID)).Scan(&saleID); err != nil {
			return fmt.Errorf("insert sale: %w", err)
		}
		for _, item := range s.Items {
			if _, err := tx.Exec(ctx, itemQuery, saleID, toPgUUID(productIDs[item.ProductIndex]), item.Quantity, item.UnitPriceCents, s.CreatedAt); err != nil {
				return fmt.Errorf("insert sale item: %w", err)
			}
		}
	}
	return nil
}

// insertStockMovements writes the historical inventory_movements audit
// trail: one SALE movement per sale line item (dated at that sale's
// timestamp), plus one PURCHASE movement per product dated at windowStart
// representing the stock brought in before the demo's sales history begins.
// Each product's PURCHASE quantity is sized as (its final TargetStock +
// everything sold from it during the window), clamped to at least 1, so the
// movement history stays a plausible narrative for that product's current
// stock — purchased minus sold roughly reconciles to what's on hand. Exact
// reconciliation isn't a system invariant (current_stock is a stored
// column, not derived from movements, exactly as in the real product's
// application logic), so this is a realism choice, not a correctness one.
func insertStockMovements(ctx context.Context, tx pgx.Tx, products []productSeed, productIDs []uuid.UUID, sales []saleSeed, adminID uuid.UUID, windowStart time.Time) error {
	const movementQuery = `
		INSERT INTO inventory_movements (product_id, type, quantity, reason, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`

	totalSold := make([]int32, len(products))
	for _, s := range sales {
		for _, item := range s.Items {
			totalSold[item.ProductIndex] += item.Quantity
			if _, err := tx.Exec(ctx, movementQuery, toPgUUID(productIDs[item.ProductIndex]), domain.MovementTypeSale, item.Quantity, nil, s.CreatedAt, toPgUUID(adminID)); err != nil {
				return fmt.Errorf("insert sale movement: %w", err)
			}
		}
	}

	const purchaseReason = "Stock inicial de la demo"
	for i, p := range products {
		purchased := p.TargetStock + totalSold[i]
		if purchased < 1 {
			purchased = 1
		}
		if _, err := tx.Exec(ctx, movementQuery, toPgUUID(productIDs[i]), domain.MovementTypePurchase, purchased, purchaseReason, windowStart, toPgUUID(adminID)); err != nil {
			return fmt.Errorf("insert purchase movement for %q: %w", p.Name, err)
		}
	}
	return nil
}
