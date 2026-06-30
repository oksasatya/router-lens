package i18n

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		accept string
		want   Lang
	}{
		{"", EN},
		{"id-ID,id;q=0.9,en;q=0.8", ID},
		{"en-US,en;q=0.9", EN},
		{"fr-FR", EN}, // unsupported falls back to default
	}
	for _, tc := range cases {
		if got := Resolve(tc.accept); got != tc.want {
			t.Errorf("Resolve(%q)=%q want %q", tc.accept, got, tc.want)
		}
	}
}

func TestMessage(t *testing.T) {
	if Message("validation_failed", ID, "x") != "Validasi gagal" {
		t.Errorf("ID validation message wrong: %q", Message("validation_failed", ID, "x"))
	}
	if Message("unknown_code", ID, "fallback msg") != "fallback msg" {
		t.Error("unknown code should use fallback")
	}
}
