package vm

import (
	"syscall"

	"golang.org/x/sys/unix"
)

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

func processTimeNs() int64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return (int64(ru.Utime.Sec)+int64(ru.Stime.Sec))*1e9 +
		int64(ru.Utime.Usec+ru.Stime.Usec)*1000
}

func processTimeSecs() float64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return float64(ru.Utime.Sec) + float64(ru.Stime.Sec) +
		float64(ru.Utime.Usec+ru.Stime.Usec)/1e6
}
