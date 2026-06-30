package project

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My Project", "my-project"},
		{"  Hello!!  World  ", "hello-world"},
		{"already-a-slug", "already-a-slug"},
		{"Trailing!!!", "trailing"},
		{"a---b", "a-b"},
		{"Café 123", "caf-123"},
		{"", "project"},
		{"!!!", "project"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := Slugify(c.in); got != c.want {
				t.Fatalf("Slugify(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
