//go:build linux

package vm

import "golang.org/x/sys/unix"

// mmapResize remaps the mapping to a new size using mremap(2).
func mmapResize(data []byte, newsize int) ([]byte, error) {
	return unix.Mremap(data, newsize, unix.MREMAP_MAYMOVE)
}
