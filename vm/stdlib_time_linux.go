package vm

import "golang.org/x/sys/unix"

// Linux CLOCK_* constants.
const (
	clockRealtime       = 0
	clockMonotonic      = 1
	clockProcessCPUTime = 2
	clockThreadCPUTime  = 3
)

func threadTimeNs() int64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(clockThreadCPUTime, &ts); err != nil {
		return 0
	}
	return ts.Sec*1e9 + int64(ts.Nsec)
}
