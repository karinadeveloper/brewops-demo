package demoseed

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// toPgUUID and fromPgUUID mirror internal/repository/convert.go's
// toUUID/fromUUID: pgx v5 only auto-encodes/scans a handful of Go builtins
// against a "uuid" column (see pgtype.UUIDCodec) — a bare
// github.com/google/uuid.UUID satisfies neither UUIDValuer (encode) nor
// UUIDScanner (scan) despite sharing the same [16]byte layout, so every
// value crossing the pgx boundary in this package goes through pgtype.UUID
// explicitly rather than relying on an implicit conversion that doesn't
// exist.
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}
