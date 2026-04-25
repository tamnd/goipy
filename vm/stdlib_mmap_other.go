//go:build unix && !linux

package vm

import "errors"

// mmapResize is not available on non-Linux Unix platforms.
func mmapResize(_ []byte, _ int) ([]byte, error) {
	return nil, errors.New("no mremap")
}
