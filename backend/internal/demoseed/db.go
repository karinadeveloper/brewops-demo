package demoseed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost matches service.AuthService's own choice — the middle of the
// recommended 10-12 range.
const bcryptCost = 11

// businessTablesInDeleteOrder lists every business-data table in the order
// required by their foreign keys — children before the parents they
// reference. The users table is deliberately absent: the demo admin account
// must survive a reset, never be recreated.
var businessTablesInDeleteOrder = []string{
	"sale_items",
	"sales",
	"inventory_movements",
	"products",
	"marketing_assets",
}

func deleteBusinessData(ctx context.Context, tx pgx.Tx) error {
	for _, table := range businessTablesInDeleteOrder {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("delete from %s: %w", table, err)
		}
	}
	return nil
}

// upsertAdminUser ensures the demo admin account exists with the given
// credentials, without disturbing its id (and therefore every FK the rest
// of the schema may reference it by) across resets.
func upsertAdminUser(ctx context.Context, tx pgx.Tx, email, password string) (uuid.UUID, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("hash admin password: %w", err)
	}

	const query = `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'ADMIN')
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id`

	var id pgtype.UUID
	if err := tx.QueryRow(ctx, query, email, string(hash)).Scan(&id); err != nil {
		return uuid.UUID{}, fmt.Errorf("upsert admin user: %w", err)
	}
	return fromPgUUID(id), nil
}
