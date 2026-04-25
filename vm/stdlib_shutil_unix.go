//go:build unix

package vm

import "syscall"

func shutilDiskUsage(path string) (total, used, free int64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return
	}
	total = int64(stat.Blocks) * int64(stat.Bsize)
	free = int64(stat.Bavail) * int64(stat.Bsize)
	used = total - int64(stat.Bfree)*int64(stat.Bsize)
	return
}
