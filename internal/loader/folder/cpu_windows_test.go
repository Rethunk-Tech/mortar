package folder_test

import (
	"time"

	"golang.org/x/sys/windows"
)

func cpu() time.Duration {
	var created, exited, kernel, user windows.Filetime
	_ = windows.GetProcessTimes(windows.CurrentProcess(), &created, &exited, &kernel, &user)
	ticks := func(f windows.Filetime) int64 { return int64(f.HighDateTime)<<32 | int64(f.LowDateTime) }
	return time.Duration((ticks(kernel) + ticks(user)) * 100)
}
