// Package winhost is the window system as the services see it: events to the page, native dialogs, the clipboard and the
// browser. It exists so the services do not import the Wails application package, whose cgo link against GTK and
// WebKit would otherwise land in every test binary that touches a service.
package winhost

// Filter is one file-type entry of a dialog; Pattern is a semicolon-separated glob list such as "*.png;*.jpg".
type Filter struct{ Name, Pattern string }

// Dialog describes a native file dialog. Folder asks for a directory instead of a file.
type Dialog struct {
	Title    string
	Filename string
	Filters  []Filter
	Folder   bool
}

// Host is implemented by the running application. A dialog returns "" (or nothing) when the user cancels it.
type Host interface {
	Emit(name string, data any)
	OpenFile(Dialog) (string, error)
	OpenFiles(Dialog) ([]string, error)
	SaveFile(Dialog) (string, error)
	ClipboardText() (string, bool)
	OpenURL(url string) error
}
