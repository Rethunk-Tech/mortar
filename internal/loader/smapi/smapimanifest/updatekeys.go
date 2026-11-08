package smapimanifest

import "github.com/Rethunk-Tech/go-smapi"

// FirstNexusID is the Nexus mod of the first update key that names one, or 0.
func FirstNexusID(keys []string) int {
	for _, k := range keys {
		if n, ok := smapi.NexusUpdateKey(k); ok {
			return n
		}
	}
	return 0
}

// FirstGitHubRepo is the repository of the first GitHub update key, or "".
func FirstGitHubRepo(keys []string) string {
	for _, k := range keys {
		if repo, ok := smapi.GitHubUpdateKey(k); ok {
			return repo
		}
	}
	return ""
}
