package tools

// Tool is an external program the user can start from Mortar for one game.
type Tool struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
	WorkingDir string   `json:"workingDir"`
}

type file struct {
	Tools []Tool `json:"tools"`
}
