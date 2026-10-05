package nxm

import (
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
)

// Schemes are the URL schemes the registered sources claim, which Mortar registers for.
func Schemes() []string { return source.Schemes() }

// Owner is the app the system currently opens a source scheme's links with. ID is what Restore takes back (a desktop file id
// on Linux, the open command on Windows); Name is for the user. An empty ID means no app owns the scheme.
type Owner struct {
	ID   string
	Name string
	Mine bool
}

// Handler registers Mortar for the nxm scheme at runtime; nothing registers it at install time.
type Handler interface {
	// Owner is the current owner of one scheme.
	Owner(scheme string) (Owner, error)
	// Register makes Mortar the owner.
	Register() error
	// Restore hands each scheme back to its previous owner's ID in previous, or leaves it unowned when it has none.
	Restore(previous map[string]string) error
	// Release hands back only the given schemes to their previous owners in previous, and leaves the rest registered.
	Release(schemes []string, previous map[string]string) error
	// ForwardOther runs the handler recorded in previous on a link for a game Mortar does not take.
	ForwardOther(link, previous string) error
	// RegisterLinks makes Mortar the app for mortar:// links and .mortar files where the installer did not, and
	// changes nothing when it already is.
	RegisterLinks() error
}

// Runner runs a command and returns its standard output.
type Runner func(name string, args ...string) (string, error)
