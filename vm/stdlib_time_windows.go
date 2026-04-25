//go:build windows

package vm

import "time"

const (
	clockRealtime       = 0
	clockMonotonic      = 1
	clockProcessCPUTime = 2
	clockThreadCPUTime  = 3
)

func threadTimeNs() int64 {
	return time.Now().UnixNano()
}

func processTimeNs() int64 {
	return time.Now().UnixNano()
}

func processTimeSecs() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}
