package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// moduleErrno is a module-level fake errno for get_errno / set_errno.
var ctypesErrno int64

// buildCtypes constructs the ctypes module: simple C types, sizeof,
// Structure/Union bases, POINTER/pointer/byref/cast/addressof,
// create_string_buffer, CDLL stub, get/set_errno, and constants.
func (i *Interp) buildCtypes() *object.Module {
	m := &object.Module{Name: "ctypes", Dict: object.NewDict()}

	// --- Simple types ---
	cBool := buildSimpleType("c_bool", 1, object.False)
	cByte := buildSimpleType("c_byte", 1, object.NewInt(0))
	cUbyte := buildSimpleType("c_ubyte", 1, object.NewInt(0))
	cChar := buildSimpleType("c_char", 1, &object.Bytes{V: []byte{0}})
	cWchar := buildSimpleType("c_wchar", 2, &object.Str{V: "\x00"})
	cShort := buildSimpleType("c_short", 2, object.NewInt(0))
	cUshort := buildSimpleType("c_ushort", 2, object.NewInt(0))
	cInt := buildSimpleType("c_int", 4, object.NewInt(0))
	cUint := buildSimpleType("c_uint", 4, object.NewInt(0))
	cLong := buildSimpleType("c_long", 8, object.NewInt(0))
	cUlong := buildSimpleType("c_ulong", 8, object.NewInt(0))
	cLonglong := buildSimpleType("c_longlong", 8, object.NewInt(0))
	cUlonglong := buildSimpleType("c_ulonglong", 8, object.NewInt(0))
	cSizeT := buildSimpleType("c_size_t", 8, object.NewInt(0))
	cSsizeT := buildSimpleType("c_ssize_t", 8, object.NewInt(0))
	cFloat := buildSimpleType("c_float", 4, &object.Float{V: 0})
	cDouble := buildSimpleType("c_double", 8, &object.Float{V: 0})
	cLongdouble := buildSimpleType("c_longdouble", 16, &object.Float{V: 0})
	cCharP := buildSimpleType("c_char_p", 8, object.None)
	cWcharP := buildSimpleType("c_wchar_p", 8, object.None)
	cVoidP := buildSimpleType("c_void_p", 8, object.None)

	// Mark pointer types so None is a valid value (don't overwrite user's None
	// with zero).
	for _, pc := range []*object.Class{cCharP, cWcharP, cVoidP} {
		pc.Dict.SetStr("_is_pointer_", object.True)
	}

	m.Dict.SetStr("c_bool", cBool)
	m.Dict.SetStr("c_byte", cByte)
	m.Dict.SetStr("c_ubyte", cUbyte)
	m.Dict.SetStr("c_char", cChar)
	m.Dict.SetStr("c_wchar", cWchar)
	m.Dict.SetStr("c_short", cShort)
	m.Dict.SetStr("c_ushort", cUshort)
	m.Dict.SetStr("c_int", cInt)
	m.Dict.SetStr("c_uint", cUint)
	m.Dict.SetStr("c_long", cLong)
	m.Dict.SetStr("c_ulong", cUlong)
	m.Dict.SetStr("c_longlong", cLonglong)
	m.Dict.SetStr("c_ulonglong", cUlonglong)
	m.Dict.SetStr("c_size_t", cSizeT)
	m.Dict.SetStr("c_ssize_t", cSsizeT)
	m.Dict.SetStr("c_float", cFloat)
	m.Dict.SetStr("c_double", cDouble)
	m.Dict.SetStr("c_longdouble", cLongdouble)
	m.Dict.SetStr("c_char_p", cCharP)
	m.Dict.SetStr("c_wchar_p", cWcharP)
	m.Dict.SetStr("c_void_p", cVoidP)

	// --- Structure base ---
	structureCls := &object.Class{Name: "Structure", Dict: object.NewDict()}
	structureCls.Dict.SetStr("_is_structure_", object.True)
	structureCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		args := a[1:]
		fields := ctypesGetFields(self.Class)
		for idx, f := range fields {
			name := f[0]
			var val object.Object = object.NewInt(0)
			if idx < len(args) {
				val = args[idx]
			} else if kw != nil {
				if v, ok2 := kw.GetStr(name); ok2 {
					val = v
				}
			}
			self.Dict.SetStr(name, val)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("Structure", structureCls)

	// --- Union base ---
	unionCls := &object.Class{Name: "Union", Dict: object.NewDict()}
	unionCls.Dict.SetStr("_is_union_", object.True)
	unionCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		args := a[1:]
		fields := ctypesGetFields(self.Class)
		for idx, f := range fields {
			name := f[0]
			var val object.Object = object.NewInt(0)
			if idx < len(args) {
				val = args[idx]
			} else if kw != nil {
				if v, ok2 := kw.GetStr(name); ok2 {
					val = v
				}
			}
			self.Dict.SetStr(name, val)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("Union", unionCls)

	// --- Array base ---
	arrayCls := &object.Class{Name: "Array", Dict: object.NewDict()}
	m.Dict.SetStr("Array", arrayCls)

	// --- sizeof ---
	sizeofFn := &object.BuiltinFunc{Name: "sizeof", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("sizeof() requires argument")
		}
		sz := ctypesSizeofObj(a[0])
		if sz < 0 {
			return nil, fmt.Errorf("ctypes.sizeof: not a ctypes type or instance")
		}
		return object.NewInt(sz), nil
	}}
	m.Dict.SetStr("sizeof", sizeofFn)

	// --- alignment ---
	alignmentFn := &object.BuiltinFunc{Name: "alignment", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("alignment() requires argument")
		}
		sz := ctypesAlignObj(a[0])
		if sz < 0 {
			return nil, fmt.Errorf("ctypes.alignment: not a ctypes type or instance")
		}
		return object.NewInt(sz), nil
	}}
	m.Dict.SetStr("alignment", alignmentFn)

	// --- byref ---
	byrefCls := &object.Class{Name: "CArgObject", Dict: object.NewDict()}
	byrefFn := &object.BuiltinFunc{Name: "byref", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: byrefCls, Dict: object.NewDict()}
		if len(a) >= 1 {
			inst.Dict.SetStr("_obj", a[0])
		}
		return inst, nil
	}}
	m.Dict.SetStr("byref", byrefFn)

	// --- POINTER(type) ---
	pointerFn := &object.BuiltinFunc{Name: "POINTER", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("POINTER() requires a type argument")
		}
		inner, ok := a[0].(*object.Class)
		if !ok {
			return nil, fmt.Errorf("POINTER() argument must be a ctypes type")
		}
		ptrName := "LP_" + inner.Name
		ptrCls := &object.Class{Name: ptrName, Dict: object.NewDict()}
		ptrCls.Dict.SetStr("_size_", object.NewInt(8))
		ptrCls.Dict.SetStr("__name__", &object.Str{V: ptrName})
		ptrCls.Dict.SetStr("_type_", inner)
		ptrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) >= 1 {
				if self, ok2 := a2[0].(*object.Instance); ok2 {
					if len(a2) >= 2 {
						self.Dict.SetStr("contents", a2[1])
					}
				}
			}
			return object.None, nil
		}})
		return ptrCls, nil
	}}
	m.Dict.SetStr("POINTER", pointerFn)

	// --- pointer(obj) ---
	pointerInstFn := &object.BuiltinFunc{Name: "pointer", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("pointer() requires an argument")
		}
		// Get the type of the object
		var innerCls *object.Class
		if inst, ok := a[0].(*object.Instance); ok {
			innerCls = inst.Class
		} else {
			innerCls = &object.Class{Name: "c_void", Dict: object.NewDict()}
		}
		ptrName := "LP_" + innerCls.Name
		ptrCls := &object.Class{Name: ptrName, Dict: object.NewDict()}
		ptrCls.Dict.SetStr("_size_", object.NewInt(8))
		ptrCls.Dict.SetStr("__name__", &object.Str{V: ptrName})
		ptrInst := &object.Instance{Class: ptrCls, Dict: object.NewDict()}
		ptrInst.Dict.SetStr("contents", a[0])
		return ptrInst, nil
	}}
	m.Dict.SetStr("pointer", pointerInstFn)

	// --- cast(obj, type) ---
	castFn := &object.BuiltinFunc{Name: "cast", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("cast() requires 2 arguments")
		}
		targetCls, ok := a[1].(*object.Class)
		if !ok {
			return nil, fmt.Errorf("cast(): second argument must be a ctypes type")
		}
		inst := &object.Instance{Class: targetCls, Dict: object.NewDict()}
		// Copy value from source if available
		if src, ok2 := a[0].(*object.Instance); ok2 {
			if v, ok3 := src.Dict.GetStr("value"); ok3 {
				inst.Dict.SetStr("value", v)
			}
		}
		// Set a default value if none was copied
		if _, ok2 := inst.Dict.GetStr("value"); !ok2 {
			inst.Dict.SetStr("value", object.NewInt(0))
		}
		return inst, nil
	}}
	m.Dict.SetStr("cast", castFn)

	// --- addressof(obj) ---
	addressofFn := &object.BuiltinFunc{Name: "addressof", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// Return 0 as a fake address (pure interpreter cannot know real address)
		return object.NewInt(0), nil
	}}
	m.Dict.SetStr("addressof", addressofFn)

	// --- create_string_buffer ---
	createStringBufFn := &object.BuiltinFunc{Name: "create_string_buffer", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("create_string_buffer() requires at least 1 argument")
		}
		var data []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			data = make([]byte, len(v.V)+1) // null-terminated
			copy(data, v.V)
		case *object.Int:
			n := v.Int64()
			if n < 0 {
				return nil, fmt.Errorf("create_string_buffer: size must be non-negative")
			}
			data = make([]byte, n)
		default:
			return nil, fmt.Errorf("create_string_buffer: argument must be bytes or int")
		}
		// Optional size override
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok && n > 0 {
				newData := make([]byte, n)
				copy(newData, data)
				data = newData
			}
		}
		return ctypesMakeStringBuffer(data), nil
	}}
	m.Dict.SetStr("create_string_buffer", createStringBufFn)

	// --- create_unicode_buffer ---
	createUnicodeBufFn := &object.BuiltinFunc{Name: "create_unicode_buffer", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, fmt.Errorf("create_unicode_buffer() requires at least 1 argument")
		}
		var size int
		switch v := a[0].(type) {
		case *object.Str:
			size = len([]rune(v.V)) + 1
		case *object.Int:
			n := v.Int64()
			if n < 0 {
				return nil, fmt.Errorf("create_unicode_buffer: size must be non-negative")
			}
			size = int(n)
		default:
			return nil, fmt.Errorf("create_unicode_buffer: argument must be str or int")
		}
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok && n > 0 {
				size = int(n)
			}
		}
		return ctypesMakeUnicodeBuffer(size), nil
	}}
	m.Dict.SetStr("create_unicode_buffer", createUnicodeBufFn)

	// --- string_at stub ---
	m.Dict.SetStr("string_at", &object.BuiltinFunc{Name: "string_at", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{}}, nil
	}})

	// --- wstring_at stub ---
	m.Dict.SetStr("wstring_at", &object.BuiltinFunc{Name: "wstring_at", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})

	// --- get_errno / set_errno ---
	m.Dict.SetStr("get_errno", &object.BuiltinFunc{Name: "get_errno", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(ctypesErrno), nil
	}})
	m.Dict.SetStr("set_errno", &object.BuiltinFunc{Name: "set_errno", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				ctypesErrno = n
			}
		}
		return object.None, nil
	}})

	// --- FormatError stub ---
	m.Dict.SetStr("FormatError", &object.BuiltinFunc{Name: "FormatError", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})

	// --- WinError stub ---
	m.Dict.SetStr("WinError", &object.BuiltinFunc{Name: "WinError", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		code := int64(0)
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				code = n
			}
		}
		return nil, object.Errorf(i.osErr, "Windows Error %d", code)
	}})

	// --- memmove / memset stubs ---
	m.Dict.SetStr("memmove", &object.BuiltinFunc{Name: "memmove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			return a[0], nil
		}
		return object.NewInt(0), nil
	}})
	m.Dict.SetStr("memset", &object.BuiltinFunc{Name: "memset", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			return a[0], nil
		}
		return object.NewInt(0), nil
	}})

	// --- CDLL and friends ---
	cdllCls := buildCDLLClass("CDLL")
	winDLLCls := buildCDLLClass("WinDLL")
	oleDLLCls := buildCDLLClass("OleDLL")
	pyDLLCls := buildCDLLClass("PyDLL")
	m.Dict.SetStr("CDLL", cdllCls)
	m.Dict.SetStr("WinDLL", winDLLCls)
	m.Dict.SetStr("OleDLL", oleDLLCls)
	m.Dict.SetStr("PyDLL", pyDLLCls)

	// --- LibraryLoader ---
	libLoaderCls := &object.Class{Name: "LibraryLoader", Dict: object.NewDict()}
	libLoaderCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if self, ok := a[0].(*object.Instance); ok {
				self.Dict.SetStr("_dlltype", a[1])
			}
		}
		return object.None, nil
	}})
	libLoaderCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// a[0] = self, a[1] = name string
		// Returns a new CDLL instance for the named library
		if len(a) >= 2 {
			if self, ok := a[0].(*object.Instance); ok {
				var dlltype *object.Class = cdllCls
				if dt, ok2 := self.Dict.GetStr("_dlltype"); ok2 {
					if cls, ok3 := dt.(*object.Class); ok3 {
						dlltype = cls
					}
				}
				inst := &object.Instance{Class: dlltype, Dict: object.NewDict()}
				inst.Dict.SetStr("_name", a[1])
				return inst, nil
			}
		}
		return object.None, nil
	}})
	m.Dict.SetStr("LibraryLoader", libLoaderCls)

	// Shared library loader instances
	cdllLoader := &object.Instance{Class: libLoaderCls, Dict: object.NewDict()}
	cdllLoader.Dict.SetStr("_dlltype", cdllCls)
	m.Dict.SetStr("cdll", cdllLoader)

	windllLoader := &object.Instance{Class: libLoaderCls, Dict: object.NewDict()}
	windllLoader.Dict.SetStr("_dlltype", winDLLCls)
	m.Dict.SetStr("windll", windllLoader)

	oledllLoader := &object.Instance{Class: libLoaderCls, Dict: object.NewDict()}
	oledllLoader.Dict.SetStr("_dlltype", oleDLLCls)
	m.Dict.SetStr("oledll", oledllLoader)

	// --- pythonapi ---
	pythonapi := &object.Instance{Class: cdllCls, Dict: object.NewDict()}
	pythonapi.Dict.SetStr("_name", &object.Str{V: "python"})
	m.Dict.SetStr("pythonapi", pythonapi)

	// --- Constants ---
	m.Dict.SetStr("RTLD_LOCAL", object.NewInt(0))
	m.Dict.SetStr("RTLD_GLOBAL", object.NewInt(256))

	return m
}

// buildSimpleType creates a ctypes simple type class.
func buildSimpleType(name string, size int64, zeroVal object.Object) *object.Class {
	cls := &object.Class{Name: name, Dict: object.NewDict()}
	cls.Dict.SetStr("_size_", object.NewInt(size))
	zero := zeroVal // capture for closure

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		val := zero
		if len(a) >= 2 {
			val = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("value"); ok2 {
				val = v
			}
		}
		self.Dict.SetStr("value", val)
		return object.None, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: name + "()"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: name + "()"}, nil
		}
		v, _ := self.Dict.GetStr("value")
		return &object.Str{V: name + "(" + object.Repr(v) + ")"}, nil
	}})

	return cls
}

// buildCDLLClass creates a CDLL-like stub class.
func buildCDLLClass(name string) *object.Class {
	cls := &object.Class{Name: name, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			if self, ok := a[0].(*object.Instance); ok {
				var nameVal object.Object = object.None
				if len(a) >= 2 {
					nameVal = a[1]
				} else if kw != nil {
					if v, ok2 := kw.GetStr("name"); ok2 {
						nameVal = v
					}
				}
				self.Dict.SetStr("_name", nameVal)
			}
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// Return a stub callable that returns None when called
		stub := &object.BuiltinFunc{Name: "cdll_func", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
		return stub, nil
	}})
	return cls
}

// ctypesGetFields walks a class hierarchy to find _fields_ and returns
// a slice of [name, type] pairs as string slices.
func ctypesGetFields(cls *object.Class) [][]string {
	if cls == nil {
		return nil
	}
	// Check the class dict directly first
	if fv, ok := cls.Dict.GetStr("_fields_"); ok {
		return ctypesParseFields(fv)
	}
	// Walk bases
	for _, base := range cls.Bases {
		if fields := ctypesGetFields(base); fields != nil {
			return fields
		}
	}
	return nil
}

// ctypesParseFields parses a Python _fields_ list: [[name, type], ...] or
// list of 2-tuples.
func ctypesParseFields(obj object.Object) [][]string {
	lst, ok := obj.(*object.List)
	if !ok {
		return nil
	}
	var result [][]string
	for _, item := range lst.V {
		switch t := item.(type) {
		case *object.Tuple:
			if len(t.V) >= 2 {
				name := ""
				if s, ok2 := t.V[0].(*object.Str); ok2 {
					name = s.V
				}
				typeName := ""
				if cls, ok2 := t.V[1].(*object.Class); ok2 {
					typeName = cls.Name
				}
				result = append(result, []string{name, typeName})
			}
		case *object.List:
			if len(t.V) >= 2 {
				name := ""
				if s, ok2 := t.V[0].(*object.Str); ok2 {
					name = s.V
				}
				typeName := ""
				if cls, ok2 := t.V[1].(*object.Class); ok2 {
					typeName = cls.Name
				}
				result = append(result, []string{name, typeName})
			}
		}
	}
	return result
}

// ctypesSizeofClass computes the byte size of a ctypes class.
func ctypesSizeofClass(cls *object.Class) int64 {
	if cls == nil {
		return 0
	}
	// Simple types carry _size_ directly
	if v, ok := cls.Dict.GetStr("_size_"); ok {
		if n, ok2 := toInt64(v); ok2 {
			return n
		}
	}
	// Structure / Union: derive from _fields_
	isUnion := ctypesIsUnion(cls)
	fields := ctypesFieldsAndTypes(cls)
	if fields == nil {
		// Check bases
		for _, base := range cls.Bases {
			sz := ctypesSizeofClass(base)
			if sz > 0 {
				return sz
			}
		}
		return 0
	}
	total := int64(0)
	for _, ft := range fields {
		sz := ctypesSizeofClass(ft)
		if isUnion {
			if sz > total {
				total = sz
			}
		} else {
			total += sz
		}
	}
	return total
}

// ctypesFieldsAndTypes returns the type classes for each field of a Structure/Union.
func ctypesFieldsAndTypes(cls *object.Class) []*object.Class {
	if cls == nil {
		return nil
	}
	fv, ok := cls.Dict.GetStr("_fields_")
	if !ok {
		for _, base := range cls.Bases {
			if types := ctypesFieldsAndTypes(base); types != nil {
				return types
			}
		}
		return nil
	}
	lst, ok := fv.(*object.List)
	if !ok {
		return nil
	}
	var result []*object.Class
	for _, item := range lst.V {
		var fieldType *object.Class
		switch t := item.(type) {
		case *object.Tuple:
			if len(t.V) >= 2 {
				if cls2, ok2 := t.V[1].(*object.Class); ok2 {
					fieldType = cls2
				}
			}
		case *object.List:
			if len(t.V) >= 2 {
				if cls2, ok2 := t.V[1].(*object.Class); ok2 {
					fieldType = cls2
				}
			}
		}
		if fieldType != nil {
			result = append(result, fieldType)
		}
	}
	return result
}

// ctypesIsUnion checks whether a class is a Union subclass.
func ctypesIsUnion(cls *object.Class) bool {
	if cls == nil {
		return false
	}
	if v, ok := cls.Dict.GetStr("_is_union_"); ok {
		if b, ok2 := v.(*object.Bool); ok2 && b.V {
			return true
		}
	}
	for _, base := range cls.Bases {
		if ctypesIsUnion(base) {
			return true
		}
	}
	return false
}

// ctypesSizeofObj returns the byte size of a ctypes type or instance.
func ctypesSizeofObj(obj object.Object) int64 {
	switch x := obj.(type) {
	case *object.Class:
		return ctypesSizeofClass(x)
	case *object.Instance:
		return ctypesSizeofClass(x.Class)
	}
	return -1
}

// ctypesAlignObj returns the alignment of a ctypes type or instance.
// For simple types, alignment == sizeof. For structures, max of field
// alignments. For now, equal to sizeof.
func ctypesAlignObj(obj object.Object) int64 {
	sz := ctypesSizeofObj(obj)
	if sz < 0 {
		return sz
	}
	// For structures: find max field size as alignment
	var cls *object.Class
	switch x := obj.(type) {
	case *object.Class:
		cls = x
	case *object.Instance:
		cls = x.Class
	}
	if cls != nil {
		if _, ok := cls.Dict.GetStr("_fields_"); ok {
			max := int64(1)
			for _, ft := range ctypesFieldsAndTypes(cls) {
				fsz := ctypesSizeofClass(ft)
				if fsz > max {
					max = fsz
				}
			}
			return max
		}
	}
	return sz
}

// ctypesMakeStringBuffer creates a mutable string buffer instance.
func ctypesMakeStringBuffer(data []byte) *object.Instance {
	size := len(data)
	cls := &object.Class{Name: "c_char_Array_" + fmt.Sprintf("%d", size), Dict: object.NewDict()}
	cls.Dict.SetStr("_size_", object.NewInt(int64(size)))
	// __len__ on the class so len(inst) works
	capturedSize := size
	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(capturedSize)), nil
	}})

	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	// .raw = full bytes
	inst.Dict.SetStr("raw", &object.Bytes{V: append([]byte(nil), data...)})

	// .value = bytes up to first null byte
	valData := data
	for k, b := range data {
		if b == 0 {
			valData = data[:k]
			break
		}
	}
	inst.Dict.SetStr("value", &object.Bytes{V: append([]byte(nil), valData...)})

	// .size property (as int for simplicity)
	inst.Dict.SetStr("size", object.NewInt(int64(size)))

	// __len__ on the instance dict so len(inst) sees it directly
	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(capturedSize)), nil
	}})

	return inst
}

// ctypesMakeUnicodeBuffer creates a mutable unicode buffer instance.
func ctypesMakeUnicodeBuffer(size int) *object.Instance {
	cls := &object.Class{Name: "c_wchar_Array_" + fmt.Sprintf("%d", size), Dict: object.NewDict()}
	cls.Dict.SetStr("_size_", object.NewInt(int64(size*2)))
	capturedSize := size
	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(capturedSize)), nil
	}})

	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("value", &object.Str{V: ""})
	inst.Dict.SetStr("size", object.NewInt(int64(size)))

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(capturedSize)), nil
	}})

	return inst
}
