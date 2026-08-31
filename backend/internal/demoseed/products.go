package demoseed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// productSeed is one row of the static demo catalog. Unlike sales, the
// catalog itself is not randomized — a portfolio demo should show the same
// recognizable products on every reset, only the sales history around them
// varies.
type productSeed struct {
	Name           string
	Category       string
	SalePriceCents int64
	CostCents      int64
	MinStock       int32
	// TargetStock is the current_stock the product ends up with after
	// reset. Three entries below deliberately set this at or under
	// MinStock so GET /products/low-stock and the dashboard's low-stock
	// widget have something to show immediately after a fresh reset.
	TargetStock int32
	ImageURL    *string
}

// demoProducts returns the fixed 19-product catalog (juice/water/soda/other
// per CLAUDE.md's domain model), assigning imageURLs to products round-robin
// so every product has a picture whenever seed images are available. Prices
// are plausible MXN retail figures for a small juice/beverage business,
// stored in cents per this project's money-handling rule.
func demoProducts(imageURLs []string) []productSeed {
	products := []productSeed{
		{Name: "Jugo de Naranja 1L", Category: "juice", SalePriceCents: 4500, CostCents: 2600, MinStock: 10, TargetStock: 42},
		{Name: "Jugo de Naranja 500ml", Category: "juice", SalePriceCents: 2800, CostCents: 1600, MinStock: 15, TargetStock: 60},
		{Name: "Jugo de Mango 1L", Category: "juice", SalePriceCents: 4800, CostCents: 2800, MinStock: 10, TargetStock: 35},
		{Name: "Jugo de Manzana 1L", Category: "juice", SalePriceCents: 4600, CostCents: 2700, MinStock: 10, TargetStock: 28},
		// Deliberately low stock: current_stock (2) <= min_stock (8).
		{Name: "Jugo Verde Detox 500ml", Category: "juice", SalePriceCents: 5200, CostCents: 3200, MinStock: 8, TargetStock: 2},
		{Name: "Agua de Jamaica 1L", Category: "juice", SalePriceCents: 3800, CostCents: 2000, MinStock: 12, TargetStock: 40},
		{Name: "Agua de Horchata 1L", Category: "juice", SalePriceCents: 3800, CostCents: 2000, MinStock: 12, TargetStock: 33},
		{Name: "Agua Natural 600ml", Category: "water", SalePriceCents: 1200, CostCents: 500, MinStock: 20, TargetStock: 90},
		{Name: "Agua Natural 1L", Category: "water", SalePriceCents: 1800, CostCents: 800, MinStock: 20, TargetStock: 70},
		{Name: "Agua Mineral 355ml", Category: "water", SalePriceCents: 1500, CostCents: 700, MinStock: 15, TargetStock: 55},
		// Deliberately low stock: current_stock (15) <= min_stock (15).
		{Name: "Agua Mineral 600ml", Category: "water", SalePriceCents: 1900, CostCents: 900, MinStock: 15, TargetStock: 15},
		{Name: "Refresco de Cola 355ml", Category: "soda", SalePriceCents: 1600, CostCents: 800, MinStock: 20, TargetStock: 65},
		{Name: "Refresco de Toronja 355ml", Category: "soda", SalePriceCents: 1600, CostCents: 800, MinStock: 20, TargetStock: 48},
		{Name: "Refresco de Manzana 600ml", Category: "soda", SalePriceCents: 2100, CostCents: 1100, MinStock: 15, TargetStock: 30},
		{Name: "Té Helado de Limón 500ml", Category: "other", SalePriceCents: 2200, CostCents: 1200, MinStock: 12, TargetStock: 25},
		{Name: "Café Frío 350ml", Category: "other", SalePriceCents: 3200, CostCents: 1800, MinStock: 10, TargetStock: 20},
		// Deliberately low stock: out of stock entirely (0 <= min_stock 10).
		{Name: "Bebida Energética 473ml", Category: "other", SalePriceCents: 3500, CostCents: 2000, MinStock: 10, TargetStock: 0},
		{Name: "Licuado de Fresa 500ml", Category: "other", SalePriceCents: 3800, CostCents: 2200, MinStock: 8, TargetStock: 18},
		{Name: "Smoothie de Piña 500ml", Category: "other", SalePriceCents: 4000, CostCents: 2400, MinStock: 8, TargetStock: 14},
	}

	if len(imageURLs) > 0 {
		for i := range products {
			url := imageURLs[i%len(imageURLs)]
			products[i].ImageURL = &url
		}
	}

	return products
}

// insertProducts persists products and returns their generated IDs in the
// same order, so callers can look up "the UUID for demoProducts()[i]" by
// index when building sales and movements against them.
func insertProducts(ctx context.Context, tx pgx.Tx, products []productSeed, adminID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		INSERT INTO products (name, category, sale_price_cents, cost_cents, current_stock, min_stock, image_url, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	ids := make([]uuid.UUID, len(products))
	for i, p := range products {
		var id pgtype.UUID
		if err := tx.QueryRow(ctx, query, p.Name, p.Category, p.SalePriceCents, p.CostCents, p.TargetStock, p.MinStock, p.ImageURL, toPgUUID(adminID)).Scan(&id); err != nil {
			return nil, fmt.Errorf("insert product %q: %w", p.Name, err)
		}
		ids[i] = fromPgUUID(id)
	}
	return ids, nil
}
