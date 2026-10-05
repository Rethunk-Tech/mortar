// Package configsvc reads and edits mod config files as typed schemas: BepInEx .cfg files and SMAPI config.json
// files in a profile, with GMCM's option metadata where the bridge captured it.
package configsvc

// Config file formats.
const (
	FormatBepInEx = "bepinex"
	FormatSMAPI   = "smapi"
)

// Entry types.
const (
	TypeBool   = "bool"
	TypeInt    = "int"
	TypeFloat  = "float"
	TypeEnum   = "enum"
	TypeString = "string"
	TypeColor  = "color"
	TypeList   = "list"
)

// ConfigFile is one editable config file of a mod.
type ConfigFile struct {
	// Name is the file name within its folder, the id Schema, Set and Reset take.
	Name   string `json:"name"`
	Format string `json:"format"`
	// Label is the plugin's name when the file says it, else the file name.
	Label string `json:"label"`
}

// Entry is one setting. Values are text in the file's own syntax: true/false, a number, an enum member, a string, or
// compact JSON for a list.
type Entry struct {
	Key string `json:"key"`
	// Label is a friendlier name when GMCM gave one, else empty.
	Label       string   `json:"label,omitempty"`
	Type        string   `json:"type"`
	Value       string   `json:"value"`
	Default     string   `json:"default,omitempty"`
	HasDefault  bool     `json:"hasDefault"`
	Description string   `json:"description,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	// Values are the acceptable values of an enum.
	Values []string `json:"values,omitempty"`
	// Flags marks an enum whose values may be combined with commas.
	Flags bool `json:"flags,omitempty"`
}

// Section groups entries; Name is "" for a SMAPI file's top-level settings. A SMAPI object nests as a section named
// by its dotted path.
type Section struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

// Schema is a file's sections in file order.
type Schema struct {
	File     ConfigFile `json:"file"`
	Sections []Section  `json:"sections"`
}
