package gmcm

const Schema = 1

type ModInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// The capture, pending and result files are the bridge's (mortar-smapi-bridge Gmcm/GmcmModels.cs); testdata holds
// copies of the fixtures its tests write, so the two sides cannot drift apart unnoticed.

// Choice is one value a choice option offers. Value is a string for a choice, a number for an image picker.
type Choice struct {
	Value any    `json:"value"`
	Label string `json:"label"`
}

// FormatSample is how the mod displays one value of a numeric option.
type FormatSample struct {
	Value any    `json:"value"`
	Label string `json:"label"`
}

type Option struct {
	Index           int            `json:"index"`
	Kind            string         `json:"kind"`
	FieldID         *string        `json:"fieldId"`
	Name            string         `json:"name"`
	Tooltip         string         `json:"tooltip"`
	Value           any            `json:"value"`
	Min             *float64       `json:"min"`
	Max             *float64       `json:"max"`
	Interval        *float64       `json:"interval"`
	Choices         []Choice       `json:"choices"`
	FormatSamples   []FormatSample `json:"formatSamples"`
	Editable        bool           `json:"editable"`
	TitleScreenOnly bool           `json:"titleScreenOnly"`
}

type Page struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Options []Option `json:"options"`
}

type Capture struct {
	Schema                 int     `json:"schema"`
	Mod                    ModInfo `json:"mod"`
	GmcmVersion            string  `json:"gmcmVersion"`
	CapturedAt             string  `json:"capturedAt"`
	TitleScreenOnlyDefault bool    `json:"titleScreenOnlyDefault"`
	Pages                  []Page  `json:"pages"`
}

type Edit struct {
	Page    string `json:"page"`
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	FieldID string `json:"fieldId"`
	Name    string `json:"name"`
	Value   any    `json:"value"`
}

type Pending struct {
	Schema int    `json:"schema"`
	Edits  []Edit `json:"edits"`
}

type Skipped struct {
	Edit   Edit   `json:"edit"`
	Reason string `json:"reason"`
}

type Result struct {
	Schema  int       `json:"schema"`
	Applied []Edit    `json:"applied"`
	Skipped []Skipped `json:"skipped"`
}
