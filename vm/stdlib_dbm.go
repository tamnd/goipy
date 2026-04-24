package vm

import (
	"encoding/gob"
	"os"
	"sync"

	"github.com/tamnd/goipy/object"
)

// dbmStore is the same gob-based file-backed KV store used by shelve,
// but keys and values are raw bytes so the interface matches CPython dbm.
type dbmStore struct {
	mu       sync.Mutex
	data     map[string][]byte
	filename string // <basename>.dbm
	readOnly bool
	closed   bool
}

func loadDbmStore(filename, flag string) (*dbmStore, error) {
	s := &dbmStore{filename: filename + ".dbm", data: map[string][]byte{}}
	switch flag {
	case "n":
		return s, nil
	case "r":
		s.readOnly = true
		if err := s.loadFile(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return s, nil
	case "c":
		if err := s.loadFile(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return s, nil
	case "w":
		if err := s.loadFile(); err != nil {
			return nil, err
		}
		return s, nil
	}
	return nil, nil
}

func (s *dbmStore) loadFile() error {
	f, err := os.Open(s.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewDecoder(f).Decode(&s.data)
}

func (s *dbmStore) saveFile() error {
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

// dbmKey converts a Python str or bytes key to a Go string (raw bytes for bytes,
// UTF-8 for str) matching CPython dbm behaviour.
func dbmKey(obj object.Object) (string, error) {
	switch v := obj.(type) {
	case *object.Str:
		return v.V, nil
	case *object.Bytes:
		return string(v.V), nil
	case *object.Bytearray:
		return string(v.V), nil
	}
	return "", object.Errorf(nil, "dbm keys must be str or bytes-like, not %s", object.TypeName(obj))
}

// dbmVal converts a Python str or bytes value to raw bytes.
func dbmVal(obj object.Object) ([]byte, error) {
	switch v := obj.(type) {
	case *object.Str:
		return []byte(v.V), nil
	case *object.Bytes:
		return v.V, nil
	case *object.Bytearray:
		return v.V, nil
	}
	return nil, object.Errorf(nil, "dbm values must be str or bytes-like, not %s", object.TypeName(obj))
}

func (i *Interp) buildDbm() *object.Module {
	m := &object.Module{Name: "dbm", Dict: object.NewDict()}

	// error class: dbm.error is a Python tuple (error_cls, OSError)
	errCls := &object.Class{Name: "error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	errTuple := &object.Tuple{V: []object.Object{errCls, i.osErr}}
	m.Dict.SetStr("error", errTuple)

	makeDb := func(filename, flag string) (object.Object, error) {
		store, err := loadDbmStore(filename, flag)
		if err != nil {
			return nil, object.Errorf(errCls, "dbm.open: %v", err)
		}
		return i.makeDbmObject(store, errCls), nil
	}

	// dbm.open(file, flag='r', mode=0o666)
	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "open() requires filename")
		}
		fnStr, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "open() filename must be str")
		}
		flag := "r"
		if len(args) >= 2 {
			if f, ok2 := args[1].(*object.Str); ok2 {
				flag = f.V
			}
		}
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("flag"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					flag = s.V
				}
			}
		}
		return makeDb(fnStr.V, flag)
	}})

	// dbm.whichdb(filename) → None / '' / 'dbm.sqlite3'
	m.Dict.SetStr("whichdb", &object.BuiltinFunc{Name: "whichdb", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "whichdb() requires filename")
		}
		fnStr, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "whichdb() filename must be str")
		}
		// Our format stores data in <filename>.dbm.
		if _, err := os.Stat(fnStr.V + ".dbm"); err == nil {
			return &object.Str{V: "dbm.sqlite3"}, nil
		}
		// If the base path itself exists but we don't recognise the format → ''.
		if _, err := os.Stat(fnStr.V); err == nil {
			return &object.Str{V: ""}, nil
		}
		return object.None, nil
	}})

	// dbm.sqlite3 submodule
	sub := &object.Module{Name: "dbm.sqlite3", Dict: object.NewDict()}
	subErrCls := &object.Class{Name: "error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	sub.Dict.SetStr("error", subErrCls)
	sub.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "open() requires filename")
		}
		fnStr, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "open() filename must be str")
		}
		flag := "r"
		if len(args) >= 2 {
			if f, ok2 := args[1].(*object.Str); ok2 {
				flag = f.V
			}
		}
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("flag"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					flag = s.V
				}
			}
		}
		return makeDb(fnStr.V, flag)
	}})
	m.Dict.SetStr("sqlite3", sub)

	return m
}

func (i *Interp) makeDbmObject(store *dbmStore, errCls *object.Class) *object.Instance {
	cls := &object.Class{Name: "_Database", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	checkOpen := func() error {
		if store.closed {
			return object.Errorf(errCls, "I/O operation on closed database")
		}
		return nil
	}
	checkWrite := func() error {
		if store.readOnly {
			return object.Errorf(errCls, "database is read-only")
		}
		return nil
	}

	keyErr := func(key string) error {
		return object.Errorf(i.keyErr, "%q", key)
	}

	getItem := &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__getitem__ requires key")
		}
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		v, ok := store.data[k]
		store.mu.Unlock()
		if !ok {
			return nil, keyErr(k)
		}
		cp := make([]byte, len(v))
		copy(cp, v)
		return &object.Bytes{V: cp}, nil
	}}

	setItem := &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if err := checkWrite(); err != nil {
			return nil, err
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires key and value")
		}
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		v, err := dbmVal(args[1])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		store.data[k] = v
		store.mu.Unlock()
		return object.None, nil
	}}

	delItem := &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if err := checkWrite(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__delitem__ requires key")
		}
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		_, ok := store.data[k]
		if ok {
			delete(store.data, k)
		}
		store.mu.Unlock()
		if !ok {
			return nil, keyErr(k)
		}
		return object.None, nil
	}}

	contains := &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "__contains__ requires key")
		}
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		_, ok := store.data[k]
		store.mu.Unlock()
		return object.BoolOf(ok), nil
	}}

	keys := &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		store.mu.Lock()
		ks := make([]object.Object, 0, len(store.data))
		for k := range store.data {
			b := []byte(k)
			cp := make([]byte, len(b))
			copy(cp, b)
			ks = append(ks, &object.Bytes{V: cp})
		}
		store.mu.Unlock()
		return &object.List{V: ks}, nil
	}}

	getMethod := &object.BuiltinFunc{Name: "get", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
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
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		v, ok := store.data[k]
		store.mu.Unlock()
		if !ok {
			return def, nil
		}
		cp := make([]byte, len(v))
		copy(cp, v)
		return &object.Bytes{V: cp}, nil
	}}

	setdefault := &object.BuiltinFunc{Name: "setdefault", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "setdefault() requires key and default")
		}
		k, err := dbmKey(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		v, ok := store.data[k]
		store.mu.Unlock()
		if ok {
			cp := make([]byte, len(v))
			copy(cp, v)
			return &object.Bytes{V: cp}, nil
		}
		// Not found: set the default.
		defBytes, err := dbmVal(args[1])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		store.mu.Lock()
		store.data[k] = defBytes
		store.mu.Unlock()
		cp := make([]byte, len(defBytes))
		copy(cp, defBytes)
		return &object.Bytes{V: cp}, nil
	}}

	clearMethod := &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if err := checkWrite(); err != nil {
			return nil, err
		}
		store.mu.Lock()
		store.data = map[string][]byte{}
		store.mu.Unlock()
		return object.None, nil
	}}

	syncFn := func() error {
		store.mu.Lock()
		err := store.saveFile()
		store.mu.Unlock()
		return err
	}

	closeFn := &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if store.closed {
			return object.None, nil
		}
		if err := syncFn(); err != nil {
			return nil, object.Errorf(errCls, "dbm close: %v", err)
		}
		store.closed = true
		return object.None, nil
	}}

	enter := &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}}

	exit := &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if store.closed {
			return object.None, nil
		}
		if err := syncFn(); err != nil {
			return nil, object.Errorf(errCls, "dbm close: %v", err)
		}
		store.closed = true
		return object.None, nil
	}}

	d.SetStr("__getitem__", getItem)
	d.SetStr("__setitem__", setItem)
	d.SetStr("__delitem__", delItem)
	d.SetStr("__contains__", contains)
	d.SetStr("keys", keys)
	d.SetStr("get", getMethod)
	d.SetStr("setdefault", setdefault)
	d.SetStr("clear", clearMethod)
	d.SetStr("close", closeFn)
	d.SetStr("__enter__", enter)
	d.SetStr("__exit__", exit)

	return inst
}

// buildDbmSqlite3 returns a standalone dbm.sqlite3 submodule for direct import.
func (i *Interp) buildDbmSqlite3() *object.Module {
	m := &object.Module{Name: "dbm.sqlite3", Dict: object.NewDict()}

	errCls := &object.Class{Name: "error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("error", errCls)

	makeDb := func(filename, flag string) (object.Object, error) {
		store, err := loadDbmStore(filename, flag)
		if err != nil {
			return nil, object.Errorf(errCls, "dbm.sqlite3.open: %v", err)
		}
		return i.makeDbmObject(store, errCls), nil
	}

	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "open() requires filename")
		}
		fnStr, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "open() filename must be str")
		}
		flag := "r"
		if len(args) >= 2 {
			if f, ok2 := args[1].(*object.Str); ok2 {
				flag = f.V
			}
		}
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("flag"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					flag = s.V
				}
			}
		}
		return makeDb(fnStr.V, flag)
	}})

	return m
}
