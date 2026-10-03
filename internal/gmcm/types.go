package gmcm

const Schema = 1

type ModInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Option struct {
	Index           int      `json:"index"`
	Kind            string   `json:"kind"`
	FieldID         *string  `json:"fieldId"`
	Name            string   `json:"name"`
	Tooltip         string   `json:"tooltip"`
	Value           any      `json:"value"`
	Min             *float64 `json:"min"`
	Max             *float64 `json:"max"`
	Interval        *float64 `json:"interval"`
	Choices         []Choice `json:"choices"`
	FormatSamples   []string `json:"formatSamples"`
	Editable        bool     `json:"editable"`
	TitleScreenOnly bool     `json:"titleScreenOnly"`
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
	Applied int       `json:"applied"`
	Skipped []Skipped `json:"skipped"`
}

type IndexMod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Index struct {
	Schema      int        `json:"schema"`
	GmcmVersion string     `json:"gmcmVersion"`
	Mods        []IndexMod `json:"mods"`
}
