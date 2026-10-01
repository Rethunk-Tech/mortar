package nxm

// Owner is the app the system currently opens nxm:// links with. ID is what Restore takes back (a desktop file id
// on Linux, the open command on Windows); Name is for the user. An empty ID means no app owns the scheme.
type Owner struct {
	ID   string
	Name string
	Mine bool
}

// Handler registers Mortar for the nxm scheme at runtime; nothing registers it at install time.
type Handler interface {
	Owner() (Owner, error)
	// Register makes Mortar the owner.
	Register() error
	// Restore hands the scheme back to previous, the ID of the owner before Mortar, or leaves it unowned when empty.
	Restore(previous string) error
	// ForwardOther runs the handler recorded in previous on a non-Stardew nxm link.
	ForwardOther(link, previous string) error
	// RegisterLinks makes Mortar the app for mortar:// links and .mortar files where the installer did not, and
	// changes nothing when it already is.
	RegisterLinks() error
}

// Runner runs a command and returns its standard output.
type Runner func(name string, args ...string) (string, error)

func defaultIcon(exe string) string { return exe + ",0" }
