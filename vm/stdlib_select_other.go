//go:build unix && !linux && !darwin

package vm

import "github.com/tamnd/goipy/object"

// extendSelectModule is a no-op on Unix platforms other than Linux and macOS.
func (i *Interp) extendSelectModule(m *object.Module, errCls *object.Class) {}
