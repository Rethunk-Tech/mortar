//go:build !windows

package devsvc

// platformEditors has nothing to add: desktop Linux types a project's files as plain text, so its file associations
// name text editors rather than the programs that open a project.
func platformEditors() []Editor {
	return nil
}
