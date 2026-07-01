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
	for part := range strings.SplitSeq(acceptLanguage, ",") {
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

	// --- auth ---
	CodeAuthInvalidCredentials     = "auth.invalid_credentials"
	CodeAuthSetupLocked            = "auth.setup_locked"
	CodeAuthInvalidCurrentPassword = "auth.invalid_current_password"

	// --- project ---
	CodeProjectNotFound  = "project.not_found"
	CodeProjectSlugTaken = "project.slug_taken"

	// --- apikey ---
	CodeAPIKeyNotFound = "apikey.not_found"

	// --- pricing ---
	CodePricingNotFound     = "pricing.not_found"
	CodePricingDuplicate    = "pricing.duplicate"
	CodePricingInvalidPrice = "pricing.invalid_price"

	// --- event ---
	CodeEventInvalidTokens    = "event.invalid_tokens"
	CodeEventInvalidLatency   = "event.invalid_latency"
	CodeEventInvalidStatus    = "event.invalid_status"
	CodeEventFutureTimestamp  = "event.future_timestamp"
	CodeEventBackdateExceeded = "event.backdate_exceeded"
	CodeEventStringTooLong    = "event.string_too_long"
	CodeEventMetadataTooLarge = "event.metadata_too_large"
	CodeEventNotFound         = "event.not_found"

	// --- analytics ---
	CodeAnalyticsInvalidInterval = "analytics.invalid_interval"
)

// catalog maps an error code to its localized messages.
var catalog = map[string]map[Lang]string{
	// --- cross-cutting ---
	CodeInternal:     {EN: "Internal server error", ID: "Terjadi kesalahan pada server"},
	CodeValidation:   {EN: "Validation failed", ID: "Validasi gagal"},
	CodeUnauthorized: {EN: "Authentication required", ID: "Perlu autentikasi"},
	CodeForbidden:    {EN: "Access denied", ID: "Akses ditolak"},
	CodeNotFound:     {EN: "Resource not found", ID: "Data tidak ditemukan"},
	// --- auth ---
	CodeAuthInvalidCredentials:     {EN: "Invalid email or password", ID: "Email atau kata sandi salah"},
	CodeAuthSetupLocked:            {EN: "Setup is already completed", ID: "Setup sudah pernah dilakukan"},
	CodeAuthInvalidCurrentPassword: {EN: "Current password is incorrect", ID: "Kata sandi saat ini salah"},
	// --- project ---
	CodeProjectNotFound:  {EN: "Project not found", ID: "Proyek tidak ditemukan"},
	CodeProjectSlugTaken: {EN: "A project with this name already exists", ID: "Proyek dengan nama ini sudah ada"},
	// --- apikey ---
	CodeAPIKeyNotFound: {EN: "API key not found", ID: "Kunci API tidak ditemukan"},
	// --- pricing ---
	CodePricingNotFound:     {EN: "Pricing rule not found", ID: "Aturan harga tidak ditemukan"},
	CodePricingDuplicate:    {EN: "A pricing rule for this provider/model already exists", ID: "Aturan harga untuk provider/model ini sudah ada"},
	CodePricingInvalidPrice: {EN: "Price must not be negative", ID: "Harga tidak boleh negatif"},
	// --- event ---
	CodeEventInvalidTokens:    {EN: "Token counts are out of range", ID: "Jumlah token di luar rentang"},
	CodeEventInvalidLatency:   {EN: "Latency must not be negative", ID: "Latensi tidak boleh negatif"},
	CodeEventInvalidStatus:    {EN: "Status code must be between 100 and 599", ID: "Kode status harus antara 100 dan 599"},
	CodeEventFutureTimestamp:  {EN: "request_started_at must not be in the future", ID: "request_started_at tidak boleh di masa depan"},
	CodeEventBackdateExceeded: {EN: "request_started_at is older than the allowed window", ID: "request_started_at lebih lama dari jendela yang diizinkan"},
	CodeEventStringTooLong:    {EN: "A field is too long", ID: "Salah satu field terlalu panjang"},
	CodeEventMetadataTooLarge: {EN: "metadata exceeds the 8KB limit", ID: "metadata melebihi batas 8KB"},
	CodeEventNotFound:         {EN: "Event not found", ID: "Event tidak ditemukan"},
	// --- analytics ---
	CodeAnalyticsInvalidInterval: {EN: "interval must be one of hour, day, or week", ID: "interval harus salah satu dari hour, day, atau week"},
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
