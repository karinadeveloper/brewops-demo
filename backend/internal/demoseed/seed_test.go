package demoseed

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
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

func TestDemoProducts_MatchesImagesByThematicFilename(t *testing.T) {
	// Arrange: only a subset of real seed-assets filenames is available.
	images := []string{
		"http://localhost:8080/local-storage/jugo-naranja.png",
		"http://localhost:8080/local-storage/jugo-mango.png",
		"http://localhost:8080/local-storage/refresco-cola.png",
	}

	// Act
	products := demoProducts(images)

	// Assert: each product gets exactly the file that matches its own
	// flavor/drink type — never an arbitrary or round-robin one — and
	// products whose matching file isn't in the fixture get no image.
	for _, p := range products {
		switch p.ImageFile {
		case "jugo-naranja.png", "jugo-mango.png", "refresco-cola.png":
			if p.ImageURL == nil || !strings.HasSuffix(*p.ImageURL, p.ImageFile) {
				t.Errorf("product %q: expected image ending in %q, got %v", p.Name, p.ImageFile, p.ImageURL)
			}
		default:
			if p.ImageURL != nil {
				t.Errorf("product %q: expected nil image URL (its seed image %q isn't in the fixture), got %q", p.Name, p.ImageFile, *p.ImageURL)
			}
		}
	}
}

// TestDemoProducts_AllProductsMatchWithFullAssetSet guards against the
// products.go catalog and backend/seed-assets/ drifting apart — every
// product's ImageFile must resolve to a real file, or a fresh demo reset
// would silently leave that product without a picture.
func TestDemoProducts_AllProductsMatchWithFullAssetSet(t *testing.T) {
	// Arrange
	filenames := []string{
		"jugo-naranja.png", "jugo-mango.png", "jugo-manzana.png", "jugo-verde.png",
		"agua-jamaica.png", "agua-horchata.png", "agua-natural.png", "agua-mineral.png",
		"refresco-cola.png", "refresco-toronja.png", "refresco-manzana.png",
		"te-limon.png", "cafe-frio.png", "bebida-energetica.png", "licuado-fresa.png", "smoothie-pina.png",
	}
	images := make([]string, len(filenames))
	for i, f := range filenames {
		images[i] = "http://localhost:8080/local-storage/" + f
	}

	// Act
	products := demoProducts(images)

	// Assert
	for _, p := range products {
		if p.ImageURL == nil {
			t.Errorf("product %q: expected a themed image (file %q) with the full seed-assets set available, got nil", p.Name, p.ImageFile)
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

func TestCopySeedAssets_EmptySrcDir_ReturnsNoImagesAndNoError(t *testing.T) {
	// Arrange
	dstDir := filepath.Join(t.TempDir(), "local-storage")

	// Act
	urls, err := copySeedAssets("", dstDir, "http://localhost:8080")

	// Assert
	if err != nil {
		t.Fatalf("expected no error for the deliberate not-configured case, got %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("expected no image URLs, got %v", urls)
	}
}

func TestCopySeedAssets_ConfiguredSrcDirMissing_ReturnsClearError(t *testing.T) {
	// Arrange: a non-empty path that does not exist on disk, mirroring a
	// deployment (e.g. a Docker image) that forgot to ship backend/seed-assets/.
	missingSrcDir := filepath.Join(t.TempDir(), "does-not-exist")
	dstDir := filepath.Join(t.TempDir(), "local-storage")

	// Act
	urls, err := copySeedAssets(missingSrcDir, dstDir, "http://localhost:8080")

	// Assert
	if err == nil {
		t.Fatal("expected an error for a configured-but-missing seed assets dir, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected the error to wrap os.ErrNotExist, got %v", err)
	}
	if urls != nil {
		t.Errorf("expected no image URLs on error, got %v", urls)
	}
}

func TestCopySeedAssets_ExistingSrcDir_CopiesImagesAndReturnsURLs(t *testing.T) {
	// Arrange
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "local-storage")
	if err := os.WriteFile(filepath.Join(srcDir, "b.png"), []byte("b"), 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.png"), []byte("a"), 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "notes.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	// Act
	urls, err := copySeedAssets(srcDir, dstDir, "http://localhost:8080")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := []string{"http://localhost:8080/local-storage/a.png", "http://localhost:8080/local-storage/b.png"}
	if len(urls) != len(want) || urls[0] != want[0] || urls[1] != want[1] {
		t.Fatalf("expected %v, got %v", want, urls)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "a.png")); err != nil {
		t.Errorf("expected a.png to be copied into dstDir: %v", err)
	}
}
