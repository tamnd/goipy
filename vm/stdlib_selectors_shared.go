package vm

import (
	"sync"

	"github.com/tamnd/goipy/object"
)

const selectorEventRead = int64(1)
const selectorEventWrite = int64(2)

// selectorState is the shared fd→SelectorKey registry for all selector types.
type selectorState struct {
	mu     sync.RWMutex
	keys   map[int32]*object.Instance // fd → SelectorKey
	closed bool
}

func newSelectorState() *selectorState {
	return &selectorState{keys: make(map[int32]*object.Instance)}
}

// selectorHooks are called by the base selector on fd lifecycle events.
type selectorHooks struct {
	onRegister   func(fd int, events int64) error
	onUnregister func(fd int)
	onModify     func(fd int, newEvents int64) error
	selectFn     func(st *selectorState, timeoutMs int) ([]object.Object, error)
	closeFn      func()
}

// selectorCtx bundles the helpers that platform files need to build selectors.
type selectorCtx struct {
	errCls          *object.Class
	makeSelectorKey func(fileobj object.Object, fd int, events int64, data object.Object) *object.Instance
	buildBase       func(className string, st *selectorState, hooks selectorHooks) *object.Instance
}
