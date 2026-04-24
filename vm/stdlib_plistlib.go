package vm

import (
	"time"

	"github.com/tamnd/goipy/lib/plist"
	"github.com/tamnd/goipy/object"
)

// buildPlistlib registers the plistlib module.
func (i *Interp) buildPlistlib() *object.Module {
	m := &object.Module{Name: "plistlib", Dict: object.NewDict()}

	// PlistFormat enum-like class (CPython uses an enum.IntEnum subclass).
	fmtCls := &object.Class{Name: "PlistFormat", Dict: object.NewDict()}
	fmtCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "<PlistFormat>"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "<PlistFormat>"}, nil
		}
		name := ""
		val := ""
		if nm, ok2 := inst.Dict.GetStr("_name_"); ok2 {
			name = object.Str_(nm)
		}
		if vv, ok2 := inst.Dict.GetStr("value"); ok2 {
			val = object.Repr(vv)
		}
		return &object.Str{V: "<PlistFormat." + name + ": " + val + ">"}, nil
	}})
	if reprFn, ok := fmtCls.Dict.GetStr("__repr__"); ok {
		fmtCls.Dict.SetStr("__str__", reprFn)
	}
	fmtXML := &object.Instance{Class: fmtCls, Dict: object.NewDict()}
	fmtXML.Dict.SetStr("value", object.NewInt(plist.FMT_XML))
	fmtXML.Dict.SetStr("_name_", &object.Str{V: "FMT_XML"})
	fmtBinary := &object.Instance{Class: fmtCls, Dict: object.NewDict()}
	fmtBinary.Dict.SetStr("value", object.NewInt(plist.FMT_BINARY))
	fmtBinary.Dict.SetStr("_name_", &object.Str{V: "FMT_BINARY"})
	m.Dict.SetStr("FMT_XML", fmtXML)
	m.Dict.SetStr("FMT_BINARY", fmtBinary)

	// InvalidFileException
	invalidFileCls := &object.Class{
		Name:  "InvalidFileException",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("InvalidFileException", invalidFileCls)

	// UID class
	uidCls := &object.Class{Name: "UID", Dict: object.NewDict()}
	uidCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "UID() requires an integer argument")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		n, ok2 := plistInt64(a[1])
		if !ok2 || n < 0 {
			return nil, object.Errorf(i.valueErr, "UID() integer must be in range 0..2^64-1")
		}
		self.Dict.SetStr("data", object.NewInt(n))
		return object.None, nil
	}})
	uidCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "UID(0)"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "UID(0)"}, nil
		}
		n := int64(0)
		if v, ok2 := self.Dict.GetStr("data"); ok2 {
			n, _ = plistInt64(v)
		}
		return &object.Str{V: "UID(" + object.Repr(object.NewInt(n)) + ")"}, nil
	}})
	uidCls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		selfInst, ok1 := a[0].(*object.Instance)
		otherInst, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return object.False, nil
		}
		sv, _ := selfInst.Dict.GetStr("data")
		ov, _ := otherInst.Dict.GetStr("data")
		sn, _ := plistInt64(sv)
		on, _ := plistInt64(ov)
		return object.BoolOf(sn == on), nil
	}})
	uidCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		if v, ok2 := self.Dict.GetStr("data"); ok2 {
			return v, nil
		}
		return object.NewInt(0), nil
	}})
	m.Dict.SetStr("UID", uidCls)

	// ── loads ─────────────────────────────────────────────────────────────────

	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "loads() requires a data argument")
		}
		var data []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			data = v.V
		case *object.Str:
			data = []byte(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "loads() argument must be bytes or str")
		}
		fmtID, awareDT, _ := plistLoadKwArgs(kw)
		val, err := plist.Loads(data, fmtID)
		if err != nil {
			return nil, object.Errorf(invalidFileCls, "%s", err.Error())
		}
		return i.plistValToObj(val, awareDT, uidCls)
	}})

	// ── load ──────────────────────────────────────────────────────────────────

	m.Dict.SetStr("load", &object.BuiltinFunc{Name: "load", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "load() requires a file argument")
		}
		data, err := i.plistReadFile(a[0])
		if err != nil {
			return nil, err
		}
		fmtID, awareDT, _ := plistLoadKwArgs(kw)
		val, err2 := plist.Loads(data, fmtID)
		if err2 != nil {
			return nil, object.Errorf(invalidFileCls, "%s", err2.Error())
		}
		return i.plistValToObj(val, awareDT, uidCls)
	}})

	// ── dumps ─────────────────────────────────────────────────────────────────

	m.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "dumps() requires a value argument")
		}
		fmtID, sortKeys, skipKeys := plistDumpKwArgs(kw)
		goVal, err := i.plistObjToVal(a[0], skipKeys, uidCls)
		if err != nil {
			return nil, err
		}
		b, err2 := plist.Dumps(goVal, fmtID, sortKeys)
		if err2 != nil {
			return nil, object.Errorf(i.typeErr, "%s", err2.Error())
		}
		return &object.Bytes{V: b}, nil
	}})

	// ── dump ──────────────────────────────────────────────────────────────────

	m.Dict.SetStr("dump", &object.BuiltinFunc{Name: "dump", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "dump() requires value and file arguments")
		}
		fmtID, sortKeys, skipKeys := plistDumpKwArgs(kw)
		goVal, err := i.plistObjToVal(a[0], skipKeys, uidCls)
		if err != nil {
			return nil, err
		}
		b, err2 := plist.Dumps(goVal, fmtID, sortKeys)
		if err2 != nil {
			return nil, object.Errorf(i.typeErr, "%s", err2.Error())
		}
		writeFn, err3 := i.getAttr(a[1], "write")
		if err3 != nil {
			return nil, object.Errorf(i.typeErr, "dump() file argument must have a write() method")
		}
		if _, err4 := i.callObject(writeFn, []object.Object{&object.Bytes{V: b}}, nil); err4 != nil {
			return nil, err4
		}
		return object.None, nil
	}})

	return m
}

// ── kwarg helpers ─────────────────────────────────────────────────────────────

func plistLoadKwArgs(kw *object.Dict) (fmtID int, awareDT bool, dictType object.Object) {
	fmtID = 0
	awareDT = false
	dictType = nil
	if kw == nil {
		return
	}
	if v, ok := kw.GetStr("fmt"); ok {
		fmtID = plistFmtID(v)
	}
	if v, ok := kw.GetStr("aware_datetime"); ok {
		awareDT = object.Truthy(v)
	}
	if v, ok := kw.GetStr("dict_type"); ok {
		dictType = v
	}
	return
}

func plistDumpKwArgs(kw *object.Dict) (fmtID int, sortKeys bool, skipKeys bool) {
	fmtID = plist.FMT_XML
	sortKeys = true
	skipKeys = false
	if kw == nil {
		return
	}
	if v, ok := kw.GetStr("fmt"); ok {
		fmtID = plistFmtID(v)
	}
	if v, ok := kw.GetStr("sort_keys"); ok {
		sortKeys = object.Truthy(v)
	}
	if v, ok := kw.GetStr("skipkeys"); ok {
		skipKeys = object.Truthy(v)
	}
	return
}

// plistFmtID extracts the integer format ID from a FMT_XML/FMT_BINARY object,
// which may be a plain *object.Int or a PlistFormat enum instance with .value.
func plistFmtID(v object.Object) int {
	if n, ok := v.(*object.Int); ok {
		return int(n.Int64())
	}
	if inst, ok := v.(*object.Instance); ok {
		if val, ok2 := inst.Dict.GetStr("value"); ok2 {
			if n, ok3 := val.(*object.Int); ok3 {
				return int(n.Int64())
			}
		}
	}
	return 0
}

// ── Go value → Python object ─────────────────────────────────────────────────

func (i *Interp) plistValToObj(v plist.Value, awareDT bool, uidCls *object.Class) (object.Object, error) {
	if v == nil {
		return object.None, nil
	}
	switch x := v.(type) {
	case bool:
		return object.BoolOf(x), nil
	case int64:
		return object.NewInt(x), nil
	case uint64:
		if x <= 1<<63-1 {
			return object.NewInt(int64(x)), nil
		}
		return &object.Float{V: float64(x)}, nil // rare: >MaxInt64
	case float64:
		return &object.Float{V: x}, nil
	case string:
		return &object.Str{V: x}, nil
	case []byte:
		return &object.Bytes{V: x}, nil
	case time.Time:
		return i.plistTimeToObj(x, awareDT)
	case plist.UID:
		inst := &object.Instance{Class: uidCls, Dict: object.NewDict()}
		inst.Dict.SetStr("data", object.NewInt(int64(x.Data)))
		return inst, nil
	case []interface{}:
		items := make([]object.Object, len(x))
		for k, elem := range x {
			obj, err := i.plistValToObj(elem, awareDT, uidCls)
			if err != nil {
				return nil, err
			}
			items[k] = obj
		}
		return &object.List{V: items}, nil
	case map[string]interface{}:
		d := object.NewDict()
		for k, val := range x {
			obj, err := i.plistValToObj(val, awareDT, uidCls)
			if err != nil {
				return nil, err
			}
			d.SetStr(k, obj)
		}
		return d, nil
	}
	return object.None, nil
}

func (i *Interp) plistTimeToObj(t time.Time, awareDT bool) (object.Object, error) {
	dtDict, err := i.dtModule()
	if err != nil {
		return nil, err
	}
	dtCls := i.dtClass(dtDict, "datetime")
	dateCls := i.dtClass(dtDict, "date")
	if awareDT {
		tzCls := i.dtClass(dtDict, "timezone")
		td := normTimedelta(0, 0, 0)
		tzInst := i.makeTZInstance(tzCls, goTimezone{offset: td})
		d := goDatetime{
			goDate: goDate{t.Year(), int(t.Month()), t.Day()},
			goTime: goTime{t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / 1000, tzInst, 0},
		}
		return i.makeDTInstance(dtCls, dateCls, d), nil
	}
	d := goDatetime{
		goDate: goDate{t.Year(), int(t.Month()), t.Day()},
		goTime: goTime{t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / 1000, object.None, 0},
	}
	return i.makeDTInstance(dtCls, dateCls, d), nil
}

// ── Python object → Go value ─────────────────────────────────────────────────

func (i *Interp) plistObjToVal(o object.Object, skipKeys bool, uidCls *object.Class) (plist.Value, error) {
	switch x := o.(type) {
	case *object.NoneType:
		return nil, object.Errorf(i.typeErr, "plistlib: cannot serialize None")
	case *object.Bool:
		return x == object.True, nil
	case *object.Int:
		return x.Int64(), nil
	case *object.Float:
		return x.V, nil
	case *object.Str:
		return x.V, nil
	case *object.Bytes:
		return x.V, nil
	case *object.Bytearray:
		return []byte(x.V), nil
	case *object.Tuple:
		items := make([]interface{}, len(x.V))
		for k, elem := range x.V {
			v, err := i.plistObjToVal(elem, skipKeys, uidCls)
			if err != nil {
				return nil, err
			}
			items[k] = v
		}
		return items, nil
	case *object.List:
		items := make([]interface{}, len(x.V))
		for k, elem := range x.V {
			v, err := i.plistObjToVal(elem, skipKeys, uidCls)
			if err != nil {
				return nil, err
			}
			items[k] = v
		}
		return items, nil
	case *object.Dict:
		m := map[string]interface{}{}
		ks, vs := x.Items()
		for idx, kObj := range ks {
			ks2, ok := kObj.(*object.Str)
			if !ok {
				if skipKeys {
					continue
				}
				return nil, object.Errorf(i.typeErr, "plistlib: keys must be strings")
			}
			v, err := i.plistObjToVal(vs[idx], skipKeys, uidCls)
			if err != nil {
				return nil, err
			}
			m[ks2.V] = v
		}
		return m, nil
	case *object.Instance:
		if x.Class == uidCls {
			if dv, ok2 := x.Dict.GetStr("data"); ok2 {
				if n, ok3 := dv.(*object.Int); ok3 {
					return plist.UID{Data: uint64(n.Int64())}, nil
				}
			}
		}
		// datetime.datetime?
		return i.plistInstanceToTime(x)
	}
	return nil, object.Errorf(i.typeErr, "plistlib: unsupported type %s", object.TypeName(o))
}

func (i *Interp) plistInstanceToTime(inst *object.Instance) (time.Time, error) {
	get := func(key string) int {
		if v, ok := inst.Dict.GetStr(key); ok {
			if n, ok2 := v.(*object.Int); ok2 && n.IsInt64() {
				return int(n.Int64())
			}
		}
		return 0
	}
	year := get("year")
	month := get("month")
	day := get("day")
	hour := get("hour")
	minute := get("minute")
	second := get("second")
	usec := get("microsecond")
	if year == 0 {
		return time.Time{}, object.Errorf(i.typeErr, "plistlib: unsupported type %s", inst.Class.Name)
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, usec*1000, time.UTC), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (i *Interp) plistReadFile(fp object.Object) ([]byte, error) {
	readFn, err := i.getAttr(fp, "read")
	if err != nil {
		return nil, object.Errorf(i.typeErr, "file argument must have a read() method")
	}
	data, err := i.callObject(readFn, nil, nil)
	if err != nil {
		return nil, err
	}
	switch v := data.(type) {
	case *object.Bytes:
		return v.V, nil
	case *object.Str:
		return []byte(v.V), nil
	}
	return nil, object.Errorf(i.typeErr, "read() must return bytes or str")
}

// plistInt64 extracts an int64 from a Python int/bool object.
func plistInt64(o object.Object) (int64, bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	case *object.Int:
		if v.IsInt64() {
			return v.Int64(), true
		}
	}
	return 0, false
}
