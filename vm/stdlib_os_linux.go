package vm

import (
	"os"
	"syscall"
)

func statAtime(sys *syscall.Stat_t) (int64, int64) {
	return sys.Atim.Sec, sys.Atim.Nsec
}

func statCtime(sys *syscall.Stat_t) (int64, int64) {
	return sys.Ctim.Sec, sys.Ctim.Nsec
}

func osAtime(info os.FileInfo) float64 {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return float64(sys.Atim.Sec) + float64(sys.Atim.Nsec)/1e9
	}
	return float64(info.ModTime().UnixNano()) / 1e9
}

func osCtime(info os.FileInfo) float64 {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return float64(sys.Ctim.Sec) + float64(sys.Ctim.Nsec)/1e9
	}
	return float64(info.ModTime().UnixNano()) / 1e9
}
