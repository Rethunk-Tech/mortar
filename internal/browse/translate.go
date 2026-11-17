package browse

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

const gtxEndpoint = "https://translate.googleapis.com/translate_a/single"

// TranslateDescription turns source-site listing copy into locale. It is live and optional: Lingui catalogs are Mortar's UI, not this text.
func (s *Service) TranslateDescription(ctx context.Context, text, locale string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	form := url.Values{
		"client": {"gtx"},
		"sl":     {"auto"},
		"tl":     {gtxLanguage(locale)},
		"dt":     {"t"},
		"q":      {text},
	}
	endpoint := cmp.Or(s.translateURL, gtxEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", appversion.UserAgent(s.Version))
	resp, err := cmp.Or(s.translateHTTP, http.DefaultClient).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, source.MaxBody+1))
	if err != nil {
		return "", err
	}
	if len(body) > source.MaxBody {
		return "", errors.New("translation: answer too large")
	}
	if resp.StatusCode != http.StatusOK {
		return "", &source.StatusError{Service: "translation", Code: resp.StatusCode, Status: resp.Status}
	}
	out, err := joinedTranslation(body)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errors.New("translation: empty answer")
	}
	return out, nil
}

func gtxLanguage(locale string) string {
	switch locale {
	case "", "en":
		return "en"
	case "zh-Hans":
		// Google's gtx endpoint has no Hans script subtag.
		return "zh-CN"
	case "es-AR", "es-PA":
		return "es"
	default:
		return locale
	}
}

func joinedTranslation(raw []byte) (string, error) {
	raw = bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimSpace(raw), []byte(")]}'")))
	var root []any
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", fmt.Errorf("translation: %w", err)
	}
	if len(root) == 0 {
		return "", errors.New("translation: unexpected answer")
	}
	rows, ok := root[0].([]any)
	if !ok {
		return "", errors.New("translation: unexpected answer")
	}
	var b strings.Builder
	for _, row := range rows {
		pair, ok := row.([]any)
		if !ok || len(pair) == 0 {
			continue
		}
		s, ok := pair[0].(string)
		if !ok {
			continue
		}
		b.WriteString(s)
	}
	return b.String(), nil
}
