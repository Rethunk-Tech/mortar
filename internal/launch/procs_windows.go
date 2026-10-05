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
			out = append(out, Process{PID: int(entry.ProcessID), Exe: imagePath(entry.ProcessID)})
		}
	}
	return out, nil
}

// imagePath is the executable path of pid, or "" when the process cannot be opened.
func imagePath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var buf [windows.MAX_LONG_PATH]uint16
	n := uint32(windows.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}
