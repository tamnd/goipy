package vm

import (
	"github.com/tamnd/goipy/lib/toml"
	"github.com/tamnd/goipy/object"
)

// buildTomllib registers the tomllib module.
func (i *Interp) buildTomllib() *object.Module {
	m := &object.Module{Name: "tomllib", Dict: object.NewDict()}

	decodeErr := &object.Class{Name: "TOMLDecodeError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("TOMLDecodeError", decodeErr)

	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "loads() requires a string argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "loads() argument must be str")
		}
		result, err := toml.Loads(s.V)
		if err != nil {
			return nil, object.Errorf(decodeErr, "%s", err.Error())
		}
		return i.tomlMapToDict(result, decodeErr)
	}})

	m.Dict.SetStr("load", &object.BuiltinFunc{Name: "load", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "load() requires a file object")
		}
		// Read bytes from file-like object.
		readFn, err := i.getAttr(a[0], "read")
		if err != nil {
			return nil, object.Errorf(i.typeErr, "load() argument must have a read() method")
		}
		data, err := i.callObject(readFn, nil, nil)
		if err != nil {
			return nil, err
		}
		var s string
		switch v := data.(type) {
		case *object.Bytes:
			s = string(v.V)
		case *object.Str:
			s = v.V
		default:
			return nil, object.Errorf(i.typeErr, "load() read() must return bytes")
		}
		result, err2 := toml.Loads(s)
		if err2 != nil {
			return nil, object.Errorf(decodeErr, "%s", err2.Error())
		}
		return i.tomlMapToDict(result, decodeErr)
	}})

	return m
}

// tomlMapToDict converts a TOML map[string]interface{} to an *object.Dict.
func (i *Interp) tomlMapToDict(m map[string]interface{}, decodeErr *object.Class) (*object.Dict, error) {
	d := object.NewDict()
	for k, v := range m {
		obj, err := i.tomlValToObj(v, decodeErr)
		if err != nil {
			return nil, err
		}
		d.SetStr(k, obj)
	}
	return d, nil
}

// tomlValToObj converts a TOML Go value to an object.Object.
func (i *Interp) tomlValToObj(v interface{}, decodeErr *object.Class) (object.Object, error) {
	switch vt := v.(type) {
	case string:
		return &object.Str{V: vt}, nil
	case int64:
		return object.NewInt(vt), nil
	case float64:
		return &object.Float{V: vt}, nil
	case bool:
		return object.BoolOf(vt), nil
	case map[string]interface{}:
		return i.tomlMapToDict(vt, decodeErr)
	case []interface{}:
		items := make([]object.Object, len(vt))
		for k, elem := range vt {
			obj, err := i.tomlValToObj(elem, decodeErr)
			if err != nil {
				return nil, err
			}
			items[k] = obj
		}
		return &object.List{V: items}, nil
	case toml.OffsetDateTime:
		return i.tomlOffsetDateTime(vt)
	case toml.LocalDateTime:
		return i.tomlLocalDateTime(vt)
	case toml.LocalDate:
		return i.tomlLocalDate(vt)
	case toml.LocalTime:
		return i.tomlLocalTime(vt)
	default:
		return object.None, nil
	}
}

// dtModule lazily loads the datetime module and returns its dict.
func (i *Interp) dtModule() (*object.Dict, error) {
	m, err := i.loadModule("datetime")
	if err != nil {
		return nil, err
	}
	return m.Dict, nil
}

func (i *Interp) tomlOffsetDateTime(dt toml.OffsetDateTime) (object.Object, error) {
	dtDict, err := i.dtModule()
	if err != nil {
		return nil, err
	}
	// Build timezone.
	tzCls := i.dtClass(dtDict, "timezone")
	tdCls := i.dtClass(dtDict, "timedelta")
	dtCls := i.dtClass(dtDict, "datetime")
	dateCls := i.dtClass(dtDict, "date")
	if tzCls == nil || tdCls == nil || dtCls == nil || dateCls == nil {
		return object.None, nil
	}
	td := normTimedelta(0, int64(dt.TZOffsetSecs), 0)
	tzInst := i.makeTZInstance(tzCls, goTimezone{offset: td})
	d := goDatetime{
		goDate: goDate{dt.Year, dt.Month, dt.Day},
		goTime: goTime{dt.Hour, dt.Minute, dt.Second, dt.Microsecond, tzInst, 0},
	}
	return i.makeDTInstance(dtCls, dateCls, d), nil
}

func (i *Interp) tomlLocalDateTime(dt toml.LocalDateTime) (object.Object, error) {
	dtDict, err := i.dtModule()
	if err != nil {
		return nil, err
	}
	dtCls := i.dtClass(dtDict, "datetime")
	dateCls := i.dtClass(dtDict, "date")
	if dtCls == nil || dateCls == nil {
		return object.None, nil
	}
	d := goDatetime{
		goDate: goDate{dt.Year, dt.Month, dt.Day},
		goTime: goTime{dt.Hour, dt.Minute, dt.Second, dt.Microsecond, object.None, 0},
	}
	return i.makeDTInstance(dtCls, dateCls, d), nil
}

func (i *Interp) tomlLocalDate(ld toml.LocalDate) (object.Object, error) {
	dtDict, err := i.dtModule()
	if err != nil {
		return nil, err
	}
	dateCls := i.dtClass(dtDict, "date")
	if dateCls == nil {
		return object.None, nil
	}
	return i.makeDateInstance(dateCls, goDate{ld.Year, ld.Month, ld.Day}), nil
}

func (i *Interp) tomlLocalTime(lt toml.LocalTime) (object.Object, error) {
	dtDict, err := i.dtModule()
	if err != nil {
		return nil, err
	}
	timeCls := i.dtClass(dtDict, "time")
	if timeCls == nil {
		return object.None, nil
	}
	return i.makeTimeInstance(timeCls, goTime{lt.Hour, lt.Minute, lt.Second, lt.Microsecond, object.None, 0}), nil
}

func (i *Interp) dtClass(dtDict *object.Dict, name string) *object.Class {
	v, ok := dtDict.GetStr(name)
	if !ok {
		return nil
	}
	cls, ok := v.(*object.Class)
	if !ok {
		return nil
	}
	return cls
}
