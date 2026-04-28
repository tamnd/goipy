package vm

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildZipimport() *object.Module {
	m := &object.Module{Name: "zipimport", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("path_sep", &object.Str{V: "/"})
	m.Dict.SetStr("alt_path_sep", &object.Str{V: ""})
	m.Dict.SetStr("END_CENTRAL_DIR_SIZE", object.NewInt(22))
	m.Dict.SetStr("END_CENTRAL_DIR_SIZE_64", object.NewInt(56))
	m.Dict.SetStr("END_CENTRAL_DIR_LOCATOR_SIZE_64", object.NewInt(20))
	m.Dict.SetStr("MAX_COMMENT_LEN", object.NewInt(65535))
	m.Dict.SetStr("MAX_UINT32", object.NewInt(4294967295))
	m.Dict.SetStr("ZIP64_EXTRA_TAG", object.NewInt(1))
	m.Dict.SetStr("STRING_END_ARCHIVE", &object.Bytes{V: []byte("PK\x05\x06")})
	m.Dict.SetStr("STRING_END_LOCATOR_64", &object.Bytes{V: []byte("PK\x06\x07")})
	m.Dict.SetStr("STRING_END_ZIP_64", &object.Bytes{V: []byte("PK\x06\x06")})
	m.Dict.SetStr("cp437_table", &object.Str{V: cp437Table})

	// ── ZipImportError ────────────────────────────────────────────────────────

	zipErrCls := &object.Class{
		Name:  "ZipImportError",
		Bases: []*object.Class{i.importErr},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("ZipImportError", zipErrCls)

	// ── zipimporter class ─────────────────────────────────────────────────────

	ziClass := &object.Class{Name: "zipimporter", Dict: object.NewDict()}

	// openZip opens the zip and returns reader + file set.
	openZip := func(archive string) (*zip.ReadCloser, map[string]struct{}, error) {
		rc, err := zip.OpenReader(archive)
		if err != nil {
			return nil, nil, err
		}
		files := make(map[string]struct{}, len(rc.File))
		for _, f := range rc.File {
			files[f.Name] = struct{}{}
		}
		return rc, files, nil
	}

	// readEntry reads bytes from named entry within an open zip.
	readZipEntry := func(archive, name string) ([]byte, error) {
		rc, err := zip.OpenReader(archive)
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		for _, f := range rc.File {
			if f.Name == name {
				r, err2 := f.Open()
				if err2 != nil {
					return nil, err2
				}
				defer r.Close()
				return io.ReadAll(r)
			}
		}
		return nil, os.ErrNotExist
	}

	// findModule locates fullname within the zip (package or module).
	// Returns (entryPath string, isPackage bool, found bool).
	findInZip := func(files map[string]struct{}, prefix, fullname string) (string, bool, bool) {
		pkgPath := prefix + fullname + "/__init__.py"
		if _, ok := files[pkgPath]; ok {
			return pkgPath, true, true
		}
		modPath := prefix + fullname + ".py"
		if _, ok := files[modPath]; ok {
			return modPath, false, true
		}
		return "", false, false
	}

	ziClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			pathStr, ok := a[1].(*object.Str)
			if !ok {
				return nil, object.Errorf(zipErrCls, "path must be a string")
			}
			path := pathStr.V

			// Find the zip file by walking up the path segments.
			// e.g. "/foo/bar.zip/pkg/sub" → archive="/foo/bar.zip", prefix="pkg/sub/"
			archive := ""
			prefix := ""

			// Normalise separators for uniform scanning.
			normalised := filepath.ToSlash(path)
			parts := strings.Split(normalised, "/")
			for end := len(parts); end >= 1; end-- {
				candidate := filepath.FromSlash(strings.Join(parts[:end], "/"))
				info, err := os.Stat(candidate)
				if err == nil && !info.IsDir() {
					// Verify it's actually a zip.
					rc, err2 := zip.OpenReader(candidate)
					if err2 == nil {
						rc.Close()
						archive = candidate
						rest := strings.Join(parts[end:], "/")
						if rest != "" {
							prefix = rest + "/"
						}
						break
					}
				}
			}
			if archive == "" {
				return nil, object.Errorf(zipErrCls, "not a Zip file: %s", path)
			}

			self.Dict.SetStr("archive", &object.Str{V: archive})
			self.Dict.SetStr("prefix", &object.Str{V: prefix})
			// Cache file list.
			rc, files, err := openZip(archive)
			if err != nil {
				return nil, object.Errorf(zipErrCls, "can't open Zip file: %s", archive)
			}
			rc.Close()
			fileDict := object.NewDict()
			for name := range files {
				fileDict.SetStr(name, object.None)
			}
			self.Dict.SetStr("_files", fileDict)
			return object.None, nil
		},
	})

	ziClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			archive := ""
			prefix := ""
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					if v, ok2 := inst.Dict.GetStr("archive"); ok2 {
						if s, ok3 := v.(*object.Str); ok3 {
							archive = s.V
						}
					}
					if v, ok2 := inst.Dict.GetStr("prefix"); ok2 {
						if s, ok3 := v.(*object.Str); ok3 {
							prefix = s.V
						}
					}
				}
			}
			path := archive
			if prefix != "" {
				path = archive + string(os.PathSeparator) + prefix
			}
			return &object.Str{V: "<zipimporter object \"" + path + "\">"}, nil
		},
	})

	// Helper to get archive/prefix/files from self.
	getZiState := func(self *object.Instance) (archive, prefix string, files map[string]struct{}) {
		if v, ok := self.Dict.GetStr("archive"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				archive = s.V
			}
		}
		if v, ok := self.Dict.GetStr("prefix"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				prefix = s.V
			}
		}
		if v, ok := self.Dict.GetStr("_files"); ok {
			if d, ok2 := v.(*object.Dict); ok2 {
				ks, _ := d.Items()
				files = make(map[string]struct{}, len(ks))
				for _, k := range ks {
					if s, ok3 := k.(*object.Str); ok3 {
						files[s.V] = struct{}{}
					}
				}
			}
		}
		return
	}

	ziClass.Dict.SetStr("find_spec", &object.BuiltinFunc{
		Name: "find_spec",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			archive, prefix, files := getZiState(self)
			entryPath, isPkg, found := findInZip(files, prefix, fullname.V)
			if !found {
				return object.None, nil
			}
			// Return a minimal ModuleSpec-like instance.
			specCls := &object.Class{Name: "ModuleSpec", Dict: object.NewDict()}
			spec := &object.Instance{Class: specCls, Dict: object.NewDict()}
			spec.Dict.SetStr("name", fullname)
			origin := archive + "/" + entryPath
			spec.Dict.SetStr("origin", &object.Str{V: origin})
			spec.Dict.SetStr("submodule_search_locations", object.None)
			if isPkg {
				searchLoc := &object.List{V: []object.Object{
					&object.Str{V: archive + "/" + prefix + fullname.V},
				}}
				spec.Dict.SetStr("submodule_search_locations", searchLoc)
			}
			return spec, nil
		},
	})

	ziClass.Dict.SetStr("is_package", &object.BuiltinFunc{
		Name: "is_package",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.False, nil
			}
			_, prefix, files := getZiState(self)
			_, isPkg, found := findInZip(files, prefix, fullname.V)
			if !found {
				return nil, object.Errorf(zipErrCls, "can't find module '%s'", fullname.V)
			}
			return object.BoolOf(isPkg), nil
		},
	})

	ziClass.Dict.SetStr("get_filename", &object.BuiltinFunc{
		Name: "get_filename",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			archive, prefix, files := getZiState(self)
			entryPath, _, found := findInZip(files, prefix, fullname.V)
			if !found {
				return nil, object.Errorf(zipErrCls, "can't find module '%s'", fullname.V)
			}
			return &object.Str{V: archive + "/" + entryPath}, nil
		},
	})

	ziClass.Dict.SetStr("get_source", &object.BuiltinFunc{
		Name: "get_source",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			archive, prefix, files := getZiState(self)
			entryPath, _, found := findInZip(files, prefix, fullname.V)
			if !found {
				return nil, object.Errorf(zipErrCls, "can't find module '%s'", fullname.V)
			}
			data, err := readZipEntry(archive, entryPath)
			if err != nil {
				return nil, object.Errorf(zipErrCls, "can't read source for '%s'", fullname.V)
			}
			return &object.Str{V: string(data)}, nil
		},
	})

	ziClass.Dict.SetStr("get_data", &object.BuiltinFunc{
		Name: "get_data",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.osErr, "no path given")
			}
			self := a[0].(*object.Instance)
			pathArg, ok := a[1].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.osErr, "path must be str")
			}
			archive, _, _ := getZiState(self)

			// Strip archive prefix to get the entry name.
			entry := filepath.ToSlash(pathArg.V)
			archiveSlash := filepath.ToSlash(archive) + "/"
			if strings.HasPrefix(entry, archiveSlash) {
				entry = entry[len(archiveSlash):]
			} else if strings.HasPrefix(entry, filepath.ToSlash(archive)) {
				entry = strings.TrimPrefix(entry, filepath.ToSlash(archive))
				entry = strings.TrimPrefix(entry, "/")
			}
			// Also handle OS-style path sep.
			entry = strings.ReplaceAll(entry, string(os.PathSeparator), "/")

			data, err := readZipEntry(archive, entry)
			if err != nil {
				return nil, object.Errorf(i.osErr, "can't read data for '%s'", pathArg.V)
			}
			return &object.Bytes{V: data}, nil
		},
	})

	ziClass.Dict.SetStr("get_code", &object.BuiltinFunc{
		Name: "get_code",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			_, prefix, files := getZiState(self)
			_, _, found := findInZip(files, prefix, fullname.V)
			if !found {
				return nil, object.Errorf(zipErrCls, "can't find module '%s'", fullname.V)
			}
			return &object.Code{Name: fullname.V}, nil
		},
	})

	ziClass.Dict.SetStr("get_resource_reader", &object.BuiltinFunc{
		Name: "get_resource_reader",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			fullname, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			_, prefix, files := getZiState(self)
			_, _, found := findInZip(files, prefix, fullname.V)
			if !found {
				return object.None, nil
			}
			// Return a minimal reader instance.
			readerCls := &object.Class{Name: "ZipReader", Dict: object.NewDict()}
			reader := &object.Instance{Class: readerCls, Dict: object.NewDict()}
			reader.Dict.SetStr("name", fullname)
			return reader, nil
		},
	})

	ziClass.Dict.SetStr("invalidate_caches", &object.BuiltinFunc{
		Name: "invalidate_caches",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	ziClass.Dict.SetStr("_get_files", &object.BuiltinFunc{
		Name: "_get_files",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewDict(), nil
			}
			self := a[0].(*object.Instance)
			archive, _, _ := getZiState(self)
			rc, err := zip.OpenReader(archive)
			if err != nil {
				return object.NewDict(), nil
			}
			defer rc.Close()
			result := object.NewDict()
			for _, f := range rc.File {
				// Value is a tuple of metadata: (compress_type, data_offset, file_size, ...)
				// goipy returns an empty tuple for simplicity.
				result.SetStr(f.Name, &object.Tuple{V: []object.Object{}})
			}
			return result, nil
		},
	})

	// Stubs for loader protocol methods.
	ziClass.Dict.SetStr("load_module", &object.BuiltinFunc{
		Name: "load_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	ziClass.Dict.SetStr("create_module", &object.BuiltinFunc{
		Name: "create_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	ziClass.Dict.SetStr("exec_module", &object.BuiltinFunc{
		Name: "exec_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("zipimporter", ziClass)

	return m
}

// cp437Table is the 256-character CP437 encoding table, matching Python's
// zipimport.cp437_table exactly.
const cp437Table = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	" !\"#$%&'()*+,-./" +
	"0123456789:;<=>?" +
	"@ABCDEFGHIJKLMNO" +
	"PQRSTUVWXYZ[\\]^_" +
	"`abcdefghijklmno" +
	"pqrstuvwxyz{|}~\x7f" +
	"ÇüéâäàåçêëèïîìÄÅ" +
	"ÉæÆôöòûùÿÖÜ¢£¥₧ƒ" +
	"áíóúñÑªº¿⌐¬½¼¡«»" +
	"░▒▓│┤╡╢╖╕╣║╗╝╜╛┐" +
	"└┴┬├─┼╞╟╚╔╩╦╠═╬╧" +
	"╨╤╥╙╘╒╓╫╪┘┌█▄▌▐▀" +
	"αßΓπΣσµτΦΘΩδ∞φε∩" +
	"≡±≥≤⌠⌡÷≈°∙·√ⁿ²■\xa0"
