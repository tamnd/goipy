package vm

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tamnd/goipy/object"
)

// shmEntry is one named shared-memory block.
type shmEntry struct {
	data []byte
	refs atomic.Int32
}

// slEntry holds the shared item slice for a ShareableList.
type slEntry struct {
	mu    sync.RWMutex
	items []object.Object
}

// shmRegistry maps name string → *shmEntry.
// slRegistry maps name string → *slEntry (ShareableList items).
var (
	shmRegistry sync.Map
	slRegistry  sync.Map
	shmNameSeq  atomic.Int64
)

func shmAutoName() string {
	n := shmNameSeq.Add(1)
	return fmt.Sprintf("psm_%016x", n)
}

// buildSharedMemory constructs the multiprocessing.shared_memory module.
func (i *Interp) buildSharedMemory() *object.Module {
	m := &object.Module{Name: "multiprocessing.shared_memory", Dict: object.NewDict()}

	// --- SharedMemory ---
	m.Dict.SetStr("SharedMemory", &object.BuiltinFunc{Name: "SharedMemory", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		name := ""
		create := false
		size := 0

		if len(args) > 0 {
			if s, ok := args[0].(*object.Str); ok {
				name = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				} else if _, ok2 := v.(*object.NoneType); ok2 {
					name = ""
				}
			}
			if v, ok := kw.GetStr("create"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					create = b.V
				}
			}
			if v, ok := kw.GetStr("size"); ok {
				if n, ok2 := v.(*object.Int); ok2 && n.IsInt64() {
					size = int(n.Int64())
				}
			}
		}
		return i.makeSharedMemory(name, create, size)
	}})

	// --- ShareableList ---
	m.Dict.SetStr("ShareableList", &object.BuiltinFunc{Name: "ShareableList", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		var seq []object.Object
		name := ""

		if len(args) > 0 {
			switch t := args[0].(type) {
			case *object.List:
				seq = t.V
			case *object.Tuple:
				seq = t.V
			case *object.NoneType:
				// seq stays nil
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("sequence"); ok {
				switch t := v.(type) {
				case *object.List:
					seq = t.V
				case *object.Tuple:
					seq = t.V
				}
			}
			if v, ok := kw.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
		}
		return i.makeShareableList(seq, name)
	}})

	return m
}

// ─── SharedMemory ────────────────────────────────────────────────────────────

func (i *Interp) makeSharedMemory(name string, create bool, size int) (*object.Instance, error) {
	var entry *shmEntry

	if create {
		if size <= 0 {
			return nil, object.Errorf(i.valueErr, "size must be a positive number different from zero")
		}
		if name == "" {
			name = shmAutoName()
		}
		entry = &shmEntry{data: make([]byte, size)}
		entry.refs.Store(1)
		if _, loaded := shmRegistry.LoadOrStore(name, entry); loaded {
			return nil, object.Errorf(i.osErr, "[Errno 17] File exists: %q", name)
		}
	} else {
		if name == "" {
			return nil, object.Errorf(i.valueErr, "name can only be None if create=True")
		}
		v, ok := shmRegistry.Load(name)
		if !ok {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", name)
		}
		entry = v.(*shmEntry)
		entry.refs.Add(1)
	}

	return i.shmInstance(name, entry), nil
}

func (i *Interp) shmInstance(name string, entry *shmEntry) *object.Instance {
	cls := &object.Class{Name: "SharedMemory", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var closed atomic.Bool

	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("size", object.NewInt(int64(len(entry.data))))

	// buf property via __getattr__
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok && s.V == "buf" {
				if closed.Load() {
					return nil, object.Errorf(i.valueErr, "operation on closed memory")
				}
				// Return a bytearray backed by the shared slice so that
				// index-writes (shm.buf[0] = 42) are visible cross-goroutine.
				return &object.Bytearray{V: entry.data}, nil
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closed.CompareAndSwap(false, true) {
			entry.refs.Add(-1)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("unlink", &object.BuiltinFunc{Name: "unlink", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		shmRegistry.Delete(name)
		return object.None, nil
	}})

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closed.CompareAndSwap(false, true) {
			entry.refs.Add(-1)
		}
		return object.False, nil
	}})

	return inst
}

// ─── ShareableList ───────────────────────────────────────────────────────────

func (i *Interp) makeShareableList(seq []object.Object, name string) (*object.Instance, error) {
	nSlots := len(seq)
	shmSize := nSlots*9 + 8
	if shmSize < 8 {
		shmSize = 8
	}

	var shmInst *object.Instance
	var err error
	var sl *slEntry

	if name != "" {
		// Attach to existing shared list by name.
		shmInst, err = i.makeSharedMemory(name, false, 0)
		if err != nil {
			return nil, err
		}
		// Look up the shared item store keyed by this name.
		v, ok := slRegistry.Load(name)
		if !ok {
			// Fallback: create a new empty entry (for attach-only scenario).
			sl = &slEntry{items: []object.Object{}}
			slRegistry.Store(name, sl)
		} else {
			sl = v.(*slEntry)
		}
	} else {
		shmInst, err = i.makeSharedMemory("", true, shmSize)
		if err != nil {
			return nil, err
		}
		// Create a new slEntry and register it under the auto-generated name.
		items := make([]object.Object, nSlots)
		for k, v := range seq {
			items[k] = v
		}
		sl = &slEntry{items: items}
	}

	// Get the name that was assigned to the SharedMemory block.
	var shmName string
	if sv, ok2 := shmInst.Dict.GetStr("name"); ok2 {
		if s, ok3 := sv.(*object.Str); ok3 {
			shmName = s.V
		}
	}
	// Register the slEntry under the SharedMemory name so goroutines can find it.
	slRegistry.Store(shmName, sl)

	cls := &object.Class{Name: "ShareableList", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	inst.Dict.SetStr("shm", shmInst)
	inst.Dict.SetStr("_name", &object.Str{V: shmName})
	inst.Dict.SetStr("format", &object.Str{V: ""})

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		sl.mu.RLock()
		n := len(sl.items)
		sl.mu.RUnlock()
		return object.NewInt(int64(n)), nil
	}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) < 1 {
			return nil, object.Errorf(i.indexErr, "index required")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indices must be integers")
		}
		sl.mu.RLock()
		defer sl.mu.RUnlock()
		idx := int(n.Int64())
		if idx < 0 {
			idx += len(sl.items)
		}
		if idx < 0 || idx >= len(sl.items) {
			return nil, object.Errorf(i.indexErr, "index out of range")
		}
		return sl.items[idx], nil
	}})

	cls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) < 2 {
			return nil, object.Errorf(i.indexErr, "index and value required")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indices must be integers")
		}
		sl.mu.Lock()
		defer sl.mu.Unlock()
		idx := int(n.Int64())
		if idx < 0 {
			idx += len(sl.items)
		}
		if idx < 0 || idx >= len(sl.items) {
			return nil, object.Errorf(i.indexErr, "index out of range")
		}
		sl.items[idx] = a[1]
		return object.None, nil
	}})

	cls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		sl.mu.RLock()
		snap := make([]object.Object, len(sl.items))
		copy(snap, sl.items)
		sl.mu.RUnlock()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(snap) {
				return nil, false, nil
			}
			v := snap[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		sl.mu.RLock()
		snap := make([]object.Object, len(sl.items))
		copy(snap, sl.items)
		sl.mu.RUnlock()
		return &object.Str{V: "ShareableList(" + object.Repr(&object.List{V: snap}) + ")"}, nil
	}})

	cls.Dict.SetStr("count", &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		target := a[0]
		sl.mu.RLock()
		defer sl.mu.RUnlock()
		n := 0
		for _, v := range sl.items {
			eq, err := object.Eq(v, target)
			if err != nil {
				return nil, err
			}
			if eq {
				n++
			}
		}
		return object.NewInt(int64(n)), nil
	}})

	cls.Dict.SetStr("index", &object.BuiltinFunc{Name: "index", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "index() requires a value argument")
		}
		target := a[0]
		sl.mu.RLock()
		defer sl.mu.RUnlock()
		for k, v := range sl.items {
			eq, err := object.Eq(v, target)
			if err != nil {
				return nil, err
			}
			if eq {
				return object.NewInt(int64(k)), nil
			}
		}
		return nil, object.Errorf(i.valueErr, "%v is not in list", object.Repr(target))
	}})

	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok && s.V == "shm" {
				return shmInst, nil
			}
		}
		return object.None, nil
	}})

	return inst, nil
}
