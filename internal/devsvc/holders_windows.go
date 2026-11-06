package devsvc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ponytail: a copy of go-safefs's Holders (its next release after v0.6.0), which also names a held folder's error
// with ErrInUse. Ceiling: two copies to keep in step. Upgrade: bump go-safefs, call safefs.Holders, delete this file.

var (
	rstrtmgr            = windows.NewLazySystemDLL("rstrtmgr.dll")
	rmStartSession      = rstrtmgr.NewProc("RmStartSession")
	rmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	rmGetList           = rstrtmgr.NewProc("RmGetList")
	rmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

const fileProcessIdsUsingFileInformation = 47

// rmUniqueProcess is RM_UNIQUE_PROCESS.
type rmUniqueProcess struct {
	pid     uint32
	started windows.Filetime
}

// rmProcessInfo is RM_PROCESS_INFO.
type rmProcessInfo struct {
	process     rmUniqueProcess
	appName     [256]uint16
	serviceName [64]uint16
	appType     uint32
	appStatus   uint32
	tsSessionID uint32
	restartable int32
}

// holderNames names the programs other than Mortar that have dir, a folder in it or a file in it open, as Windows
// shows them ("Visual Studio Code"). A folder's holders include a program whose current folder it is.
func holderNames(dir string) ([]string, bool) {
	self := os.Getpid()
	var pids []uint32
	for _, p := range projectPaths(dir) {
		for _, pid := range pidsUsing(p) {
			if int(pid) != self && !slices.Contains(pids, pid) {
				pids = append(pids, pid)
			}
		}
	}
	names := appNames(pids)
	out := make([]string, 0, len(pids))
	for _, pid := range pids {
		name := names[pid]
		if name == "" {
			name = exeName(pid)
		}
		out = append(out, name)
	}
	return uniqueNames(out), true
}

// pidsUsing asks the file system which processes have path open. Opening it for its attributes alone passes any
// sharing the holders asked for.
func pidsUsing(path string) []uint32 {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil
	}
	defer func() { _ = windows.CloseHandle(h) }()
	// FILE_PROCESS_IDS_USING_FILE_INFORMATION: a ULONG count, then ULONG_PTR ids; as uintptrs the count is the low half
	// of the first word.
	buf := make([]uintptr, 256)
	for {
		var iosb windows.IO_STATUS_BLOCK
		err := windows.NtQueryInformationFile(h, &iosb, (*byte)(unsafe.Pointer(&buf[0])), uint32(len(buf))*uint32(unsafe.Sizeof(buf[0])), fileProcessIdsUsingFileInformation) //nolint:gosec // G103: the call fills a caller-owned buffer
		if errors.Is(err, windows.STATUS_INFO_LENGTH_MISMATCH) && len(buf) < 1<<16 {
			buf = make([]uintptr, len(buf)*4)
			continue
		}
		if err != nil {
			return nil
		}
		n := min(int(uint32(buf[0])), len(buf)-1) //nolint:gosec // G115: the count is a ULONG
		out := make([]uint32, 0, n)
		for _, id := range buf[1 : 1+n] {
			out = append(out, uint32(id)) //nolint:gosec // G115: Windows process ids fit in 32 bits
		}
		return out
	}
}

// appNames asks the Restart Manager for each process's name as it shows it. A process that has exited or cannot be
// opened is left out.
func appNames(pids []uint32) map[uint32]string {
	out := map[uint32]string{}
	var procs []rmUniqueProcess
	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			continue
		}
		var created, exited, kernel, user windows.Filetime
		err = windows.GetProcessTimes(h, &created, &exited, &kernel, &user)
		_ = windows.CloseHandle(h)
		if err == nil {
			procs = append(procs, rmUniqueProcess{pid: pid, started: created})
		}
	}
	if len(procs) == 0 {
		return out
	}
	var session uint32
	key := make([]uint16, 33)                                                                                           // CCH_RM_SESSION_KEY + 1
	if r, _, _ := rmStartSession.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0]))); r != 0 { //nolint:gosec // G103: the Restart Manager takes pointers to its out-parameters
		return out
	}
	defer func() { _, _, _ = rmEndSession.Call(uintptr(session)) }()
	if r, _, _ := rmRegisterResources.Call(uintptr(session), 0, 0, uintptr(len(procs)), uintptr(unsafe.Pointer(&procs[0])), 0, 0); r != 0 { //nolint:gosec // G103: RmRegisterResources reads a caller-owned array
		return out
	}
	infos := make([]rmProcessInfo, len(procs)+4)
	for {
		var needed, reasons uint32
		count := uint32(len(infos))                                                                                                                                                           //nolint:gosec // G115: a handful of processes
		r, _, _ := rmGetList.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&infos[0])), uintptr(unsafe.Pointer(&reasons))) //nolint:gosec // G103: RmGetList fills caller-owned buffers
		if windows.Errno(r) == windows.ERROR_MORE_DATA {
			infos = make([]rmProcessInfo, needed+4)
			continue
		}
		if r != 0 {
			return out
		}
		for _, in := range infos[:count] {
			out[in.process.pid] = windows.UTF16ToString(in.appName[:])
		}
		return out
	}
}

// exeName is the file name of the program running as pid, or "" when it cannot be read.
func exeName(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer func() { _ = windows.CloseHandle(h) }()
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(windows.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:size]))
}

// maxHolderPaths bounds how much of a project is asked about when it cannot be moved: its folders first, since a
// terminal or a file watcher holds a folder, then its files.
const maxHolderPaths = 4000

// projectPaths lists dir's folders, then its files, up to maxHolderPaths; a .git folder's contents are left out.
func projectPaths(dir string) []string {
	var dirs, files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		switch {
		case len(dirs)+len(files) >= maxHolderPaths:
			return fs.SkipAll
		case err != nil:
			return fs.SkipDir
		case d.IsDir():
			dirs = append(dirs, path)
			if d.Name() == ".git" {
				return fs.SkipDir
			}
		default:
			files = append(files, path)
		}
		return nil
	})
	return slices.Concat(dirs, files)
}

// uniqueNames keeps each non-empty name once, in order.
func uniqueNames(names []string) []string {
	out := []string{}
	for _, n := range names {
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}
