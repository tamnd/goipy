//go:build windows

package vm

import (
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

func shutilDiskUsage(path string) (total, used, free int64, err error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	r, _, e := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r == 0 {
		err = e
		return
	}
	total = int64(totalBytes)
	free = int64(totalFreeBytes)
	used = total - free
	return
}
