//go:build unix && !linux && !darwin

package vm

import "github.com/tamnd/goipy/object"

// extendSignalModule is a no-op on Unix platforms other than Linux and macOS.
func (i *Interp) extendSignalModule(m *object.Module, _ sigCtx) {}

var _ *object.Module // keep import used
