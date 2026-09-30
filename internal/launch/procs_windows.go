//go:build windows

package launch

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Processes lists processes by executable name through Toolhelp. Their command lines are not read
// (that needs the process's PEB), so Args stays nil and callers treat the profile as unknown.
func Processes(_, name string) ([]Process, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var out []Process
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		exe := strings.TrimSuffix(windows.UTF16ToString(entry.ExeFile[:]), ".exe")
		if strings.EqualFold(exe, name) {
			out = append(out, Process{PID: int(entry.ProcessID)})
		}
	}
	return out, nil
}
