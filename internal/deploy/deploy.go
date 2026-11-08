// Package deploy places a loader's install-side files (BepInEx's Doorstop proxy) into a game's folder for a launch and
// removes them again. The profile holds everything else, so nothing else of the game folder is read, moved or taken
// back. Games whose loader redirects the game to the profile's folder (SMAPI's --mods-path) need none of this.
//
// A launch is Plan, Apply, play, Purge. Apply persists the manifest in the journal folder before it touches the game
// folder, so a crash at any point leaves enough to finish the job: Recover purges from it.
package deploy

// View is the profile's side of a deploy.
type View struct {
	// JournalDir holds the manifest and the displaced-file backups.
	JournalDir string `json:"journalDir"`
	// Roots maps a catalog path role to the absolute folder a plan file naming it is placed under.
	Roots map[string]string `json:"roots,omitempty"`
}

// Op is one file to place. WriteBack marks a file placed in a path role's folder from the profile's own copy, which
// Purge writes back to Src when the game or a mod rewrote it.
type Op struct {
	Src, Dst  string
	WriteBack bool
}

// Owned is a folder a profile entry owns in a path role's folder: Dst is the shared folder and Src the profile's folder
// of the same name, which adopts what a mod creates in Dst during play.
type Owned struct {
	Dst string `json:"dst"`
	Src string `json:"src"`
}

// Plan is what Apply will do: copy each Op's Src to Dst, a path in Dir.
type Plan struct {
	Ops   []Op
	Owned []Owned
	Dir   string
	View  View
}

// Placed is one placed file as the manifest records it.
type Placed struct {
	Dst  string `json:"dst"`
	Src  string `json:"src"`
	Hash string `json:"hash"`
	// Displaced is where the file that was at Dst before is kept, "" when there was none.
	Displaced string `json:"displaced,omitempty"`
	// WriteBack is set for a file placed from a profile's copy: a changed Dst is copied back over Src at purge.
	WriteBack bool `json:"writeBack,omitempty"`
	// Done is set once the file is in place.
	Done bool `json:"done,omitempty"`
	// Undone is set once Purge has taken the file back, so a Purge run again after a failure leaves it alone.
	Undone bool `json:"undone,omitempty"`
	// SrcSize and SrcMTime are the source's size and modification time when its Hash was taken, and Size and MTime the
	// placed file's; a later Play that kept its files judges "unchanged" from these before reading anything.
	SrcSize  int64 `json:"srcSize,omitempty"`
	SrcMTime int64 `json:"srcMTime,omitempty"`
	Size     int64 `json:"size,omitempty"`
	MTime    int64 `json:"mTime,omitempty"`
}

// Manifest is the persisted record of a deploy.
type Manifest struct {
	Dir  string   `json:"dir"`
	View View     `json:"view"`
	Ops  []Placed `json:"ops"`
	// Owned are the folders whose new files Purge adopts into the profile; Existing lists the files the player already had
	// in them when Apply began, which are the player's own whatever the game does to them.
	Owned    []Owned  `json:"owned,omitempty"`
	Existing []string `json:"existing,omitempty"`
	// notes are what this purge left behind for the player to hear about; they are saved beside the journal.
	notes []Note
	// Created are the folders Apply made, removed again by Purge when empty.
	Created []string `json:"created,omitempty"`
	// Keep is set when this Play left the files it placed in place for the next one: Purge then takes back only the
	// files something changed, and the record stays for the next Apply to build on.
	Keep bool `json:"keep,omitempty"`
	// Gen counts the Plays that built on this journal, so a later Play's displaced files never land on an earlier one's
	// backup at the same index.
	Gen int `json:"gen,omitempty"`
}
