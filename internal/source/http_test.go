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

func TestSecureDownloadWantsTLSExceptOnLoopback(t *testing.T) {
	t.Parallel()
	for link, want := range map[string]bool{
		"https://cdn.example/a.zip":  true,
		"http://cdn.example/a.zip":   false,
		"http://127.0.0.1:8080/a":    true,
		"http://localhost/a":         true,
		"ftp://cdn.example/a":        false,
		"http://localhost.evil.io/a": false,
		"":                           false,
	} {
		if got := SecureDownload(link); got != want {
			t.Errorf("SecureDownload(%q) = %v, want %v", link, got, want)
		}
	}
}
