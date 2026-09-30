//go:build !windows

package backdrop

import (
	"context"
	"errors"
	"testing"
)

func fake(values map[string]string) Runner {
	return func(_ context.Context, args ...string) (string, error) {
		v, ok := values[args[len(args)-1]]
		if !ok {
			return "", errors.New("no such key")
		}
		return v + "\n", nil
	}
}

func TestGnomeWallpaper(t *testing.T) {
	cases := []struct {
		name string
		gs   map[string]string
		want string
	}{
		{"dark scheme uses the dark uri", map[string]string{"color-scheme": "'prefer-dark'", "picture-uri": "'file:///l.jpg'", "picture-uri-dark": "'file:///d%20k.jpg'"}, "/d k.jpg"},
		{"light scheme uses the plain uri", map[string]string{"color-scheme": "'default'", "picture-uri": "'file:///l.jpg'", "picture-uri-dark": "'file:///d.jpg'"}, "/l.jpg"},
		{"missing scheme uses the plain uri", map[string]string{"picture-uri": "'file:///l.jpg'"}, "/l.jpg"},
		{"remote uri is refused", map[string]string{"picture-uri": "'https://x/l.jpg'"}, ""},
		{"gsettings unavailable", map[string]string{}, ""},
	}
	for _, c := range cases {
		if got := gnomeWallpaper(fake(c.gs)); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
