package vm

import "golang.org/x/sys/unix"

// Darwin CLOCK_* constants (from <time.h>).
const (
	clockRealtime       = 0
	clockMonotonic      = 6
	clockProcessCPUTime = 12
	clockThreadCPUTime  = 16
)

func threadTimeNs() int64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(clockThreadCPUTime, &ts); err != nil {
		return 0
	}
	return ts.Sec*1e9 + int64(ts.Nsec)
}
