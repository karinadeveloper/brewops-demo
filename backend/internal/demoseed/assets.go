package demoseed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

var seedImageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

// copySeedAssets copies every image in srcDir into dstDir (creating it if
// needed) and returns the fetchable "<publicBaseURL>/local-storage/<file>"
// URL for each, sorted by filename for determinism. An empty srcDir is the
// deliberate "no seed images configured" case and is not an error — products
// and marketing assets simply get no image_url. But a *non-empty* srcDir
// that doesn't exist on disk is a misconfiguration, not that case — every
// real caller (cmd/seed, cmd/server) always passes a hardcoded non-empty
// path, so a missing directory there means the deployment forgot to ship
// backend/seed-assets/ (e.g. a Docker image that only copies the compiled
// binary). Silently returning zero images previously masked that as a
// suspiciously-low-but-successful reset count instead of a clear failure.
func copySeedAssets(srcDir, dstDir, publicBaseURL string) ([]string, error) {
	if srcDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(srcDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("seed assets dir %q does not exist (was it copied into the deployment image/build context?): %w", srcDir, err)
	}
	if err != nil {
		return nil, fmt.Errorf("read seed assets dir %q: %w", srcDir, err)
	}

	var filenames []string
	for _, entry := range entries {
		if entry.IsDir() || !seedImageExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		filenames = append(filenames, entry.Name())
	}
	sort.Strings(filenames)

	if len(filenames) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, fmt.Errorf("create local storage dir %q: %w", dstDir, err)
	}

	urls := make([]string, 0, len(filenames))
	for _, filename := range filenames {
		data, err := os.ReadFile(filepath.Join(srcDir, filename))
		if err != nil {
			return nil, fmt.Errorf("read seed asset %q: %w", filename, err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, filename), data, 0o644); err != nil {
			return nil, fmt.Errorf("write seed asset %q: %w", filename, err)
		}
		urls = append(urls, fmt.Sprintf("%s/local-storage/%s", publicBaseURL, filename))
	}
	return urls, nil
}

type marketingAssetSeed struct {
	Name     string
	Type     string
	ImageURL string
}

// demoMarketingAssets returns a handful of example gallery entries built
// from whatever seed images are available. marketing_assets.image_url is
// NOT NULL, so with no seed images at all there is simply nothing to seed
// here.
func demoMarketingAssets(imageURLs []string) []marketingAssetSeed {
	if len(imageURLs) == 0 {
		return nil
	}

	definitions := []struct {
		Name string
		Type string
	}{
		{Name: "Promoción de temporada: 2x1 en jugos", Type: domain.MarketingAssetTypePromotion},
		{Name: "Foto de catálogo: línea de aguas", Type: domain.MarketingAssetTypeProduct},
		{Name: "Anuncio de fin de semana", Type: domain.MarketingAssetTypePromotion},
		{Name: "Foto de catálogo: refrescos", Type: domain.MarketingAssetTypeProduct},
	}

	assets := make([]marketingAssetSeed, len(definitions))
	for i, d := range definitions {
		assets[i] = marketingAssetSeed{Name: d.Name, Type: d.Type, ImageURL: imageURLs[i%len(imageURLs)]}
	}
	return assets
}

func insertMarketingAssets(ctx context.Context, tx pgx.Tx, assets []marketingAssetSeed, adminID uuid.UUID) error {
	const query = `
		INSERT INTO marketing_assets (name, image_url, type, created_by)
		VALUES ($1, $2, $3, $4)`

	for _, a := range assets {
		if _, err := tx.Exec(ctx, query, a.Name, a.ImageURL, a.Type, toPgUUID(adminID)); err != nil {
			return fmt.Errorf("insert marketing asset %q: %w", a.Name, err)
		}
	}
	return nil
}
