// Package fomod parses ModuleConfig.xml and resolves which files a FOMOD installer copies.
package fomod

const (
	SelectExactlyOne = "SelectExactlyOne"
	SelectAtMostOne  = "SelectAtMostOne"
	SelectAtLeastOne = "SelectAtLeastOne"
	SelectAll        = "SelectAll"
	SelectAny        = "SelectAny"
)

const (
	TypeRequired      = "Required"
	TypeOptional      = "Optional"
	TypeRecommended   = "Recommended"
	TypeNotUsable     = "NotUsable"
	TypeCouldBeUsable = "CouldBeUsable"
)

const (
	FileActive   = "Active"
	FileInactive = "Inactive"
	FileMissing  = "Missing"
)

// Choices is install step name → group name → selected plugin names.
type Choices map[string]map[string][]string

// Op is one file or folder to copy from the store item into the install folder.
type Op struct {
	Source      string
	Destination string
	Priority    int
	Folder      bool
}

// FileIndex reports a dependency file's state in the profile's mods folder.
type FileIndex func(name string) string

// Config is a parsed ModuleConfig.xml.
type Config struct {
	ModuleName   string
	Required     []Op
	Steps        []Step
	Conditionals []Conditional
}

type Step struct {
	Name    string
	Visible Composite
	Groups  []Group
}

type Group struct {
	Name    string
	Type    string
	Plugins []Plugin
}

type Plugin struct {
	Name        string
	Description string
	Image       string
	Files       []Op
	Flags       []FlagSet
	TypeName    string
	DefaultType string
	Patterns    []TypePattern
}

type FlagSet struct {
	Name  string
	Value string
}

type TypePattern struct {
	Type Composite
	Name string
}

type Conditional struct {
	When  Composite
	Files []Op
}

type Composite struct {
	Operator string
	Files    []FileDep
	Flags    []FlagDep
	Games    []GameDep
	Nested   []Composite
}

type FileDep struct {
	File  string
	State string
}

type FlagDep struct {
	Flag  string
	Value string
}

type GameDep struct {
	Version string
}
