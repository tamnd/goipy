//go:build unix && !linux && !darwin

package vm

import "github.com/tamnd/goipy/object"

// extendSelectorsModule sets DefaultSelector = PollSelector on non-Linux,
// non-macOS Unix platforms.
func (i *Interp) extendSelectorsModule(m *object.Module, _ selectorCtx) {
	if v, ok := m.Dict.GetStr("PollSelector"); ok {
		m.Dict.SetStr("DefaultSelector", v)
	}
}
