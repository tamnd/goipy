package vm

import (
	"os"
	"syscall"
)

func statAtime(sys *syscall.Stat_t) (int64, int64) {
	return sys.Atimespec.Sec, sys.Atimespec.Nsec
}

func statCtime(sys *syscall.Stat_t) (int64, int64) {
	return sys.Ctimespec.Sec, sys.Ctimespec.Nsec
}

func statMtime(sys *syscall.Stat_t) (int64, int64) {
	return sys.Mtimespec.Sec, sys.Mtimespec.Nsec
}

func osAtime(info os.FileInfo) float64 {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return float64(sys.Atimespec.Sec) + float64(sys.Atimespec.Nsec)/1e9
	}
	return float64(info.ModTime().UnixNano()) / 1e9
}

func osCtime(info os.FileInfo) float64 {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return float64(sys.Ctimespec.Sec) + float64(sys.Ctimespec.Nsec)/1e9
	}
	return float64(info.ModTime().UnixNano()) / 1e9
}
