// Package crashdump reads what a native crash leaves behind (a Windows minidump, a Unity error.log, a systemd
// coredump report, the Windows event log, CatLib's crash report) just far enough to name the module the process died in.
package crashdump

import (
	"fmt"

	"github.com/Rethunk-Tech/go-winsym"
)

// Fault is the module a crash happened in. Module is empty when the address lies in none of the process's modules.
type Fault struct {
	Module string
	// Detail is the exception in a few words, such as "exception 0xc0000005 at 0x7ff8a1b2c3d4".
	Detail string
	// Frame is, for an unhandled .NET exception, the first stack frame outside the runtime and the loader.
	Frame string
	// Managed is that exception's text and stack, for blaming a mod by namespace.
	Managed string
	// CalledFrom is, for a CatLib report, the first frame outside Windows and the C runtime.
	CalledFrom string
	// ReportDir is the folder of the crash report a mod library kept, for the player to open.
	ReportDir string
	// Offset is the crash address's offset into Module when HasOffset, and CodeView that module's debug record when
	// the dump carries it: what a symbol server needs to name the function. ModulePath is where the event log says
	// Module was loaded from, whose own record serves when the dump gave none.
	Offset     uint64
	HasOffset  bool
	CodeView   winsym.CodeView
	ModulePath string
}

// fromMinidump is a minidump's fault in the shape every crash source shares.
func fromMinidump(m winsym.Fault) Fault {
	f := Fault{Detail: fmt.Sprintf("exception 0x%x at 0x%x", m.Code, m.Address), CodeView: m.CodeView}
	if m.Module != "" {
		f.Module, f.Offset, f.HasOffset = baseName(m.Module), m.Offset, true
	}
	return f
}

// baseName is the last element of a Windows or Unix path.
func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '\\' || p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
