package vm

import (
	"bufio"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// mimetypesDB holds the global MIME database.
type mimetypesDB struct {
	mu           sync.RWMutex
	inited       bool
	strictExt    map[string]string   // ext (with dot) → MIME type
	nonStrict    map[string]string   // non-strict ext → MIME type
	strictInv    map[string][]string // MIME type → []ext (strict)
	nonStrictInv map[string][]string // MIME type → []ext (non-strict)
	suffixMap    map[string]string   // e.g. ".tgz" → ".tar.gz"
	encodingMap  map[string]string   // e.g. ".gz" → "gzip"
}

var globalMimetypesDB = &mimetypesDB{}

// builtinExtToType is the default strict extension→type mapping.
var builtinExtToType = map[string]string{
	".html":  "text/html",
	".htm":   "text/html",
	".txt":   "text/plain",
	".css":   "text/css",
	".csv":   "text/csv",
	".xml":   "application/xml",
	".json":  "application/json",
	".js":    "application/javascript",
	".mjs":   "text/javascript",
	".pdf":   "application/pdf",
	".zip":   "application/zip",
	".gz":    "application/gzip",
	".bz2":   "application/x-bzip2",
	".tar":   "application/x-tar",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".ico":   "image/x-icon",
	".mp3":   "audio/mpeg",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".ogg":   "application/ogg",
	".wav":   "audio/wav",
	".flac":  "audio/flac",
	".ttf":   "font/ttf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".py":    "text/x-python",
	".c":     "text/x-csrc",
	".h":     "text/x-chdr",
	".go":    "text/x-go",
	".md":    "text/markdown",
	".yaml":  "application/x-yaml",
	".yml":   "application/x-yaml",
	".toml":  "application/toml",
	".wasm":  "application/wasm",
	".bin":   "application/octet-stream",
	".exe":   "application/octet-stream",
	".dll":   "application/octet-stream",
	".sh":    "application/x-sh",
	".bat":   "text/x-batch",
	".rtf":   "application/rtf",
	".doc":   "application/msword",
	".docx":  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":   "application/vnd.ms-excel",
	".xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":   "application/vnd.ms-powerpoint",
	".pptx":  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".epub":  "application/epub+zip",
	".ics":   "text/calendar",
	".sql":   "application/sql",
	".apk":   "application/vnd.android.package-archive",
	".aar":   "application/x-android-archive",
	".dmg":   "application/x-apple-diskimage",
	".iso":   "application/x-iso9660-image",
}

// builtinSuffixMap is the default suffix_map.
var builtinSuffixMap = map[string]string{
	".tgz":  ".tar.gz",
	".taz":  ".tar.Z",
	".tbz2": ".tar.bz2",
	".txz":  ".tar.xz",
	".tlz":  ".tar.lz",
}

// builtinEncodingMap is the default encodings_map.
var builtinEncodingMap = map[string]string{
	".gz":  "gzip",
	".Z":   "compress",
	".bz2": "bzip2",
	".xz":  "xz",
	".br":  "br",
	".zst": "zst",
}

// initDB resets the global DB to built-in defaults.
func initDB(db *mimetypesDB) {
	db.strictExt = make(map[string]string, len(builtinExtToType))
	db.nonStrict = make(map[string]string)
	db.strictInv = make(map[string][]string)
	db.nonStrictInv = make(map[string][]string)
	db.suffixMap = make(map[string]string, len(builtinSuffixMap))
	db.encodingMap = make(map[string]string, len(builtinEncodingMap))

	for ext, typ := range builtinExtToType {
		db.strictExt[ext] = typ
		db.strictInv[typ] = append(db.strictInv[typ], ext)
	}
	for k, v := range builtinSuffixMap {
		db.suffixMap[k] = v
	}
	for k, v := range builtinEncodingMap {
		db.encodingMap[k] = v
	}
	// Sort inverse lists for determinism.
	for typ := range db.strictInv {
		sort.Strings(db.strictInv[typ])
	}
	db.inited = true
}

// ensureInit initialises the global DB on first use.
func ensureInit(db *mimetypesDB) {
	db.mu.RLock()
	ok := db.inited
	db.mu.RUnlock()
	if ok {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if !db.inited {
		initDB(db)
	}
}

// addTypeToDB registers a MIME type↔extension mapping in db.
func addTypeToDB(db *mimetypesDB, typ, ext string, strict bool) {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if strict {
		db.strictExt[ext] = typ
		if !sliceContains(db.strictInv[typ], ext) {
			db.strictInv[typ] = append(db.strictInv[typ], ext)
			sort.Strings(db.strictInv[typ])
		}
	} else {
		db.nonStrict[ext] = typ
		if !sliceContains(db.nonStrictInv[typ], ext) {
			db.nonStrictInv[typ] = append(db.nonStrictInv[typ], ext)
			sort.Strings(db.nonStrictInv[typ])
		}
	}
}

func sliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// guessTypeFromDB implements guess_type logic using the given maps.
func guessTypeFromDB(
	url string,
	strict bool,
	suffixMap, encodingMap, strictExt, nonStrict map[string]string,
) (typ, encoding string) {
	// Strip query and fragment.
	if idx := strings.IndexAny(url, "?#"); idx >= 0 {
		url = url[:idx]
	}

	ext := strings.ToLower(path.Ext(url))
	if ext == "" {
		return "", ""
	}

	// Check suffix_map first (e.g. .tgz → .tar.gz).
	if mapped, ok := suffixMap[ext]; ok {
		// mapped is something like ".tar.gz"; the encoding ext is the last part.
		encExt := strings.ToLower(path.Ext(mapped))
		if enc, ok2 := encodingMap[encExt]; ok2 {
			encoding = enc
		}
		// Re-derive extension from the base (before the encoding ext).
		base := mapped[:len(mapped)-len(encExt)]
		ext = strings.ToLower(path.Ext(base))
	} else if enc, ok := encodingMap[ext]; ok {
		// Direct encoding extension (e.g. ".gz").
		encoding = enc
		// Strip the encoding ext and re-derive the base ext.
		withoutEnc := url[:len(url)-len(ext)]
		ext = strings.ToLower(path.Ext(withoutEnc))
	}

	if ext == "" {
		return "", encoding
	}

	if t, ok := strictExt[ext]; ok {
		return t, encoding
	}
	if !strict {
		if t, ok := nonStrict[ext]; ok {
			return t, encoding
		}
	}
	return "", encoding
}

// guessAllExtensionsFromDB returns sorted extensions for a MIME type.
func guessAllExtensionsFromDB(typ string, strict bool, strictInv, nonStrictInv map[string][]string) []string {
	var result []string
	if exts, ok := strictInv[typ]; ok {
		result = append(result, exts...)
	}
	if !strict {
		if exts, ok := nonStrictInv[typ]; ok {
			for _, e := range exts {
				if !sliceContains(result, e) {
					result = append(result, e)
				}
			}
		}
	}
	sort.Strings(result)
	return result
}

// parseMimeTypesFile parses a mime.types-style file and returns a map of
// ext → type. Returns nil if the file cannot be read.
func parseMimeTypesFile(filename string) map[string]string {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()
	result := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		typ := fields[0]
		for _, ext := range fields[1:] {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			result[ext] = typ
		}
	}
	return result
}

// dictFromStringMap converts a Go map[string]string to a Python *object.Dict.
func dictFromStringMap(m map[string]string) *object.Dict {
	d := object.NewDict()
	for k, v := range m {
		d.SetStr(k, &object.Str{V: v})
	}
	return d
}

// ── buildMimetypes ────────────────────────────────────────────────────────────

func (i *Interp) buildMimetypes() *object.Module {
	m := &object.Module{Name: "mimetypes", Dict: object.NewDict()}
	db := globalMimetypesDB

	ensureInit(db)

	// ── inited ────────────────────────────────────────────────────────────────
	// We use a live accessor so changes via init() are reflected.
	m.Dict.SetStr("inited", object.BoolOf(db.inited))

	// ── knownfiles ───────────────────────────────────────────────────────────
	knownfiles := &object.List{V: []object.Object{}}
	m.Dict.SetStr("knownfiles", knownfiles)

	// ── suffix_map ───────────────────────────────────────────────────────────
	db.mu.RLock()
	suffixMapDict := dictFromStringMap(db.suffixMap)
	encodingsMapDict := dictFromStringMap(db.encodingMap)
	db.mu.RUnlock()
	m.Dict.SetStr("suffix_map", suffixMapDict)
	m.Dict.SetStr("encodings_map", encodingsMapDict)

	// ── types_map: (strict_dict, non_strict_dict) ────────────────────────────
	db.mu.RLock()
	strictExtDict := dictFromStringMap(db.strictExt)
	nonStrictDict := dictFromStringMap(db.nonStrict)
	db.mu.RUnlock()
	typesMapTuple := &object.Tuple{V: []object.Object{strictExtDict, nonStrictDict}}
	m.Dict.SetStr("types_map", typesMapTuple)

	// ── common_types: (strict_inv, non_strict_inv) ───────────────────────────
	db.mu.RLock()
	strictInvDict := object.NewDict()
	for typ, exts := range db.strictInv {
		lst := make([]object.Object, len(exts))
		for j, e := range exts {
			lst[j] = &object.Str{V: e}
		}
		strictInvDict.SetStr(typ, &object.List{V: lst})
	}
	nonStrictInvDict := object.NewDict()
	for typ, exts := range db.nonStrictInv {
		lst := make([]object.Object, len(exts))
		for j, e := range exts {
			lst[j] = &object.Str{V: e}
		}
		nonStrictInvDict.SetStr(typ, &object.List{V: lst})
	}
	db.mu.RUnlock()
	commonTypesTuple := &object.Tuple{V: []object.Object{strictInvDict, nonStrictInvDict}}
	m.Dict.SetStr("common_types", commonTypesTuple)

	// ── guess_type ────────────────────────────────────────────────────────────
	m.Dict.SetStr("guess_type", &object.BuiltinFunc{Name: "guess_type",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_type() requires url argument")
			}
			urlStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_type() url must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("strict"); ok2 {
					if b, ok3 := v.(*object.Bool); ok3 {
						strict = b.V
					}
				}
			}
			db.mu.RLock()
			t, enc := guessTypeFromDB(urlStr.V, strict, db.suffixMap, db.encodingMap, db.strictExt, db.nonStrict)
			db.mu.RUnlock()
			var typeObj object.Object = object.None
			var encObj object.Object = object.None
			if t != "" {
				typeObj = &object.Str{V: t}
			}
			if enc != "" {
				encObj = &object.Str{V: enc}
			}
			return &object.Tuple{V: []object.Object{typeObj, encObj}}, nil
		},
	})

	// ── guess_all_extensions ──────────────────────────────────────────────────
	m.Dict.SetStr("guess_all_extensions", &object.BuiltinFunc{Name: "guess_all_extensions",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_all_extensions() requires type argument")
			}
			typStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_all_extensions() type must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("strict"); ok2 {
					if b, ok3 := v.(*object.Bool); ok3 {
						strict = b.V
					}
				}
			}
			db.mu.RLock()
			exts := guessAllExtensionsFromDB(typStr.V, strict, db.strictInv, db.nonStrictInv)
			db.mu.RUnlock()
			lst := make([]object.Object, len(exts))
			for j, e := range exts {
				lst[j] = &object.Str{V: e}
			}
			return &object.List{V: lst}, nil
		},
	})

	// ── guess_extension ───────────────────────────────────────────────────────
	m.Dict.SetStr("guess_extension", &object.BuiltinFunc{Name: "guess_extension",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_extension() requires type argument")
			}
			typStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_extension() type must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("strict"); ok2 {
					if b, ok3 := v.(*object.Bool); ok3 {
						strict = b.V
					}
				}
			}
			db.mu.RLock()
			exts := guessAllExtensionsFromDB(typStr.V, strict, db.strictInv, db.nonStrictInv)
			db.mu.RUnlock()
			if len(exts) == 0 {
				return object.None, nil
			}
			return &object.Str{V: exts[0]}, nil
		},
	})

	// ── add_type ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("add_type", &object.BuiltinFunc{Name: "add_type",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(args) < 2 {
				return nil, object.Errorf(i.typeErr, "add_type() requires type and ext")
			}
			typStr, ok1 := args[0].(*object.Str)
			extStr, ok2 := args[1].(*object.Str)
			if !ok1 || !ok2 {
				return nil, object.Errorf(i.typeErr, "add_type() type and ext must be str")
			}
			strict := true
			if len(args) >= 3 {
				if b, ok3 := args[2].(*object.Bool); ok3 {
					strict = b.V
				}
			}
			if kwargs != nil {
				if v, ok3 := kwargs.GetStr("strict"); ok3 {
					if b, ok4 := v.(*object.Bool); ok4 {
						strict = b.V
					}
				}
			}
			db.mu.Lock()
			addTypeToDB(db, typStr.V, extStr.V, strict)
			db.mu.Unlock()
			// Update the module-level suffix_map/encodings_map dicts so attribute
			// access reflects the new state.
			return object.None, nil
		},
	})

	// ── init ──────────────────────────────────────────────────────────────────
	m.Dict.SetStr("init", &object.BuiltinFunc{Name: "init",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			var files []string
			if len(args) >= 1 && args[0] != object.None {
				switch v := args[0].(type) {
				case *object.List:
					for _, item := range v.V {
						if s, ok2 := item.(*object.Str); ok2 {
							files = append(files, s.V)
						}
					}
				case *object.Tuple:
					for _, item := range v.V {
						if s, ok2 := item.(*object.Str); ok2 {
							files = append(files, s.V)
						}
					}
				}
			}
			db.mu.Lock()
			initDB(db)
			for _, fn := range files {
				result := parseMimeTypesFile(fn)
				if result != nil {
					for ext, typ := range result {
						addTypeToDB(db, typ, ext, true)
					}
				}
			}
			db.mu.Unlock()
			// Refresh module attribute.
			m.Dict.SetStr("inited", object.True)
			return object.None, nil
		},
	})

	// ── read_mime_types ───────────────────────────────────────────────────────
	m.Dict.SetStr("read_mime_types", &object.BuiltinFunc{Name: "read_mime_types",
		Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "read_mime_types() requires filename")
			}
			fnStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "read_mime_types() filename must be str")
			}
			result := parseMimeTypesFile(fnStr.V)
			if result == nil {
				return object.None, nil
			}
			d := object.NewDict()
			for ext, typ := range result {
				d.SetStr(ext, &object.Str{V: typ})
			}
			return d, nil
		},
	})

	// ── MimeTypes class ───────────────────────────────────────────────────────
	mimeTypesCls := &object.Class{Name: "MimeTypes", Dict: object.NewDict()}

	mimeTypesCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(args) == 0 {
				return nil, object.Errorf(i.typeErr, "MimeTypes.__init__() missing self")
			}
			self, ok := args[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "MimeTypes.__init__() self must be instance")
			}
			// Per-instance dicts mirror the global defaults.
			db.mu.RLock()
			instStrict := make(map[string]string, len(db.strictExt))
			for k, v := range db.strictExt {
				instStrict[k] = v
			}
			instNonStrict := make(map[string]string, len(db.nonStrict))
			for k, v := range db.nonStrict {
				instNonStrict[k] = v
			}
			instStrictInv := make(map[string][]string, len(db.strictInv))
			for k, v := range db.strictInv {
				cp := make([]string, len(v))
				copy(cp, v)
				instStrictInv[k] = cp
			}
			instNonStrictInv := make(map[string][]string, len(db.nonStrictInv))
			for k, v := range db.nonStrictInv {
				cp := make([]string, len(v))
				copy(cp, v)
				instNonStrictInv[k] = cp
			}
			instSuffixMap := make(map[string]string, len(db.suffixMap))
			for k, v := range db.suffixMap {
				instSuffixMap[k] = v
			}
			instEncodingMap := make(map[string]string, len(db.encodingMap))
			for k, v := range db.encodingMap {
				instEncodingMap[k] = v
			}
			db.mu.RUnlock()

			self.Dict.SetStr("suffix_map", dictFromStringMap(instSuffixMap))
			self.Dict.SetStr("encodings_map", dictFromStringMap(instEncodingMap))
			self.Dict.SetStr("types_map", &object.Tuple{V: []object.Object{
				dictFromStringMap(instStrict),
				dictFromStringMap(instNonStrict),
			}})

			// Attach instance-level methods that close over the per-instance maps.
			i.attachMimeTypesInstanceMethods(self, instStrict, instNonStrict, instStrictInv, instNonStrictInv, instSuffixMap, instEncodingMap)

			// Read any files passed to the constructor.
			strict := true
			if len(args) >= 2 {
				switch v := args[1].(type) {
				case *object.List:
					for _, item := range v.V {
						if s, ok2 := item.(*object.Str); ok2 {
							result := parseMimeTypesFile(s.V)
							if result != nil {
								for ext, typ := range result {
									addTypeToDB(&mimetypesDB{
										strictExt: instStrict, nonStrict: instNonStrict,
										strictInv: instStrictInv, nonStrictInv: instNonStrictInv,
									}, typ, ext, strict)
								}
							}
						}
					}
				}
			}
			_ = strict
			return object.None, nil
		},
	})

	m.Dict.SetStr("MimeTypes", mimeTypesCls)

	return m
}

// attachMimeTypesInstanceMethods wires instance methods onto a MimeTypes instance.
func (i *Interp) attachMimeTypesInstanceMethods(
	self *object.Instance,
	strictExt, nonStrict map[string]string,
	strictInv, nonStrictInv map[string][]string,
	suffixMap, encodingMap map[string]string,
) {
	// Helper to make an instance DB for add operations.
	instDB := &mimetypesDB{
		strictExt:    strictExt,
		nonStrict:    nonStrict,
		strictInv:    strictInv,
		nonStrictInv: nonStrictInv,
		suffixMap:    suffixMap,
		encodingMap:  encodingMap,
	}

	self.Dict.SetStr("guess_type", &object.BuiltinFunc{Name: "guess_type",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			args = mpArgs(args)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_type() requires url")
			}
			urlStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_type() url must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			t, enc := guessTypeFromDB(urlStr.V, strict, suffixMap, encodingMap, strictExt, nonStrict)
			var typeObj object.Object = object.None
			var encObj object.Object = object.None
			if t != "" {
				typeObj = &object.Str{V: t}
			}
			if enc != "" {
				encObj = &object.Str{V: enc}
			}
			return &object.Tuple{V: []object.Object{typeObj, encObj}}, nil
		},
	})

	self.Dict.SetStr("guess_all_extensions", &object.BuiltinFunc{Name: "guess_all_extensions",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			args = mpArgs(args)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_all_extensions() requires type")
			}
			typStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_all_extensions() type must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			exts := guessAllExtensionsFromDB(typStr.V, strict, strictInv, nonStrictInv)
			lst := make([]object.Object, len(exts))
			for j, e := range exts {
				lst[j] = &object.Str{V: e}
			}
			return &object.List{V: lst}, nil
		},
	})

	self.Dict.SetStr("guess_extension", &object.BuiltinFunc{Name: "guess_extension",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			args = mpArgs(args)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "guess_extension() requires type")
			}
			typStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "guess_extension() type must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			exts := guessAllExtensionsFromDB(typStr.V, strict, strictInv, nonStrictInv)
			if len(exts) == 0 {
				return object.None, nil
			}
			return &object.Str{V: exts[0]}, nil
		},
	})

	self.Dict.SetStr("add", &object.BuiltinFunc{Name: "add",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			args = mpArgs(args)
			if len(args) < 2 {
				return nil, object.Errorf(i.typeErr, "add() requires type and ext")
			}
			typStr, ok1 := args[0].(*object.Str)
			extStr, ok2 := args[1].(*object.Str)
			if !ok1 || !ok2 {
				return nil, object.Errorf(i.typeErr, "add() type and ext must be str")
			}
			strict := true
			if len(args) >= 3 {
				if b, ok3 := args[2].(*object.Bool); ok3 {
					strict = b.V
				}
			}
			addTypeToDB(instDB, typStr.V, extStr.V, strict)
			return object.None, nil
		},
	})

	self.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
			args = mpArgs(args)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "read() requires filename")
			}
			fnStr, ok := args[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "read() filename must be str")
			}
			strict := true
			if len(args) >= 2 {
				if b, ok2 := args[1].(*object.Bool); ok2 {
					strict = b.V
				}
			}
			result := parseMimeTypesFile(fnStr.V)
			if result != nil {
				for ext, typ := range result {
					addTypeToDB(instDB, typ, ext, strict)
				}
			}
			return object.None, nil
		},
	})

	self.Dict.SetStr("readfp", &object.BuiltinFunc{Name: "readfp",
		Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			// readfp reads from a file-like object; we only support objects
			// that have a read() method returning a string.
			args = mpArgs(args)
			return object.None, nil
		},
	})

	self.Dict.SetStr("read_windows_registry", &object.BuiltinFunc{Name: "read_windows_registry",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
}
