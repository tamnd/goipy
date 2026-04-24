package vm

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildShutil() *object.Module {
	m := &object.Module{Name: "shutil", Dict: object.NewDict()}

	// Exception classes
	sameFileErr := &object.Class{Name: "SameFileError", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	shutilErr := &object.Class{Name: "Error", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	readErr := &object.Class{Name: "ReadError", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	m.Dict.SetStr("SameFileError", sameFileErr)
	m.Dict.SetStr("Error", shutilErr)
	m.Dict.SetStr("ReadError", readErr)

	// copyfileobj(fsrc, fdst, length=0)
	m.Dict.SetStr("copyfileobj", &object.BuiltinFunc{Name: "copyfileobj", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copyfileobj() requires fsrc and fdst")
		}
		return object.None, shutilCopyFileObj(i, a[0], a[1])
	}})

	// copyfile(src, dst, *, follow_symlinks=True)
	m.Dict.SetStr("copyfile", &object.BuiltinFunc{Name: "copyfile", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copyfile() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copyfile() args must be str")
		}
		srcAbs, _ := filepath.Abs(src.V)
		dstAbs, _ := filepath.Abs(dst.V)
		if srcAbs == dstAbs {
			return nil, object.NewException(sameFileErr, src.V+" and "+dst.V+" are the same file")
		}
		if err := shutilCopyFileContent(src.V, dst.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: dst.V}, nil
	}})

	// copymode(src, dst, *, follow_symlinks=True)
	m.Dict.SetStr("copymode", &object.BuiltinFunc{Name: "copymode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copymode() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copymode() args must be str")
		}
		info, err := os.Stat(src.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		if err := os.Chmod(dst.V, info.Mode().Perm()); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// copystat(src, dst, *, follow_symlinks=True)
	m.Dict.SetStr("copystat", &object.BuiltinFunc{Name: "copystat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copystat() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copystat() args must be str")
		}
		info, err := os.Stat(src.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		if err := os.Chmod(dst.V, info.Mode().Perm()); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		mtime := info.ModTime()
		_ = os.Chtimes(dst.V, mtime, mtime)
		return object.None, nil
	}})

	// copy(src, dst, *, follow_symlinks=True) — content + mode
	shutilCopy := func(src, dst string) (string, error) {
		info, err := os.Stat(dst)
		if err == nil && info.IsDir() {
			dst = filepath.Join(dst, filepath.Base(src))
		}
		if err := shutilCopyFileContent(src, dst); err != nil {
			return "", err
		}
		srcInfo, err := os.Stat(src)
		if err == nil {
			_ = os.Chmod(dst, srcInfo.Mode().Perm())
		}
		return dst, nil
	}

	m.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copy() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copy() args must be str")
		}
		result, err := shutilCopy(src.V, dst.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: result}, nil
	}})

	// copy2(src, dst, *, follow_symlinks=True) — content + full stat
	m.Dict.SetStr("copy2", &object.BuiltinFunc{Name: "copy2", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copy2() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copy2() args must be str")
		}
		dstPath := dst.V
		info, err := os.Stat(dstPath)
		if err == nil && info.IsDir() {
			dstPath = filepath.Join(dstPath, filepath.Base(src.V))
		}
		if err := shutilCopyFileContent(src.V, dstPath); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		srcInfo, err := os.Stat(src.V)
		if err == nil {
			_ = os.Chmod(dstPath, srcInfo.Mode().Perm())
			mtime := srcInfo.ModTime()
			_ = os.Chtimes(dstPath, mtime, mtime)
		}
		return &object.Str{V: dstPath}, nil
	}})

	// ignore_patterns(*patterns) — returns a callable
	m.Dict.SetStr("ignore_patterns", &object.BuiltinFunc{Name: "ignore_patterns", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		patterns := make([]string, len(a))
		for idx, arg := range a {
			s, ok := arg.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "ignore_patterns() args must be str")
			}
			patterns[idx] = s.V
		}
		fn := &object.BuiltinFunc{Name: "ignore_patterns_fn", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			// args: (directory, contents) — contents is a list of names
			if len(args) < 2 {
				return nil, object.Errorf(i.typeErr, "ignore function requires 2 args")
			}
			namesObj, err := i.iterStrings(args[1])
			if err != nil {
				return nil, err
			}
			ignoredSet := object.NewSet()
			for _, name := range namesObj {
				for _, pat := range patterns {
					if matched, _ := filepath.Match(pat, name); matched {
						_ = ignoredSet.Add(&object.Str{V: name})
						break
					}
				}
			}
			return ignoredSet, nil
		}}
		return fn, nil
	}})

	// copytree(src, dst, symlinks=False, ignore=None, copy_function=copy2,
	//          ignore_dangling_symlinks=False, dirs_exist_ok=False)
	m.Dict.SetStr("copytree", &object.BuiltinFunc{Name: "copytree", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copytree() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "copytree() args must be str")
		}
		dirsExistOk := false
		var ignoreFn *object.BuiltinFunc
		if kw != nil {
			if v, ok := kw.GetStr("dirs_exist_ok"); ok {
				dirsExistOk = object.Truthy(v)
			}
			if v, ok := kw.GetStr("ignore"); ok && v != object.None {
				if bf, ok := v.(*object.BuiltinFunc); ok {
					ignoreFn = bf
				}
			}
		}
		if err := shutilCopyTree(i, src.V, dst.V, dirsExistOk, ignoreFn); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: dst.V}, nil
	}})

	// rmtree(path, ignore_errors=False, onerror=None, *, onexc=None, dir_fd=None)
	m.Dict.SetStr("rmtree", &object.BuiltinFunc{Name: "rmtree", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "rmtree() requires path")
		}
		path, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "rmtree() path must be str")
		}
		ignoreErrors := false
		if len(a) > 1 {
			ignoreErrors = object.Truthy(a[1])
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("ignore_errors"); ok2 {
				ignoreErrors = object.Truthy(v)
			}
		}
		err := os.RemoveAll(path.V)
		if err != nil && !ignoreErrors {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// move(src, dst, copy_function=copy2)
	m.Dict.SetStr("move", &object.BuiltinFunc{Name: "move", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "move() requires src and dst")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "move() args must be str")
		}
		dstPath := dst.V
		// If dst is an existing directory, move src inside it.
		if info, err := os.Stat(dstPath); err == nil && info.IsDir() {
			dstPath = filepath.Join(dstPath, filepath.Base(src.V))
		}
		// Try atomic rename first.
		if err := os.Rename(src.V, dstPath); err != nil {
			// Cross-device: copy then remove.
			srcInfo, serr := os.Stat(src.V)
			if serr != nil {
				return nil, object.Errorf(i.osErr, "%v", serr)
			}
			if srcInfo.IsDir() {
				if err2 := shutilCopyTree(i, src.V, dstPath, false, nil); err2 != nil {
					return nil, object.Errorf(i.osErr, "%v", err2)
				}
				if err2 := os.RemoveAll(src.V); err2 != nil {
					return nil, object.Errorf(i.osErr, "%v", err2)
				}
			} else {
				if _, err2 := shutilCopy(src.V, dstPath); err2 != nil {
					return nil, object.Errorf(i.osErr, "%v", err2)
				}
				if err2 := os.Remove(src.V); err2 != nil {
					return nil, object.Errorf(i.osErr, "%v", err2)
				}
			}
		}
		return &object.Str{V: dstPath}, nil
	}})

	// disk_usage(path) — returns named tuple (total, used, free)
	m.Dict.SetStr("disk_usage", &object.BuiltinFunc{Name: "disk_usage", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "disk_usage() requires path")
		}
		path, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "disk_usage() path must be str")
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path.V, &stat); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		total := int64(stat.Blocks) * int64(stat.Bsize)
		free := int64(stat.Bavail) * int64(stat.Bsize)
		used := total - int64(stat.Bfree)*int64(stat.Bsize)
		cls := &object.Class{Name: "usage", Bases: nil, Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("total", object.NewInt(total))
		inst.Dict.SetStr("used", object.NewInt(used))
		inst.Dict.SetStr("free", object.NewInt(free))
		return inst, nil
	}})

	// which(cmd, mode=os.F_OK|os.X_OK, path=None)
	m.Dict.SetStr("which", &object.BuiltinFunc{Name: "which", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "which() requires cmd")
		}
		cmd, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "which() cmd must be str")
		}
		result, err := exec.LookPath(cmd.V)
		if err != nil {
			return object.None, nil
		}
		return &object.Str{V: result}, nil
	}})

	// get_terminal_size(fallback=(80, 24))
	m.Dict.SetStr("get_terminal_size", &object.BuiltinFunc{Name: "get_terminal_size", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		fallbackCols, fallbackRows := 80, 24
		if len(a) > 0 {
			if tup, ok := a[0].(*object.Tuple); ok && len(tup.V) == 2 {
				if c, ok := tup.V[0].(*object.Int); ok {
					fallbackCols = int(c.Int64())
				}
				if r, ok := tup.V[1].(*object.Int); ok {
					fallbackRows = int(r.Int64())
				}
			}
		}
		cols, rows := os.Getenv("COLUMNS"), os.Getenv("LINES")
		ncols, nrows := fallbackCols, fallbackRows
		if c := parseEnvInt(cols); c > 0 {
			ncols = c
		}
		if r := parseEnvInt(rows); r > 0 {
			nrows = r
		}
		cls := &object.Class{Name: "terminal_size", Bases: nil, Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("columns", object.NewInt(int64(ncols)))
		inst.Dict.SetStr("lines", object.NewInt(int64(nrows)))
		return inst, nil
	}})

	// get_archive_formats()
	m.Dict.SetStr("get_archive_formats", &object.BuiltinFunc{Name: "get_archive_formats", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		formats := [][]string{
			{"bztar", "bzip2'ed tar-file"},
			{"gztar", "gzip'ed tar-file"},
			{"tar", "uncompressed tar file"},
			{"xztar", "xz'ed tar-file"},
			{"zip", "ZIP file"},
		}
		out := make([]object.Object, len(formats))
		for idx, f := range formats {
			tup := &object.Tuple{V: []object.Object{
				&object.Str{V: f[0]},
				&object.Str{V: f[1]},
			}}
			out[idx] = tup
		}
		return &object.List{V: out}, nil
	}})

	// get_unpack_formats()
	m.Dict.SetStr("get_unpack_formats", &object.BuiltinFunc{Name: "get_unpack_formats", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		formats := []struct {
			name string
			exts []string
			desc string
		}{
			{"bztar", []string{".tar.bz2", ".tbz2"}, "bzip2'ed tar-file"},
			{"gztar", []string{".tar.gz", ".tgz"}, "gzip'ed tar-file"},
			{"tar", []string{".tar"}, "uncompressed tar file"},
			{"xztar", []string{".tar.xz", ".txz"}, "xz'ed tar-file"},
			{"zip", []string{".zip"}, "ZIP file"},
		}
		out := make([]object.Object, len(formats))
		for idx, f := range formats {
			exts := make([]object.Object, len(f.exts))
			for j, e := range f.exts {
				exts[j] = &object.Str{V: e}
			}
			tup := &object.Tuple{V: []object.Object{
				&object.Str{V: f.name},
				&object.List{V: exts},
				&object.Str{V: f.desc},
			}}
			out[idx] = tup
		}
		return &object.List{V: out}, nil
	}})

	// make_archive(base_name, format, root_dir=None, base_dir=None, ...)
	m.Dict.SetStr("make_archive", &object.BuiltinFunc{Name: "make_archive", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "make_archive() requires base_name and format")
		}
		baseName, ok1 := a[0].(*object.Str)
		format, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "make_archive() args must be str")
		}
		rootDir := ""
		baseDir := "."
		if len(a) > 2 {
			if s, ok := a[2].(*object.Str); ok {
				rootDir = s.V
			}
		}
		if len(a) > 3 {
			if s, ok := a[3].(*object.Str); ok {
				baseDir = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("root_dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					rootDir = s.V
				}
			}
			if v, ok := kw.GetStr("base_dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					baseDir = s.V
				}
			}
		}
		archPath, err := shutilMakeArchive(baseName.V, format.V, rootDir, baseDir)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: archPath}, nil
	}})

	// unpack_archive(filename, extract_dir=None, format=None)
	m.Dict.SetStr("unpack_archive", &object.BuiltinFunc{Name: "unpack_archive", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "unpack_archive() requires filename")
		}
		filename, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "unpack_archive() filename must be str")
		}
		extractDir := "."
		format := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				extractDir = s.V
			}
		}
		if len(a) > 2 {
			if s, ok := a[2].(*object.Str); ok {
				format = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("extract_dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					extractDir = s.V
				}
			}
			if v, ok := kw.GetStr("format"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					format = s.V
				}
			}
		}
		if err := shutilUnpackArchive(filename.V, extractDir, format); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	return m
}

// shutilCopyFileObj copies from src to dst file-like objects using read/write methods.
func shutilCopyFileObj(i *Interp, src, dst object.Object) error {
	readFn, err := i.getAttr(src, "read")
	if err != nil {
		return err
	}
	writeFn, err := i.getAttr(dst, "write")
	if err != nil {
		return err
	}
	for {
		chunk, err := i.callObject(readFn, []object.Object{object.NewInt(16384)}, nil)
		if err != nil {
			return err
		}
		switch v := chunk.(type) {
		case *object.Bytes:
			if len(v.V) == 0 {
				return nil
			}
		case *object.Str:
			if v.V == "" {
				return nil
			}
		default:
			return nil
		}
		if _, err := i.callObject(writeFn, []object.Object{chunk}, nil); err != nil {
			return err
		}
	}
}

// shutilCopyFileContent copies file bytes from src to dst path.
// os.ReadFile/WriteFile avoids copy_file_range on Linux (which does not fall
// back on EBADF the way it does for ENOSYS/EXDEV).
func shutilCopyFileContent(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o666)
}

// shutilCopyTree copies a directory tree from src to dst.
func shutilCopyTree(i *Interp, src, dst string, dirsExistOk bool, ignoreFn *object.BuiltinFunc) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil && !dirsExistOk {
				return err
			}
			// Apply ignore filter to directory contents.
			if ignoreFn != nil && rel != "." {
				entries, rerr := os.ReadDir(path)
				if rerr == nil {
					names := make([]object.Object, len(entries))
					for idx, e := range entries {
						names[idx] = &object.Str{V: e.Name()}
					}
					ignored, _ := ignoreFn.Call(nil, []object.Object{
						&object.Str{V: path},
						&object.List{V: names},
					}, nil)
					_ = ignored // set is used by walking; we handle below
				}
			}
			return nil
		}

		// Check if this file should be ignored.
		if ignoreFn != nil {
			parentDir := filepath.Dir(path)
			baseName := d.Name()
			entries, rerr := os.ReadDir(parentDir)
			if rerr == nil {
				names := make([]object.Object, len(entries))
				for idx, e := range entries {
					names[idx] = &object.Str{V: e.Name()}
				}
				ignored, _ := ignoreFn.Call(nil, []object.Object{
					&object.Str{V: parentDir},
					&object.List{V: names},
				}, nil)
				if s, ok := ignored.(*object.Set); ok {
					for _, k := range s.Items() {
						if ks, ok := k.(*object.Str); ok && ks.V == baseName {
							return nil // skip
						}
					}
				}
			}
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := shutilCopyFileContent(path, dstPath); err != nil {
			return err
		}
		_ = os.Chmod(dstPath, info.Mode().Perm())
		return nil
	})
}

// shutilMakeArchive creates a zip or tar archive.
func shutilMakeArchive(baseName, format, rootDir, baseDir string) (string, error) {
	sourceDir := rootDir
	if sourceDir == "" {
		sourceDir = "."
	}
	if baseDir == "" {
		baseDir = "."
	}

	switch format {
	case "zip":
		archPath := baseName + ".zip"
		f, err := os.Create(archPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		w := zip.NewWriter(f)
		defer w.Close()
		err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(sourceDir, path)
			fw, err := w.Create(rel)
			if err != nil {
				return err
			}
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()
			_, err = io.Copy(fw, src)
			return err
		})
		if err != nil {
			return "", err
		}
		return archPath, nil

	case "tar":
		archPath := baseName + ".tar"
		return shutilWriteTar(archPath, sourceDir, nil)

	case "gztar":
		archPath := baseName + ".tar.gz"
		return shutilWriteTarGz(archPath, sourceDir)

	default:
		return "", &os.PathError{Op: "make_archive", Path: baseName, Err: os.ErrInvalid}
	}
}

func shutilWriteTar(archPath, sourceDir string, compress func(io.Writer) (io.WriteCloser, error)) (string, error) {
	f, err := os.Create(archPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var tw *tar.Writer
	if compress != nil {
		cw, err := compress(f)
		if err != nil {
			return "", err
		}
		defer cw.Close()
		tw = tar.NewWriter(cw)
	} else {
		tw = tar.NewWriter(f)
	}
	defer tw.Close()
	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(sourceDir, path)
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr := &tar.Header{
			Name:    rel,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		return err
	})
	if err != nil {
		return "", err
	}
	return archPath, nil
}

func shutilWriteTarGz(archPath, sourceDir string) (string, error) {
	return shutilWriteTar(archPath, sourceDir, func(w io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriter(w), nil
	})
}

// shutilUnpackArchive extracts a zip or tar archive to extractDir.
func shutilUnpackArchive(filename, extractDir, format string) error {
	if format == "" {
		switch {
		case strings.HasSuffix(filename, ".zip"):
			format = "zip"
		case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
			format = "gztar"
		case strings.HasSuffix(filename, ".tar.bz2") || strings.HasSuffix(filename, ".tbz2"):
			format = "bztar"
		case strings.HasSuffix(filename, ".tar"):
			format = "tar"
		default:
			return &os.PathError{Op: "unpack_archive", Path: filename, Err: os.ErrInvalid}
		}
	}
	switch format {
	case "zip":
		return shutilUnpackZip(filename, extractDir)
	case "tar":
		return shutilUnpackTar(filename, extractDir, nil)
	case "gztar":
		return shutilUnpackTar(filename, extractDir, func(r io.Reader) (io.Reader, error) {
			return gzip.NewReader(r)
		})
	default:
		return &os.PathError{Op: "unpack_archive", Path: filename, Err: os.ErrInvalid}
	}
}

func shutilUnpackZip(filename, extractDir string) error {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		dstPath := filepath.Join(extractDir, filepath.Clean(f.Name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		out, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func shutilUnpackTar(filename, extractDir string, decompress func(io.Reader) (io.Reader, error)) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if decompress != nil {
		r, err = decompress(f)
		if err != nil {
			return err
		}
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		dstPath := filepath.Join(extractDir, filepath.Clean(hdr.Name))
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		out, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// parseEnvInt parses an integer from an environment variable string.
func parseEnvInt(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
