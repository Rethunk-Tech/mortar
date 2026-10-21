package pack

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGaleCodeFollowsTheRedirectAndReadsTheProfile(t *testing.T) {
	zipped := makeZip(t, map[string]string{exportFile: export})
	mux := http.NewServeMux()
	mux.HandleFunc("/profile/E5RG8F", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cdn/p", http.StatusSeeOther)
	})
	mux.HandleFunc("/cdn/p", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(zipped) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	d, err := GaleCode{URL: srv.URL}.Parse(t.Context(), Input{Text: " E5RG8F\n"})
	if err != nil || d.Name != "Friends" || len(d.Packages) != 2 {
		t.Fatalf("draft = %+v, err = %v", d, err)
	}
	if _, err := (GaleCode{URL: srv.URL}).Parse(t.Context(), Input{Text: "ZZZZZZ"}); err == nil {
		t.Fatal("an unknown code was accepted")
	}
}

func TestGaleCodeDetectsOnlyItsOwnShape(t *testing.T) {
	for text, want := range map[string]bool{
		"E5RG8F": true, "e5rg8f": false, "E5RG8": false, "E5RG8FX": false, "E5RG-8": false, "hello": false,
		"0123abcd-0123-0123-0123-0123456789ab": false, "#r2modman": false,
	} {
		if got := (GaleCode{}).Detect(Input{Text: text}); got != want {
			t.Errorf("Detect(%q) = %v", text, got)
		}
	}
	if (GaleCode{}).Detect(Input{Path: "/x/ABCDEF", Text: "ABCDEF"}) {
		t.Error("a path input was detected")
	}
}

func TestGaleCodeRefusesAMalformedOrOversizedAnswer(t *testing.T) {
	for name, body := range map[string]string{
		"not a zip": "hello",
		"too large": strings.Repeat("x", maxInput+1),
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
		_, err := GaleCode{URL: srv.URL}.Parse(t.Context(), Input{Text: "E5RG8F"})
		srv.Close()
		if err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
