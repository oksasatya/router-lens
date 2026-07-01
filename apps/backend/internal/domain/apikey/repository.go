package apikey

import "context"

// APIKeyRepository is the port for persisting and querying API keys.
type APIKeyRepository interface {
	// Create inserts k, setting ID + CreatedAt.
	Create(ctx context.Context, k *APIKey) error
	ListByProject(ctx context.Context, projectID string) ([]*APIKey, error)
	// Revoke soft-deletes by setting revoked_at = now(). Returns ErrNotFound
	// when no row matched the id.
	Revoke(ctx context.Context, id string) error
	// FindByHash resolves a key by its sha256 hash. Returns ErrNotFound when absent.
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
	// TouchLastUsed sets last_used_at = now() for the key. Best-effort; a missing
	// row is not an error here (the caller already authenticated the key).
	TouchLastUsed(ctx context.Context, id string) error
}
