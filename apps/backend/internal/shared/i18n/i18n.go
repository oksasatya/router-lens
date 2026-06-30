// Package i18n resolves the request language and maps error codes to localized
// messages. It is pure (no Echo import) so domain/shared code can depend on it.
package i18n

import "strings"

type Lang string

const (
	EN Lang = "en"
	ID Lang = "id"
)

const (
	Default    = EN
	ContextKey = "lang"
)

func supported(l Lang) bool { return l == EN || l == ID }

// Resolve picks the language from the Accept-Language header — the first
// supported tag wins, otherwise the default. Header-driven content negotiation
// per RFC 7231; no query/cookie override in v0.1.
func Resolve(acceptLanguage string) Lang {
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.ToLower(strings.TrimSpace(part))
		if i := strings.IndexAny(tag, ";-"); i >= 0 {
			tag = tag[:i]
		}
		if l := Lang(tag); supported(l) {
			return l
		}
	}
	return Default
}

// Error codes are the stable contract the frontend switches on. Cross-cutting
// codes are flat; feature domains add namespaced codes (e.g.
// "auth.invalid_credentials", "pricing.duplicate_rule") as a new const + catalog
// section, so a typo is caught at compile time rather than at runtime.
const (
	CodeInternal     = "internal_error"
	CodeValidation   = "validation_failed"
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
	CodeNotFound     = "not_found"
)

// catalog maps an error code to its localized messages.
var catalog = map[string]map[Lang]string{
	// --- cross-cutting ---
	CodeInternal:     {EN: "Internal server error", ID: "Terjadi kesalahan pada server"},
	CodeValidation:   {EN: "Validation failed", ID: "Validasi gagal"},
	CodeUnauthorized: {EN: "Authentication required", ID: "Perlu autentikasi"},
	CodeForbidden:    {EN: "Access denied", ID: "Akses ditolak"},
	CodeNotFound:     {EN: "Resource not found", ID: "Data tidak ditemukan"},
	// --- feature domains append sections below: auth.*, project.*, apikey.*, event.*, pricing.* ---
}

// Message returns the localized message for code, falling back to the default
// language and finally to the provided fallback string.
func Message(code string, lang Lang, fallback string) string {
	byLang, ok := catalog[code]
	if !ok {
		return fallback
	}
	if msg, ok := byLang[lang]; ok {
		return msg
	}
	if msg, ok := byLang[Default]; ok {
		return msg
	}
	return fallback
}
