package vm

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/tamnd/goipy/object"
)

// Python 3.14 magic number for .pyc files.
const py314MagicNumber = "\x2b\x0e\x0d\x0a"

// pythonVersion is the version tag used in __pycache__ paths.
const pythonVersion = "cpython-314"

// buildModuleSpecClass returns a shared ModuleSpec class used by both
// importlib.machinery and importlib.util.
func buildModuleSpecClass() *object.Class {
	cls := &object.Class{Name: "ModuleSpec", Dict: object.NewDict()}

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "ModuleSpec()"}, nil
			}
			self := a[0].(*object.Instance)
			name := specStrAttr(self, "name")
			loader := "None"
			if v, ok := self.Dict.GetStr("loader"); ok && v != object.None {
				loader = object.Repr(v)
			}
			origin := specStrAttr(self, "origin")
			s := fmt.Sprintf("ModuleSpec(name=%q, loader=%s", name, loader)
			if origin != "" {
				s += fmt.Sprintf(", origin=%q", origin)
			}
			s += ")"
			return &object.Str{V: s}, nil
		}})

	return cls
}

func specStrAttr(inst *object.Instance, key string) string {
	if v, ok := inst.Dict.GetStr(key); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			return s.V
		}
	}
	return ""
}

// makeModuleSpec creates a ModuleSpec instance.
func makeModuleSpec(cls *object.Class, name string, loader object.Object, origin string, isPackage bool) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("loader", loader)
	if origin == "" {
		inst.Dict.SetStr("origin", object.None)
		inst.Dict.SetStr("has_location", object.False)
	} else {
		inst.Dict.SetStr("origin", &object.Str{V: origin})
		inst.Dict.SetStr("has_location", object.True)
	}
	if isPackage {
		inst.Dict.SetStr("submodule_search_locations", &object.List{V: nil})
	} else {
		inst.Dict.SetStr("submodule_search_locations", object.None)
	}
	// parent: everything before the last dot, or empty string
	parent := ""
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		parent = name[:dot]
	}
	inst.Dict.SetStr("parent", &object.Str{V: parent})
	// cached: computed from origin when has_location
	if origin != "" {
		cached := cacheFromSource(origin, "")
		inst.Dict.SetStr("cached", &object.Str{V: cached})
	} else {
		inst.Dict.SetStr("cached", object.None)
	}
	return inst
}

// cacheFromSource mirrors importlib.util.cache_from_source logic.
func cacheFromSource(path, optimization string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	var tag string
	if optimization == "" {
		tag = pythonVersion + ".pyc"
	} else {
		tag = pythonVersion + ".opt-" + optimization + ".pyc"
	}
	return filepath.Join(dir, "__pycache__", stem+"."+tag)
}

// fnv1a64 is a simple deterministic 8-byte hash for source_hash.
func fnv1a64(data []byte) uint64 {
	h := uint64(14695981039346656037)
	for _, b := range data {
		h ^= uint64(b)
		h = bits.RotateLeft64(h, 13) * 1099511628211
	}
	return h
}

// ── importlib.util ────────────────────────────────────────────────────────────

func (i *Interp) buildImportlibUtil() *object.Module {
	m := &object.Module{Name: "importlib.util", Dict: object.NewDict()}

	specCls := buildModuleSpecClass()

	// MAGIC_NUMBER
	m.Dict.SetStr("MAGIC_NUMBER", &object.Bytes{V: []byte(py314MagicNumber)})

	// cache_from_source(path, debug_override=None, *, optimization=None)
	m.Dict.SetStr("cache_from_source", &object.BuiltinFunc{Name: "cache_from_source",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "cache_from_source() requires path")
			}
			path := ""
			if s, ok := a[0].(*object.Str); ok {
				path = s.V
			}
			opt := ""
			if kw != nil {
				if v, ok := kw.GetStr("optimization"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						opt = s.V
					} else if n, ok2 := toInt64(v); ok2 {
						opt = fmt.Sprintf("%d", n)
					}
				}
			}
			return &object.Str{V: cacheFromSource(path, opt)}, nil
		}})

	// source_from_cache(path)
	m.Dict.SetStr("source_from_cache", &object.BuiltinFunc{Name: "source_from_cache",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "source_from_cache() requires path")
			}
			path := ""
			if s, ok := a[0].(*object.Str); ok {
				path = s.V
			}
			// Must contain __pycache__ as the direct parent directory.
			dir := filepath.Dir(path)
			if filepath.Base(dir) != "__pycache__" {
				return nil, object.Errorf(i.valueErr, "__pycache__ not bottom-level directory in %q", path)
			}
			// Strip cpython-NNN and optional .opt-N tag from stem.
			base := filepath.Base(path)
			stem := strings.TrimSuffix(base, ".pyc")
			// Remove .cpython-NNN[.opt-N] suffix.
			if dot := strings.LastIndex(stem, "."); dot >= 0 {
				stem = stem[:dot]
			}
			srcDir := filepath.Dir(dir)
			return &object.Str{V: filepath.Join(srcDir, stem+".py")}, nil
		}})

	// resolve_name(name, package)
	m.Dict.SetStr("resolve_name", &object.BuiltinFunc{Name: "resolve_name",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "resolve_name() requires name and package")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			pkg := ""
			if a[1] != object.None {
				if s, ok := a[1].(*object.Str); ok {
					pkg = s.V
				}
			}
			// Count leading dots (relative level).
			level := 0
			for level < len(name) && name[level] == '.' {
				level++
			}
			if level == 0 {
				return &object.Str{V: name}, nil
			}
			if pkg == "" {
				return nil, object.Errorf(i.importErr, "the 'package' argument is required to perform a relative import for %q", name)
			}
			base := name[level:]
			// Walk up pkg by (level-1) components.
			parts := strings.Split(pkg, ".")
			if level-1 > len(parts) {
				return nil, object.Errorf(i.importErr, "attempted relative import beyond top-level package")
			}
			parts = parts[:len(parts)-(level-1)]
			anchor := strings.Join(parts, ".")
			if base == "" {
				return &object.Str{V: anchor}, nil
			}
			if anchor == "" {
				return &object.Str{V: base}, nil
			}
			return &object.Str{V: anchor + "." + base}, nil
		}})

	// find_spec(name, package=None)
	m.Dict.SetStr("find_spec", &object.BuiltinFunc{Name: "find_spec",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "find_spec() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			mod, err := i.loadModule(name)
			if err != nil || mod == nil {
				return object.None, nil
			}
			return makeModuleSpec(specCls, name, object.None, mod.Path, false), nil
		}})

	// source_hash(source_bytes) → 8 bytes
	m.Dict.SetStr("source_hash", &object.BuiltinFunc{Name: "source_hash",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "source_hash() requires source_bytes")
			}
			var data []byte
			switch v := a[0].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			}
			h := fnv1a64(data)
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, h)
			return &object.Bytes{V: buf}, nil
		}})

	// decode_source(source_bytes) → str
	m.Dict.SetStr("decode_source", &object.BuiltinFunc{Name: "decode_source",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			var data []byte
			if b, ok := a[0].(*object.Bytes); ok {
				data = b.V
			} else if s, ok := a[0].(*object.Str); ok {
				return s, nil
			}
			// Strip UTF-8 BOM if present.
			if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
				data = data[3:]
			}
			// Ensure valid UTF-8.
			if !utf8.Valid(data) {
				return nil, object.Errorf(i.valueErr, "source is not valid UTF-8")
			}
			return &object.Str{V: string(data)}, nil
		}})

	// spec_from_file_location(name, location=None, *, loader=None, submodule_search_locations=...)
	m.Dict.SetStr("spec_from_file_location", &object.BuiltinFunc{Name: "spec_from_file_location",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "spec_from_file_location() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			origin := ""
			if len(a) >= 2 && a[1] != object.None {
				if s, ok := a[1].(*object.Str); ok {
					origin = s.V
				}
			}
			var loader object.Object = object.None
			isPackage := false
			if kw != nil {
				if v, ok := kw.GetStr("loader"); ok && v != object.None {
					loader = v
				}
				if v, ok := kw.GetStr("submodule_search_locations"); ok && v != object.None {
					isPackage = true
					_ = v
				}
			}
			// If origin looks like __init__.py, treat as package.
			if strings.HasSuffix(origin, "__init__.py") {
				isPackage = true
			}
			inst := makeModuleSpec(specCls, name, loader, origin, isPackage)
			return inst, nil
		}})

	// spec_from_loader(name, loader, *, origin=None, is_package=None)
	m.Dict.SetStr("spec_from_loader", &object.BuiltinFunc{Name: "spec_from_loader",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "spec_from_loader() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			var loader object.Object = object.None
			if len(a) >= 2 {
				loader = a[1]
			}
			origin := ""
			isPackage := false
			if kw != nil {
				if v, ok := kw.GetStr("origin"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						origin = s.V
					}
				}
				if v, ok := kw.GetStr("is_package"); ok && v != object.None {
					isPackage = v == object.True
				}
			}
			return makeModuleSpec(specCls, name, loader, origin, isPackage), nil
		}})

	// module_from_spec(spec) → module
	// NOTE: do NOT call mpArgs here — spec is *object.Instance and mpArgs
	// would incorrectly strip it thinking it is a method self argument.
	m.Dict.SetStr("module_from_spec", &object.BuiltinFunc{Name: "module_from_spec",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "module_from_spec() requires spec")
			}
			spec, ok := a[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "module_from_spec() argument must be a ModuleSpec")
			}
			name := specStrAttr(spec, "name")
			mod := &object.Module{Name: name, Dict: object.NewDict()}
			mod.Dict.SetStr("__name__", &object.Str{V: name})
			mod.Dict.SetStr("__spec__", spec)
			if v, ok2 := spec.Dict.GetStr("loader"); ok2 {
				mod.Dict.SetStr("__loader__", v)
			} else {
				mod.Dict.SetStr("__loader__", object.None)
			}
			if v, ok2 := spec.Dict.GetStr("parent"); ok2 {
				mod.Dict.SetStr("__package__", v)
			}
			if v, ok2 := spec.Dict.GetStr("origin"); ok2 && v != object.None {
				mod.Dict.SetStr("__file__", v)
			}
			return mod, nil
		}})

	// Loader abstract class
	loaderCls := &object.Class{Name: "Loader", Dict: object.NewDict()}
	m.Dict.SetStr("Loader", loaderCls)

	// LazyLoader(loader)
	lazyLoaderCls := &object.Class{Name: "LazyLoader", Dict: object.NewDict()}
	lazyLoaderCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				self := a[0].(*object.Instance)
				self.Dict.SetStr("loader", a[1])
			}
			return object.None, nil
		}})
	m.Dict.SetStr("LazyLoader", &object.BuiltinFunc{Name: "LazyLoader",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: lazyLoaderCls, Dict: object.NewDict()}
			if len(a) > 0 {
				inst.Dict.SetStr("loader", a[0])
			}
			return inst, nil
		}})

	return m
}

// ── importlib.machinery ───────────────────────────────────────────────────────

func (i *Interp) buildImportlibMachinery() *object.Module {
	m := &object.Module{Name: "importlib.machinery", Dict: object.NewDict()}

	specCls := buildModuleSpecClass()

	// ModuleSpec constructor
	m.Dict.SetStr("ModuleSpec", &object.BuiltinFunc{Name: "ModuleSpec",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "ModuleSpec() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			var loader object.Object = object.None
			if len(a) >= 2 {
				loader = a[1]
			}
			origin := ""
			isPackage := false
			if kw != nil {
				if v, ok := kw.GetStr("origin"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						origin = s.V
					}
				}
				if v, ok := kw.GetStr("is_package"); ok && v != object.None {
					isPackage = v == object.True
				}
			}
			return makeModuleSpec(specCls, name, loader, origin, isPackage), nil
		}})

	// Suffix constants
	m.Dict.SetStr("SOURCE_SUFFIXES", &object.List{V: []object.Object{&object.Str{V: ".py"}}})
	m.Dict.SetStr("BYTECODE_SUFFIXES", &object.List{V: []object.Object{&object.Str{V: ".pyc"}}})
	m.Dict.SetStr("EXTENSION_SUFFIXES", &object.List{V: []object.Object{&object.Str{V: ".so"}}})

	// all_suffixes()
	m.Dict.SetStr("all_suffixes", &object.BuiltinFunc{Name: "all_suffixes",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{
				&object.Str{V: ".py"},
				&object.Str{V: ".pyc"},
				&object.Str{V: ".so"},
			}}, nil
		}})

	// BuiltinImporter
	biCls := &object.Class{Name: "BuiltinImporter", Dict: object.NewDict()}
	biCls.Dict.SetStr("find_spec", &object.BuiltinFunc{Name: "find_spec",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	biCls.Dict.SetStr("find_module", &object.BuiltinFunc{Name: "find_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	biCls.Dict.SetStr("load_module", &object.BuiltinFunc{Name: "load_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("BuiltinImporter", biCls)

	// FrozenImporter
	fiCls := &object.Class{Name: "FrozenImporter", Dict: object.NewDict()}
	fiCls.Dict.SetStr("find_spec", &object.BuiltinFunc{Name: "find_spec",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("FrozenImporter", fiCls)

	// PathFinder: returns a ModuleSpec for known modules, None otherwise
	pfCls := &object.Class{Name: "PathFinder", Dict: object.NewDict()}
	pfCls.Dict.SetStr("find_spec", &object.BuiltinFunc{Name: "find_spec",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return object.None, nil
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			mod, err := i.loadModule(name)
			if err != nil || mod == nil {
				return object.None, nil
			}
			return makeModuleSpec(specCls, name, object.None, mod.Path, false), nil
		}})
	pfCls.Dict.SetStr("invalidate_caches", &object.BuiltinFunc{Name: "invalidate_caches",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("PathFinder", pfCls)

	// Stub loader classes
	for _, name := range []string{
		"FileFinder", "SourceFileLoader", "SourcelessFileLoader",
		"ExtensionFileLoader", "NamespaceLoader", "AppleFrameworkLoader",
		"WindowsRegistryFinder",
	} {
		n := name
		cls := &object.Class{Name: n, Dict: object.NewDict()}
		m.Dict.SetStr(n, cls)
	}

	return m
}

// ── importlib.abc ─────────────────────────────────────────────────────────────

func (i *Interp) buildImportlibAbc() *object.Module {
	m := &object.Module{Name: "importlib.abc", Dict: object.NewDict()}
	for _, name := range []string{
		"Loader", "MetaPathFinder", "PathEntryFinder",
		"InspectLoader", "ExecutionLoader", "FileLoader",
		"SourceLoader", "ResourceLoader",
	} {
		n := name
		cls := &object.Class{Name: n, Dict: object.NewDict()}
		m.Dict.SetStr(n, cls)
	}
	return m
}
