package project

import "strings"

const fallbackSlug = "project"

// Slugify converts a display name to a URL-safe slug: lowercase, ASCII
// alphanumeric runs joined by single hyphens, no leading/trailing hyphen.
// Non-ASCII characters are dropped. Empty or all-punctuation input yields
// "project". ponytail: ASCII-only is sufficient for a dev tool; widen if a
// real consumer needs unicode slugs.
func Slugify(name string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
			continue
		}
		pendingHyphen = true
	}
	if b.Len() == 0 {
		return fallbackSlug
	}
	return b.String()
}
