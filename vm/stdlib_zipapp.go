package vm

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildZipapp() *object.Module {
	m := &object.Module{Name: "zipapp", Dict: object.NewDict()}

	// isZipFile reports whether the file at path begins with a PK zip signature,
	// possibly after a shebang line.
	isZipFile := func(path string) bool {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		defer f.Close()
		var hdr [4]byte
		f.Read(hdr[:]) //nolint
		if hdr[0] == '#' && hdr[1] == '!' {
			// skip shebang line
			f.Seek(0, io.SeekStart) //nolint
			io.ReadAll(io.LimitReader(f, 512)) //nolint -- find newline; simplistic
			f.Seek(0, io.SeekStart) //nolint
			// find first \n
			buf := make([]byte, 512)
			n, _ := f.Read(buf)
			nl := bytes.IndexByte(buf[:n], '\n')
			if nl < 0 {
				return false
			}
			f.Seek(int64(nl+1), io.SeekStart) //nolint
			f.Read(hdr[:]) //nolint
		}
		return hdr[0] == 'P' && hdr[1] == 'K' && hdr[2] == 3 && hdr[3] == 4
	}

	// readZipBytes returns the raw zip bytes from an archive file, skipping any shebang.
	readZipBytes := func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(data) >= 2 && data[0] == '#' && data[1] == '!' {
			nl := bytes.IndexByte(data, '\n')
			if nl >= 0 {
				data = data[nl+1:]
			}
		}
		return data, nil
	}

	// generateMain builds the __main__.py content for a "module:callable" spec.
	generateMain := func(spec string) string {
		parts := strings.SplitN(spec, ":", 2)
		mod := parts[0]
		if len(parts) == 2 {
			return fmt.Sprintf("import sys\nfrom %s import %s\nsys.exit(%s())\n", mod, parts[1], parts[1])
		}
		return fmt.Sprintf("import runpy\nrunpy.run_module(%q, run_name='__main__', alter_sys=True)\n", mod)
	}

	m.Dict.SetStr("create_archive", &object.BuiltinFunc{
		Name: "create_archive",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.valueErr, "create_archive requires source")
			}
			sourceStr := ""
			if s, ok := a[0].(*object.Str); ok {
				sourceStr = s.V
			}
			var targetStr string
			var interpreter string
			var mainSpec string
			compressed := false

			if len(a) >= 2 && a[1] != object.None {
				if s, ok := a[1].(*object.Str); ok {
					targetStr = s.V
				}
			}
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					interpreter = s.V
				}
			}
			if len(a) >= 4 && a[3] != object.None {
				if s, ok := a[3].(*object.Str); ok {
					mainSpec = s.V
				}
			}
			// a[4] is filter — accepted but ignored
			if len(a) >= 6 {
				if b, ok := a[5].(*object.Bool); ok {
					compressed = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("target"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						targetStr = s.V
					}
				}
				if v, ok := kw.GetStr("interpreter"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						interpreter = s.V
					}
				}
				if v, ok := kw.GetStr("main"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						mainSpec = s.V
					}
				}
				if v, ok := kw.GetStr("compressed"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						compressed = b.V
					}
				}
			}

			// determine compression method
			comprMethod := zip.Store
			if compressed {
				comprMethod = zip.Deflate
			}

			// build zip bytes
			var zipBuf bytes.Buffer

			if isZipFile(sourceStr) {
				// source is an existing archive — copy its zip bytes
				if targetStr == "" {
					return nil, object.Errorf(i.valueErr, "target is required when source is an archive")
				}
				raw, err := readZipBytes(sourceStr)
				if err != nil {
					return nil, object.Errorf(i.valueErr, "cannot read source archive: %v", err)
				}
				zipBuf.Write(raw)
			} else {
				// source is a directory
				if targetStr == "" {
					targetStr = sourceStr + ".pyz"
				}
				zw := zip.NewWriter(&zipBuf)
				// walk directory
				err := filepath.Walk(sourceStr, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(sourceStr, path)
					if err != nil {
						return err
					}
					rel = filepath.ToSlash(rel)
					fh := &zip.FileHeader{
						Name:   rel,
						Method: uint16(comprMethod),
					}
					w, err := zw.CreateHeader(fh)
					if err != nil {
						return err
					}
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					_, err = w.Write(data)
					return err
				})
				if err != nil {
					return nil, object.Errorf(i.valueErr, "cannot create archive: %v", err)
				}
				// add __main__.py from main= spec if provided
				if mainSpec != "" {
					fh := &zip.FileHeader{
						Name:   "__main__.py",
						Method: uint16(comprMethod),
					}
					w, err := zw.CreateHeader(fh)
					if err != nil {
						return nil, object.Errorf(i.valueErr, "cannot write __main__.py: %v", err)
					}
					w.Write([]byte(generateMain(mainSpec))) //nolint
				}
				if err := zw.Close(); err != nil {
					return nil, object.Errorf(i.valueErr, "cannot finalise archive: %v", err)
				}
			}

			// write target with optional shebang prefix
			var out bytes.Buffer
			if interpreter != "" {
				out.WriteString("#!" + interpreter + "\n")
			}
			out.Write(zipBuf.Bytes())

			if err := os.WriteFile(targetStr, out.Bytes(), 0o755); err != nil {
				return nil, object.Errorf(i.valueErr, "cannot write target: %v", err)
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_interpreter", &object.BuiltinFunc{
		Name: "get_interpreter",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.valueErr, "get_interpreter requires archive")
			}
			path := ""
			if s, ok := a[0].(*object.Str); ok {
				path = s.V
			}
			f, err := os.Open(path)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "cannot open archive: %v", err)
			}
			defer f.Close()
			var hdr [2]byte
			if _, err := f.Read(hdr[:]); err != nil {
				return object.None, nil
			}
			if hdr[0] != '#' || hdr[1] != '!' {
				return object.None, nil
			}
			// read rest of first line
			line := []byte{'#', '!'}
			buf := make([]byte, 1)
			for {
				_, err := f.Read(buf)
				if err != nil || buf[0] == '\n' {
					break
				}
				line = append(line, buf[0])
			}
			interp := strings.TrimPrefix(string(line), "#!")
			return &object.Str{V: interp}, nil
		},
	})

	return m
}
