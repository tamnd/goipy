package vm

import "syscall"

// Linux CLOCK_* constants.
const (
	clockRealtime       = 0
	clockMonotonic      = 1
	clockProcessCPUTime = 2
	clockThreadCPUTime  = 3
)

func threadTimeNs() int64 {
	var ts syscall.Timespec
	if err := syscall.ClockGettime(clockThreadCPUTime, &ts); err != nil {
		return 0
	}
	return ts.Sec*1e9 + int64(ts.Nsec)
}
