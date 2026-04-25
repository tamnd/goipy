//go:build linux && arm64

package vm

import (
	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

func (i *Interp) extendSignalModule(m *object.Module, ctx sigCtx) {
	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}
	setInt("SIGIO", int(unix.SIGIO))
	setInt("SIGPWR", int(unix.SIGPWR))
}
