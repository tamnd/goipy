//go:build darwin

package vm

import "syscall"

func osDup2(oldfd, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
