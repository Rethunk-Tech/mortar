package browse

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTranslateDescriptionMapsLocaleAndJoinsSegments(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("User-Agent") != "Mortar/test" {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		got, err = url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`[[["Hallo ","Hello "],[ "Welt","world"]],null,"en"]`))
	}))
	t.Cleanup(srv.Close)

	s := &Service{Version: "test", translateHTTP: srv.Client(), translateURL: srv.URL}
	out, err := s.TranslateDescription(t.Context(), "  Hello world  ", "zh-Hans")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hallo Welt" {
		t.Fatalf("got %q", out)
	}
	if got.Get("tl") != "zh-CN" || got.Get("sl") != "auto" || got.Get("q") != "Hello world" || got.Get("client") != "gtx" {
		t.Fatalf("form %v", got)
	}
}

func TestTranslateDescriptionLeavesEmptyTextUnsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("empty text must not call the translator")
	}))
	t.Cleanup(srv.Close)

	s := &Service{translateHTTP: srv.Client(), translateURL: srv.URL}
	out, err := s.TranslateDescription(t.Context(), "  \n", "es")
	if err != nil || out != "" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestGtxLanguageUsesThePlayersMortarLocale(t *testing.T) {
	cases := map[string]string{
		"":        "en",
		"en":      "en",
		"es":      "es",
		"es-AR":   "es",
		"es-PA":   "es",
		"pl":      "pl",
		"pt-BR":   "pt-BR",
		"zh-Hans": "zh-CN",
	}
	for locale, want := range cases {
		if got := gtxLanguage(locale); got != want {
			t.Errorf("gtxLanguage(%q) = %q, want %q", locale, got, want)
		}
	}
}
