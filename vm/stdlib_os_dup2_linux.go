//go:build linux

package vm

import "syscall"

func osDup2(oldfd, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
