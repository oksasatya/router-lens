package dto

import apikeydomain "router-lens/internal/domain/apikey"

// APIKeyRequest is the create payload for an API key.
type APIKeyRequest struct {
	Name string `json:"name" validate:"required,max=120"`
}

// APIKeyResponse is the wire shape of an API key (never carries the secret).
type APIKeyResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	LastUsedAt *string `json:"last_used_at"`
	RevokedAt  *string `json:"revoked_at"`
	CreatedAt  string  `json:"created_at"`
}

// APIKeyCreatedResponse is returned ONCE on creation — it carries the plaintext
// key, which is never persisted and never returned again.
type APIKeyCreatedResponse struct {
	APIKeyResponse
	Key string `json:"key"`
}

// FromAPIKey maps a domain API key to its response shape.
func FromAPIKey(k *apikeydomain.APIKey) APIKeyResponse {
	return APIKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		LastUsedAt: formatNullableTime(k.LastUsedAt),
		RevokedAt:  formatNullableTime(k.RevokedAt),
		CreatedAt:  k.CreatedAt.UTC().Format(timeLayout),
	}
}
