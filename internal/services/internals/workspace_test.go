package internals

import "testing"

func TestSameOutputIgnoresWhitespaceAndCase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", "Done. Created docs/api.md", "Done. Created docs/api.md", true},
		{"whitespace and case", "Done.  Created\n\tdocs/api.md", "done. created docs/api.md", true},
		{"different content", "Created docs/api.md", "Created docs/index.md", false},
		{"empty left", "", "anything", false},
		{"empty right", "anything", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sameOutput(tc.a, tc.b); got != tc.want {
				t.Fatalf("sameOutput(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
