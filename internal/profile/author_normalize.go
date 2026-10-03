package profile

import (
	"regexp"
	"strings"
)

var authorPartSplit = regexp.MustCompile(`\s*[,&]\s*`)

// NormalizeAuthorName folds case and whitespace for author matching.
func NormalizeAuthorName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

// SplitManifestAuthors splits a manifest Author field on commas and ampersands.
func SplitManifestAuthors(field string) []string {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	parts := authorPartSplit.Split(field, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Join(strings.Fields(strings.TrimSpace(part)), " ")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// AuthorFieldIncludes reports whether field lists author, compared with NormalizeAuthorName.
func AuthorFieldIncludes(field, author string) bool {
	want := NormalizeAuthorName(author)
	if want == "" {
		return false
	}
	for _, part := range SplitManifestAuthors(field) {
		if NormalizeAuthorName(part) == want {
			return true
		}
	}
	return false
}
