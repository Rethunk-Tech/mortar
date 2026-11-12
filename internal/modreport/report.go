package modreport

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Input is everything needed to build a mod problem report and author URL.
type Input struct {
	ModName    string
	ModVersion string
	// Game is the profile's game id; it names the game in the report and picks the Nexus domain.
	Game          string
	GameVersion   string
	SMAPIVersion  string
	MortarVersion string
	// LoaderName labels SMAPIVersion in the text ("SMAPI", "BepInEx"); empty reads "Loader".
	LoaderName string
	ErrorLines []string
	// Entries and LogName supply ErrorLines from a parsed run when none are given.
	Entries     []launch.Entry
	LogName     string
	LogShareURL string
	Source      profile.Source
	NexusDomain string
	GitHubRepo  string
	NexusModID  int
	IssueTitle  string
	// PageURL is where the mod's own site takes reports when it has no GitHub repository or Nexus bug tracker.
	PageURL string
}

// Result is plain text for the author and a URL to open (GitHub prefilled or Nexus bugs tab).
type Result struct {
	Text   string `json:"text"`
	URL    string `json:"url"`
	Nexus  bool   `json:"nexus"`
	GitHub bool   `json:"github"`
	Page   bool   `json:"page"`
	// Cut says the issue link holds only the start of Text; the whole text goes on the clipboard.
	Cut bool `json:"cut"`
}

// Build formats the report and chooses GitHub issue prefill or a Nexus bugs page.
func Build(in Input) Result {
	lines := in.ErrorLines
	if len(lines) == 0 {
		lines = EntryErrorLines(in.Entries, in.LogName)
	}
	gameInfo, _ := components.Game(in.Game)
	gameName := gameInfo.Name
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", strings.TrimSpace("Mod: "+in.ModName+" "+strings.TrimSpace(in.ModVersion)))
	if in.GameVersion != "" {
		fmt.Fprintf(&b, "Game: %s\n", strings.TrimSpace(gameName+" "+in.GameVersion))
	}
	if in.SMAPIVersion != "" {
		fmt.Fprintf(&b, "%s: %s\n", cmp.Or(in.LoaderName, "Loader"), in.SMAPIVersion)
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
	title = cmp.Or(title, "Error in "+in.ModName)
	repo := in.GitHubRepo
	if repo == "" && in.Source.Kind == profile.KindGitHub {
		repo = in.Source.Repo
	}
	modID := in.NexusModID
	if modID <= 0 && in.Source.Kind == profile.KindNexus {
		modID = in.Source.ModID
	}
	domain := in.NexusDomain
	domain = cmp.Or(domain, gameInfo.NexusDomain())
	if strings.TrimSpace(repo) != "" {
		link, cut := GitHubIssueURL(repo, title, text)
		return Result{Text: text, URL: link, GitHub: true, Cut: cut}
	}
	if modID > 0 && domain != "" {
		return Result{Text: text, URL: NexusBugsURL(domain, modID), Nexus: true}
	}
	return Result{Text: text, URL: in.PageURL, Page: in.PageURL != ""}
}

// BuildFromLog fills version fields from the log summary when empty and collects error lines for modName.
func BuildFromLog(log, modName string, in Input) Result {
	summary := launch.Summarize(log)
	in.LoaderName = cmp.Or(in.LoaderName, "SMAPI")
	in.SMAPIVersion = cmp.Or(in.SMAPIVersion, summary.SMAPI)
	in.GameVersion = cmp.Or(in.GameVersion, summary.Game)
	if len(in.ErrorLines) == 0 {
		in.ErrorLines = ErrorLines(log, modName)
	}
	return Build(in)
}
