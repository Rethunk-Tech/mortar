package source

import "testing"

func TestParseHTTPSURL(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"https://example.com/a", true},
		{"  https://example.com/a \n", true},
		{"http://example.com/a", false},
		{"https://user:pw@example.com/a", false},
		{"https:///a", false},
		{"https:a", false},
		{"%zz", false},
	} {
		if _, ok := ParseHTTPSURL(c.in); ok != c.want {
			t.Errorf("ParseHTTPSURL(%q) = %v, want %v", c.in, ok, c.want)
		}
	}
}
