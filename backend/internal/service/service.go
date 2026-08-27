// Package service contains business logic: stock validation, pricing
// arithmetic in cents, optimistic concurrency checks, and orchestration
// between repositories. Services depend on repository interfaces (not
// concrete implementations) so they can be tested with mocks.
package service
