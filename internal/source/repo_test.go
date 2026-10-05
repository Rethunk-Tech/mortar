package source

import "testing"

func TestGitHubRepo(t *testing.T) {
	for in, want := range map[string]string{
		"https://github.com/Alice/Cool":        "Alice/Cool",
		"https://github.com/Alice/Cool.git":    "Alice/Cool",
		"https://github.com/Alice/Cool/issues": "Alice/Cool",
		"https://example.com/Alice/Cool":       "",
		"https://github.com/Alice":             "",
		"":                                     "",
	} {
		if got := GitHubRepo(in); got != want {
			t.Errorf("GitHubRepo(%q) = %q, want %q", in, got, want)
		}
	}
}
