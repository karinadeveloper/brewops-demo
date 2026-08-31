package demoseed

import (
	"math/rand"
	"testing"
	"time"
)

func TestDemoProducts_BetweenFifteenAndTwentyWithAtLeastThreeLowStock(t *testing.T) {
	// Arrange
	products := demoProducts(nil)

	// Act
	lowStock := 0
	for _, p := range products {
		if p.TargetStock <= p.MinStock {
			lowStock++
		}
	}

	// Assert
	if len(products) < 15 || len(products) > 20 {
		t.Fatalf("expected 15-20 products, got %d", len(products))
	}
	if lowStock < 2 {
		t.Errorf("expected at least 2-3 deliberately low-stock products, got %d", lowStock)
	}
}

func TestDemoProducts_AssignsImagesRoundRobin(t *testing.T) {
	// Arrange
	images := []string{"a.png", "b.png"}

	// Act
	products := demoProducts(images)

	// Assert
	for i, p := range products {
		if p.ImageURL == nil {
			t.Fatalf("product %d: expected an image URL, got nil", i)
		}
		want := images[i%len(images)]
		if *p.ImageURL != want {
			t.Errorf("product %d: expected image %q, got %q", i, want, *p.ImageURL)
		}
	}
}

func TestDemoProducts_NoImages_LeavesImageURLNil(t *testing.T) {
	// Arrange & Act
	products := demoProducts(nil)

	// Assert
	for i, p := range products {
		if p.ImageURL != nil {
			t.Errorf("product %d: expected nil image URL, got %q", i, *p.ImageURL)
		}
	}
}

func TestBuildSales_SameSeed_ProducesIdenticalResult(t *testing.T) {
	// Arrange
	products := demoProducts(nil)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	windowStart := now.Add(-salesWindow)

	// Act
	first := buildSales(rand.New(rand.NewSource(seedRandSeed)), products, windowStart, now)
	second := buildSales(rand.New(rand.NewSource(seedRandSeed)), products, windowStart, now)

	// Assert
	if len(first) != len(second) {
		t.Fatalf("expected identical sale counts across runs, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if !first[i].CreatedAt.Equal(second[i].CreatedAt) || first[i].TotalCents != second[i].TotalCents || len(first[i].Items) != len(second[i].Items) {
			t.Fatalf("sale %d differs between runs: %+v vs %+v", i, first[i], second[i])
		}
	}
}

func TestBuildSales_CountWithinDocumentedRange(t *testing.T) {
	// Arrange
	products := demoProducts(nil)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	windowStart := now.Add(-salesWindow)

	// Act
	sales := buildSales(rand.New(rand.NewSource(seedRandSeed)), products, windowStart, now)

	// Assert
	if len(sales) < 30 || len(sales) > 50 {
		t.Fatalf("expected 30-50 sales, got %d", len(sales))
	}
}

func TestBuildSales_EveryItemPricedFromItsProductAndPositiveQuantity(t *testing.T) {
	// Arrange
	products := demoProducts(nil)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	windowStart := now.Add(-salesWindow)

	// Act
	sales := buildSales(rand.New(rand.NewSource(seedRandSeed)), products, windowStart, now)

	// Assert
	for _, s := range sales {
		var total int64
		for _, item := range s.Items {
			if item.Quantity < 1 {
				t.Fatalf("expected a positive quantity, got %d", item.Quantity)
			}
			if item.UnitPriceCents != products[item.ProductIndex].SalePriceCents {
				t.Fatalf("expected unit price to match the product's sale price")
			}
			total += item.UnitPriceCents * int64(item.Quantity)
		}
		if total != s.TotalCents {
			t.Fatalf("expected TotalCents %d to equal the sum of its items, got %d", total, s.TotalCents)
		}
	}
}

func TestDemoMarketingAssets_NoImages_ReturnsEmpty(t *testing.T) {
	// Arrange & Act
	assets := demoMarketingAssets(nil)

	// Assert
	if len(assets) != 0 {
		t.Errorf("expected no marketing assets without seed images, got %d", len(assets))
	}
}

func TestDemoMarketingAssets_WithImages_ReturnsThreeToFour(t *testing.T) {
	// Arrange & Act
	assets := demoMarketingAssets([]string{"a.png"})

	// Assert
	if len(assets) < 3 || len(assets) > 4 {
		t.Errorf("expected 3-4 marketing assets, got %d", len(assets))
	}
}
