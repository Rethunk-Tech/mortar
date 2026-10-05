package modreport

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Input is everything needed to build a mod problem report and author URL.
type Input struct {
	ModName       string
	ModVersion    string
	GameVersion   string
	SMAPIVersion  string
	MortarVersion string
	ErrorLines    []string
	LogShareURL   string
	Source        profile.Source
	NexusDomain   string
	GitHubRepo    string
	NexusModID    int
	IssueTitle    string
}

// Result is plain text for the author and a URL to open (GitHub prefilled or Nexus bugs tab).
type Result struct {
	Text   string `json:"text"`
	URL    string `json:"url"`
	Nexus  bool   `json:"nexus"`
	GitHub bool   `json:"github"`
}

// Build formats the report and chooses GitHub issue prefill or a Nexus bugs page.
func Build(in Input) Result {
	lines := in.ErrorLines
	if lines == nil {
		lines = []string{}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Mod: %s %s\n", in.ModName, strings.TrimSpace(in.ModVersion))
	if in.GameVersion != "" {
		fmt.Fprintf(&b, "Game: Stardew Valley %s\n", in.GameVersion)
	}
	if in.SMAPIVersion != "" {
		fmt.Fprintf(&b, "SMAPI: %s\n", in.SMAPIVersion)
	}
	if in.MortarVersion != "" {
		fmt.Fprintf(&b, "Mortar: %s\n", in.MortarVersion)
	}
	b.WriteString("\nErrors:\n")
	if len(lines) == 0 {
		b.WriteString("(none captured)\n")
	} else {
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if in.LogShareURL != "" {
		b.WriteString("\nLog: ")
		b.WriteString(in.LogShareURL)
		b.WriteByte('\n')
	}
	text := b.String()
	title := in.IssueTitle
	if title == "" {
		title = "Error in " + in.ModName
	}
	repo := in.GitHubRepo
	if repo == "" && in.Source.Kind == profile.KindGitHub {
		repo = in.Source.Repo
	}
	modID := in.NexusModID
	if modID == 0 && in.Source.Kind == profile.KindNexus {
		modID = in.Source.ModID
	}
	domain := in.NexusDomain
	if domain == "" {
		if info, ok := components.BundledGame("stardew"); ok {
			domain = info.NexusDomain()
		}
	}
	if repo != "" {
		return Result{Text: text, URL: GitHubIssueURL(repo, title, text), GitHub: true}
	}
	if modID > 0 && domain != "" {
		return Result{Text: text, URL: NexusBugsURL(domain, modID), Nexus: true}
	}
	return Result{Text: text}
}

// BuildFromLog fills version fields from the log summary when empty and collects error lines for modName.
func BuildFromLog(log, modName string, in Input) Result {
	summary := launch.Summarize(log)
	if in.SMAPIVersion == "" {
		in.SMAPIVersion = summary.SMAPI
	}
	if in.GameVersion == "" {
		in.GameVersion = summary.Game
	}
	if len(in.ErrorLines) == 0 {
		in.ErrorLines = ErrorLines(log, modName)
	}
	return Build(in)
}
