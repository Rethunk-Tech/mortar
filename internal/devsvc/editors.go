package devsvc

import "github.com/Rethunk-Tech/go-editorlocator"

// Editor is one installed program a project can open in; each installed version is its own.
type Editor struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// systemEditor as the codeEditor setting opens a project with the system's default app for its solution. An empty
// setting is the first editor found, and anything else is the program at that path.
const systemEditor = "system"

func asEditors(found []editorlocator.Editor) []Editor {
	out := make([]Editor, len(found))
	for i, e := range found {
		out[i] = Editor{Name: e.Name, Path: e.Program}
	}
	return out
}
