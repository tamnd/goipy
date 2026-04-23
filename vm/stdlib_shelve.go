package vm

import (
	"encoding/gob"
	"os"
	"sync"

	"github.com/tamnd/goipy/object"
)

// shelfStore is the file-backed KV store for a Shelf.
type shelfStore struct {
	mu       sync.Mutex
	data     map[string][]byte
	filename string
	readOnly bool
	closed   bool
}

func loadShelfStore(filename, flag string) (*shelfStore, error) {
	s := &shelfStore{filename: filename + ".shelf", data: map[string][]byte{}}
	switch flag {
	case "n":
		// Start fresh, ignore existing file.
		return s, nil
	case "r":
		s.readOnly = true
		if err := s.load(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return s, nil
	case "c":
		if err := s.load(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return s, nil
	case "w":
		if err := s.load(); err != nil {
			return nil, err
		}
		return s, nil
	}
	return nil, nil
}

func (s *shelfStore) load() error {
	f, err := os.Open(s.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewDecoder(f).Decode(&s.data)
}

func (s *shelfStore) save() error {
	if s.readOnly {
		return nil
	}
	f, err := os.Create(s.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(s.data)
}

// shelfInstance is the Python-facing Shelf object.
type shelfInstance struct {
	store     *shelfStore
	proto     int
	writeback bool
	cache     map[string]object.Object // writeback cache
}

func (i *Interp) buildShelve() *object.Module {
	m := &object.Module{Name: "shelve", Dict: object.NewDict()}

	// Shelf class skeleton
	shelfClass := &object.Class{Name: "Shelf", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	dbfilenameShelfClass := &object.Class{Name: "DbfilenameShelf", Bases: []*object.Class{shelfClass}, Dict: object.NewDict()}
	m.Dict.SetStr("Shelf", shelfClass)
	m.Dict.SetStr("DbfilenameShelf", dbfilenameShelfClass)
	m.Dict.SetStr("DEFAULT_PROTOCOL", object.IntFromInt64(2))

	// error class for I/O errors (like dbm.error)
	shelveErrClass := &object.Class{Name: "error", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	m.Dict.SetStr("error", shelveErrClass)

	pickleErr := i.exception // use base Exception for internal pickling errors

	makeShelf := func(filename, flag string, proto int, writeback bool) (object.Object, error) {
		store, err := loadShelfStore(filename, flag)
		if err != nil {
			return nil, object.Errorf(i.osErr, "shelve.open: %v", err)
		}
		sh := &shelfInstance{store: store, proto: proto, writeback: writeback}
		if writeback {
			sh.cache = map[string]object.Object{}
		}
		return i.makeShelfObject(sh, shelfClass, shelveErrClass, pickleErr), nil
	}

	// shelve.open(filename, flag='c', protocol=None, writeback=False)
	openFunc := &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "shelve.open() requires filename")
		}
		fnStr, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shelve.open() filename must be str")
		}
		flag := "c"
		proto := 2
		writeback := false

		if len(args) >= 2 {
			if f, ok2 := args[1].(*object.Str); ok2 {
				flag = f.V
			}
		}
		if len(args) >= 3 {
			if p, ok2 := args[2].(*object.Int); ok2 {
				proto = int(p.Int64())
			}
		}
		if len(args) >= 4 {
			if wb, ok2 := args[3].(*object.Bool); ok2 {
				writeback = wb.V
			}
		}
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("flag"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					flag = s.V
				}
			}
			if v, ok2 := kwargs.GetStr("protocol"); ok2 {
				if v != object.None {
					if p, ok3 := v.(*object.Int); ok3 {
						proto = int(p.Int64())
					}
				}
			}
			if v, ok2 := kwargs.GetStr("writeback"); ok2 {
				if wb, ok3 := v.(*object.Bool); ok3 {
					writeback = wb.V
				}
			}
		}
		return makeShelf(fnStr.V, flag, proto, writeback)
	}}
	m.Dict.SetStr("open", openFunc)

	return m
}

// makeShelfObject wraps a shelfInstance in a Python-like Instance object.
func (i *Interp) makeShelfObject(sh *shelfInstance, cls, shelveErr, pickleErr *object.Class) *object.Instance {
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	checkOpen := func() error {
		if sh.store.closed {
			return object.Errorf(i.valueErr, "I/O operation on closed shelf")
		}
		return nil
	}

	getItem := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__getitem__ requires key")
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		if sh.writeback {
			if cached, ok := sh.cache[key]; ok {
				return cached, nil
			}
		}
		sh.store.mu.Lock()
		data, ok := sh.store.data[key]
		sh.store.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%q", key)
		}
		obj, _, err2 := pickleDeserializeN(data)
		if err2 != nil {
			return nil, object.Errorf(pickleErr, "unpickling error: %v", err2)
		}
		if sh.writeback {
			sh.cache[key] = obj
		}
		return obj, nil
	}

	setItem := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if sh.store.readOnly {
			return nil, object.Errorf(shelveErr, "db type does not support writeable access")
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires key and value")
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		data, err := pickleSerialize(args[1], sh.proto)
		if err != nil {
			return nil, object.Errorf(pickleErr, "pickling error: %v", err)
		}
		sh.store.mu.Lock()
		sh.store.data[key] = data
		sh.store.mu.Unlock()
		if sh.writeback {
			sh.cache[key] = args[1]
		}
		return object.None, nil
	}

	delItem := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if sh.store.readOnly {
			return nil, object.Errorf(shelveErr, "db type does not support writeable access")
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__delitem__ requires key")
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		_, ok := sh.store.data[key]
		if ok {
			delete(sh.store.data, key)
		}
		sh.store.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%q", key)
		}
		if sh.writeback {
			delete(sh.cache, key)
		}
		return object.None, nil
	}

	contains := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__contains__ requires key")
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		_, ok := sh.store.data[key]
		sh.store.mu.Unlock()
		return object.BoolOf(ok), nil
	}

	length := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		n := len(sh.store.data)
		sh.store.mu.Unlock()
		return object.IntFromInt64(int64(n)), nil
	}

	keys := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		ks := make([]object.Object, 0, len(sh.store.data))
		for k := range sh.store.data {
			ks = append(ks, &object.Str{V: k})
		}
		sh.store.mu.Unlock()
		return &object.List{V: ks}, nil
	}

	values := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		rawData := make(map[string][]byte, len(sh.store.data))
		for k, v := range sh.store.data {
			rawData[k] = v
		}
		sh.store.mu.Unlock()
		vs := make([]object.Object, 0, len(rawData))
		for _, data := range rawData {
			obj, _, err := pickleDeserializeN(data)
			if err != nil {
				return nil, object.Errorf(pickleErr, "unpickling error: %v", err)
			}
			vs = append(vs, obj)
		}
		return &object.List{V: vs}, nil
	}

	items := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		rawData := make(map[string][]byte, len(sh.store.data))
		for k, v := range sh.store.data {
			rawData[k] = v
		}
		sh.store.mu.Unlock()
		it := make([]object.Object, 0, len(rawData))
		for k, data := range rawData {
			obj, _, err := pickleDeserializeN(data)
			if err != nil {
				return nil, object.Errorf(pickleErr, "unpickling error: %v", err)
			}
			it = append(it, &object.Tuple{V: []object.Object{&object.Str{V: k}, obj}})
		}
		return &object.List{V: it}, nil
	}

	getMethod := func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "get() requires key")
		}
		var def object.Object = object.None
		if len(args) >= 2 {
			def = args[1]
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		data, ok := sh.store.data[key]
		sh.store.mu.Unlock()
		if !ok {
			return def, nil
		}
		obj, _, err2 := pickleDeserializeN(data)
		if err2 != nil {
			return nil, object.Errorf(pickleErr, "unpickling error: %v", err2)
		}
		return obj, nil
	}

	pop := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if sh.store.readOnly {
			return nil, object.Errorf(shelveErr, "db type does not support writeable access")
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "pop() requires key")
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		data, ok := sh.store.data[key]
		if ok {
			delete(sh.store.data, key)
		}
		sh.store.mu.Unlock()
		if !ok {
			if len(args) >= 2 {
				return args[1], nil
			}
			return nil, object.Errorf(i.keyErr, "%q", key)
		}
		if sh.writeback {
			delete(sh.cache, key)
		}
		obj, _, err2 := pickleDeserializeN(data)
		if err2 != nil {
			return nil, object.Errorf(pickleErr, "unpickling error: %v", err2)
		}
		return obj, nil
	}

	setdefault := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "setdefault() requires key")
		}
		var def object.Object = object.None
		if len(args) >= 2 {
			def = args[1]
		}
		key, err := i.toShelfKey(args[0])
		if err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		data, ok := sh.store.data[key]
		sh.store.mu.Unlock()
		if ok {
			obj, _, err2 := pickleDeserializeN(data)
			if err2 != nil {
				return nil, object.Errorf(pickleErr, "unpickling error: %v", err2)
			}
			return obj, nil
		}
		// Not found: set default
		if sh.store.readOnly {
			return nil, object.Errorf(shelveErr, "db type does not support writeable access")
		}
		serialized, err2 := pickleSerialize(def, sh.proto)
		if err2 != nil {
			return nil, object.Errorf(pickleErr, "pickling error: %v", err2)
		}
		sh.store.mu.Lock()
		sh.store.data[key] = serialized
		sh.store.mu.Unlock()
		return def, nil
	}

	update := func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if sh.store.readOnly {
			return nil, object.Errorf(shelveErr, "db type does not support writeable access")
		}
		applyPair := func(k, v object.Object) error {
			key, err := i.toShelfKey(k)
			if err != nil {
				return err
			}
			data, err := pickleSerialize(v, sh.proto)
			if err != nil {
				return object.Errorf(pickleErr, "pickling error: %v", err)
			}
			sh.store.mu.Lock()
			sh.store.data[key] = data
			sh.store.mu.Unlock()
			return nil
		}
		if len(args) >= 1 {
			switch src := args[0].(type) {
			case *object.Dict:
				ks, vs := src.Items()
				for idx, k := range ks {
					if err := applyPair(k, vs[idx]); err != nil {
						return nil, err
					}
				}
			case *object.List:
				for _, item := range src.V {
					tup, ok := item.(*object.Tuple)
					if !ok || len(tup.V) != 2 {
						return nil, object.Errorf(i.typeErr, "update() sequence elements must be pairs")
					}
					if err := applyPair(tup.V[0], tup.V[1]); err != nil {
						return nil, err
					}
				}
			}
		}
		if kwargs != nil {
			ks, vs := kwargs.Items()
			for idx, k := range ks {
				if err := applyPair(k, vs[idx]); err != nil {
					return nil, err
				}
			}
		}
		return object.None, nil
	}

	syncFn := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if sh.writeback && len(sh.cache) > 0 {
			for key, obj := range sh.cache {
				data, err := pickleSerialize(obj, sh.proto)
				if err != nil {
					return nil, object.Errorf(pickleErr, "pickling error: %v", err)
				}
				sh.store.mu.Lock()
				sh.store.data[key] = data
				sh.store.mu.Unlock()
			}
			sh.cache = map[string]object.Object{}
		}
		sh.store.mu.Lock()
		err := sh.store.save()
		sh.store.mu.Unlock()
		if err != nil {
			return nil, object.Errorf(i.osErr, "shelf sync: %v", err)
		}
		return object.None, nil
	}

	closeFn := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if sh.store.closed {
			return object.None, nil
		}
		// sync first
		if _, err := syncFn(nil, nil, nil); err != nil {
			return nil, err
		}
		sh.store.closed = true
		return object.None, nil
	}

	enter := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}

	exit := func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		_, err := closeFn(nil, nil, nil)
		return object.None, err
	}

	iterFn := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		sh.store.mu.Lock()
		ks := make([]string, 0, len(sh.store.data))
		for k := range sh.store.data {
			ks = append(ks, k)
		}
		sh.store.mu.Unlock()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(ks) {
				return nil, false, nil
			}
			k := ks[idx]
			idx++
			return &object.Str{V: k}, true, nil
		}}, nil
	}

	bf := func(name string, fn func(any, []object.Object, *object.Dict) (object.Object, error)) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: fn}
	}

	d.SetStr("__getitem__", bf("__getitem__", getItem))
	d.SetStr("__setitem__", bf("__setitem__", setItem))
	d.SetStr("__delitem__", bf("__delitem__", delItem))
	d.SetStr("__contains__", bf("__contains__", contains))
	d.SetStr("__len__", bf("__len__", length))
	d.SetStr("__iter__", bf("__iter__", iterFn))
	d.SetStr("__enter__", bf("__enter__", enter))
	d.SetStr("__exit__", bf("__exit__", exit))
	d.SetStr("keys", bf("keys", keys))
	d.SetStr("values", bf("values", values))
	d.SetStr("items", bf("items", items))
	d.SetStr("get", bf("get", getMethod))
	d.SetStr("pop", bf("pop", pop))
	d.SetStr("setdefault", bf("setdefault", setdefault))
	d.SetStr("update", bf("update", update))
	d.SetStr("sync", bf("sync", syncFn))
	d.SetStr("close", bf("close", closeFn))

	return inst
}

// toShelfKey extracts a string key from a Python object.
func (i *Interp) toShelfKey(obj object.Object) (string, error) {
	s, ok := obj.(*object.Str)
	if !ok {
		return "", object.Errorf(i.typeErr, "shelf keys must be strings, not %s", object.TypeName(obj))
	}
	return s.V, nil
}
