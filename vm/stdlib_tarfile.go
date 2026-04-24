package vm

import (
	"archive/tar"
	"bytes"
	stdbzip2 "compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/goipy/lib/bzip2"
	"github.com/tamnd/goipy/lib/lzma"
	"github.com/tamnd/goipy/object"
)

// Python `tarfile` stdlib module.
//
// Read and write tar archives. Compression modes (gz, bz2, xz) layer
// through compress/gzip, our lib/bzip2, and our lib/lzma (XZ).

func (i *Interp) buildTarfile() *object.Module {
	m := &object.Module{Name: "tarfile", Dict: object.NewDict()}

	tarErr := &object.Class{Name: "TarError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	readErr := &object.Class{Name: "ReadError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	compressionErr := &object.Class{Name: "CompressionError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	streamErr := &object.Class{Name: "StreamError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	extractErr := &object.Class{Name: "ExtractError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	headerErr := &object.Class{Name: "HeaderError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	filterErr := &object.Class{Name: "FilterError", Bases: []*object.Class{tarErr}, Dict: object.NewDict()}
	absPathErr := &object.Class{Name: "AbsolutePathError", Bases: []*object.Class{filterErr}, Dict: object.NewDict()}
	outsideErr := &object.Class{Name: "OutsideDestinationError", Bases: []*object.Class{filterErr}, Dict: object.NewDict()}
	specialErr := &object.Class{Name: "SpecialFileError", Bases: []*object.Class{filterErr}, Dict: object.NewDict()}
	absLinkErr := &object.Class{Name: "AbsoluteLinkError", Bases: []*object.Class{filterErr}, Dict: object.NewDict()}
	linkOutErr := &object.Class{Name: "LinkOutsideDestinationError", Bases: []*object.Class{filterErr}, Dict: object.NewDict()}

	for name, cls := range map[string]*object.Class{
		"TarError": tarErr, "ReadError": readErr, "CompressionError": compressionErr,
		"StreamError": streamErr, "ExtractError": extractErr, "HeaderError": headerErr,
		"FilterError": filterErr, "AbsolutePathError": absPathErr,
		"OutsideDestinationError": outsideErr, "SpecialFileError": specialErr,
		"AbsoluteLinkError": absLinkErr, "LinkOutsideDestinationError": linkOutErr,
	} {
		m.Dict.SetStr(name, cls)
	}

	// Type constants (match CPython bytestrings).
	for name, v := range map[string]string{
		"REGTYPE": "0", "AREGTYPE": "\x00", "LNKTYPE": "1", "SYMTYPE": "2",
		"CHRTYPE": "3", "BLKTYPE": "4", "DIRTYPE": "5", "FIFOTYPE": "6",
		"CONTTYPE": "7", "XHDTYPE": "x", "XGLTYPE": "g",
	} {
		m.Dict.SetStr(name, &object.Bytes{V: []byte(v)})
	}

	for name, v := range map[string]int64{
		"USTAR_FORMAT":   0,
		"GNU_FORMAT":     1,
		"PAX_FORMAT":     2,
		"DEFAULT_FORMAT": 2,
	} {
		m.Dict.SetStr(name, object.NewInt(v))
	}

	tarInfoCls := i.buildTarInfoClass()
	m.Dict.SetStr("TarInfo", tarInfoCls)

	tarFileCls := i.buildTarFileClass(tarInfoCls, readErr)
	m.Dict.SetStr("TarFile", tarFileCls)

	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		name := ""
		if len(args) >= 1 {
			if s, ok := args[0].(*object.Str); ok {
				name = s.V
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
		}
		mode := "r"
		if len(args) >= 2 {
			if s, ok := args[1].(*object.Str); ok {
				mode = s.V
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("mode"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					mode = s.V
				}
			}
		}
		inst := &object.Instance{Class: tarFileCls, Dict: object.NewDict()}
		if err := i.initTarFileInstance(inst, name, mode, tarInfoCls, readErr); err != nil {
			return nil, err
		}
		return inst, nil
	}})

	m.Dict.SetStr("is_tarfile", &object.BuiltinFunc{Name: "is_tarfile", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return object.False, nil
		}
		var data []byte
		switch a := args[0].(type) {
		case *object.Str:
			b, err := os.ReadFile(a.V)
			if err != nil {
				return object.False, nil
			}
			data = b
		case *object.Bytes:
			data = a.V
		default:
			return object.False, nil
		}
		// Try each compression.
		for _, decoder := range []func([]byte) ([]byte, error){
			func(b []byte) ([]byte, error) { return b, nil },
			gunzipBytes,
			bunzipBytes,
			unxzBytes,
		} {
			raw, err := decoder(data)
			if err != nil {
				continue
			}
			tr := tar.NewReader(bytes.NewReader(raw))
			if _, err := tr.Next(); err == nil {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	return m
}

// ── TarInfo ───────────────────────────────────────────────────────────────

func (i *Interp) buildTarInfoClass() *object.Class {
	cls := &object.Class{Name: "TarInfo", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "TarInfo() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TarInfo() self must be instance")
		}
		name := ""
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				name = s.V
			}
		}
		initTarInfo(self, &tar.Header{Name: name, Typeflag: tar.TypeReg})
		return nil, nil
	}})
	return cls
}

func initTarInfo(inst *object.Instance, h *tar.Header) {
	d := inst.Dict
	d.SetStr("name", &object.Str{V: h.Name})
	d.SetStr("size", object.NewInt(h.Size))
	d.SetStr("mtime", object.NewInt(h.ModTime.Unix()))
	d.SetStr("mode", object.NewInt(h.Mode))
	d.SetStr("type", &object.Bytes{V: []byte{h.Typeflag}})
	d.SetStr("linkname", &object.Str{V: h.Linkname})
	d.SetStr("uid", object.NewInt(int64(h.Uid)))
	d.SetStr("gid", object.NewInt(int64(h.Gid)))
	d.SetStr("uname", &object.Str{V: h.Uname})
	d.SetStr("gname", &object.Str{V: h.Gname})
	d.SetStr("pax_headers", object.NewDict())

	typeflag := h.Typeflag
	mkPred := func(want byte) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: "pred", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if typeflag == want {
				return object.True, nil
			}
			return object.False, nil
		}}
	}
	d.SetStr("isdir", mkPred(tar.TypeDir))
	d.SetStr("issym", mkPred(tar.TypeSymlink))
	d.SetStr("islnk", mkPred(tar.TypeLink))
	d.SetStr("ischr", mkPred(tar.TypeChar))
	d.SetStr("isblk", mkPred(tar.TypeBlock))
	d.SetStr("isfifo", mkPred(tar.TypeFifo))

	isReg := typeflag == tar.TypeReg || typeflag == tar.TypeRegA
	d.SetStr("isreg", &object.BuiltinFunc{Name: "isreg", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if isReg {
			return object.True, nil
		}
		return object.False, nil
	}})
	d.SetStr("isfile", &object.BuiltinFunc{Name: "isfile", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if isReg {
			return object.True, nil
		}
		return object.False, nil
	}})
	isDev := typeflag == tar.TypeChar || typeflag == tar.TypeBlock
	d.SetStr("isdev", &object.BuiltinFunc{Name: "isdev", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if isDev {
			return object.True, nil
		}
		return object.False, nil
	}})
}

func newTarInfoInstance(cls *object.Class, h *tar.Header) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	initTarInfo(inst, h)
	return inst
}

// tarInfoToHeader extracts fields from a Python TarInfo instance.
func tarInfoToHeader(inst *object.Instance) *tar.Header {
	h := &tar.Header{Typeflag: tar.TypeReg}
	d := inst.Dict
	if v, ok := d.GetStr("name"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			h.Name = s.V
		}
	}
	if v, ok := d.GetStr("size"); ok {
		if n, ok2 := toInt64(v); ok2 {
			h.Size = n
		}
	}
	if v, ok := d.GetStr("mtime"); ok {
		if n, ok2 := toInt64(v); ok2 {
			h.ModTime = time.Unix(n, 0)
		}
	}
	if v, ok := d.GetStr("mode"); ok {
		if n, ok2 := toInt64(v); ok2 {
			h.Mode = n
		}
	}
	if v, ok := d.GetStr("type"); ok {
		if b, ok2 := v.(*object.Bytes); ok2 && len(b.V) == 1 {
			h.Typeflag = b.V[0]
		}
	}
	if v, ok := d.GetStr("linkname"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			h.Linkname = s.V
		}
	}
	if v, ok := d.GetStr("uid"); ok {
		if n, ok2 := toInt64(v); ok2 {
			h.Uid = int(n)
		}
	}
	if v, ok := d.GetStr("gid"); ok {
		if n, ok2 := toInt64(v); ok2 {
			h.Gid = int(n)
		}
	}
	if v, ok := d.GetStr("uname"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			h.Uname = s.V
		}
	}
	if v, ok := d.GetStr("gname"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			h.Gname = s.V
		}
	}
	return h
}

// ── TarFile ───────────────────────────────────────────────────────────────

type tarEntry struct {
	header *tar.Header
	data   []byte
}

type tarFileState struct {
	filename   string
	mode       string // raw "r", "r:gz", "w:xz", etc.
	compType   string // "", "gz", "bz2", "xz"
	writeMode  bool
	closed     bool
	entries    []*tarEntry
	byName     map[string]*tarEntry
	wbuf       *bytes.Buffer
	writer     *tar.Writer
}

func parseTarMode(mode string) (op, comp string) {
	if mode == "" {
		return "r", ""
	}
	// e.g. "r", "r:", "r:gz", "r:*", "w", "w:bz2", "x:xz", "a"
	parts := strings.SplitN(mode, ":", 2)
	op = parts[0]
	if len(parts) == 2 {
		comp = parts[1]
	}
	return op, comp
}

func (i *Interp) buildTarFileClass(tarInfoCls *object.Class, readErr *object.Class) *object.Class {
	cls := &object.Class{Name: "TarFile", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "TarFile() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TarFile() self must be instance")
		}
		name := ""
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				name = s.V
			}
		}
		mode := "r"
		if len(args) >= 3 {
			if s, ok2 := args[2].(*object.Str); ok2 {
				mode = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
		}
		return nil, i.initTarFileInstance(self, name, mode, tarInfoCls, readErr)
	}})
	return cls
}

func (i *Interp) initTarFileInstance(inst *object.Instance, filename, mode string, tarInfoCls, readErr *object.Class) error {
	op, comp := parseTarMode(mode)
	st := &tarFileState{filename: filename, mode: mode, compType: comp, byName: map[string]*tarEntry{}}
	d := inst.Dict
	d.SetStr("name", &object.Str{V: filename})
	d.SetStr("mode", &object.Str{V: mode})

	switch op {
	case "r":
		raw, err := os.ReadFile(filename)
		if err != nil {
			return object.Errorf(readErr, "%v", err)
		}
		decompressed, err := tarDecompress(raw, comp)
		if err != nil {
			return object.Errorf(readErr, "%v", err)
		}
		tr := tar.NewReader(bytes.NewReader(decompressed))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return object.Errorf(readErr, "%v", err)
			}
			var data []byte
			if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA {
				b, rerr := io.ReadAll(tr)
				if rerr != nil {
					return object.Errorf(readErr, "%v", rerr)
				}
				data = b
			}
			e := &tarEntry{header: h, data: data}
			st.entries = append(st.entries, e)
			st.byName[h.Name] = e
		}
	case "w", "x":
		st.writeMode = true
		st.wbuf = &bytes.Buffer{}
		st.writer = tar.NewWriter(st.wbuf)
	case "a":
		st.writeMode = true
		// Read existing entries into memory, then open a fresh writer.
		if raw, err := os.ReadFile(filename); err == nil {
			decompressed, derr := tarDecompress(raw, comp)
			if derr == nil {
				tr := tar.NewReader(bytes.NewReader(decompressed))
				for {
					h, terr := tr.Next()
					if terr == io.EOF {
						break
					}
					if terr != nil {
						break
					}
					var data []byte
					if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA {
						b, _ := io.ReadAll(tr)
						data = b
					}
					e := &tarEntry{header: h, data: data}
					st.entries = append(st.entries, e)
					st.byName[h.Name] = e
				}
			}
		}
		st.wbuf = &bytes.Buffer{}
		st.writer = tar.NewWriter(st.wbuf)
		for _, e := range st.entries {
			if err := st.writer.WriteHeader(e.header); err != nil {
				return object.Errorf(readErr, "%v", err)
			}
			if len(e.data) > 0 {
				if _, err := st.writer.Write(e.data); err != nil {
					return object.Errorf(readErr, "%v", err)
				}
			}
		}
	default:
		return object.Errorf(i.valueErr, "unknown tarfile mode %q", mode)
	}

	d.SetStr("getmembers", &object.BuiltinFunc{Name: "getmembers", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var out []object.Object
		for _, e := range st.entries {
			out = append(out, newTarInfoInstance(tarInfoCls, e.header))
		}
		return &object.List{V: out}, nil
	}})

	d.SetStr("getnames", &object.BuiltinFunc{Name: "getnames", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var out []object.Object
		for _, e := range st.entries {
			out = append(out, &object.Str{V: e.header.Name})
		}
		return &object.List{V: out}, nil
	}})

	d.SetStr("getmember", &object.BuiltinFunc{Name: "getmember", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "getmember() missing name")
		}
		name, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getmember() name must be str")
		}
		if e, ok := st.byName[name.V]; ok {
			return newTarInfoInstance(tarInfoCls, e.header), nil
		}
		return nil, object.Errorf(i.keyErr, "filename %q not found", name.V)
	}})

	d.SetStr("extractfile", &object.BuiltinFunc{Name: "extractfile", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "extractfile() missing member")
		}
		name, err := tarMemberName(args[0])
		if err != nil {
			return nil, err
		}
		e, ok := st.byName[name]
		if !ok {
			return nil, object.Errorf(i.keyErr, "filename %q not found", name)
		}
		if e.header.Typeflag != tar.TypeReg && e.header.Typeflag != tar.TypeRegA {
			return object.None, nil
		}
		return i.makeBytesIOLike(e.data), nil
	}})

	d.SetStr("extract", &object.BuiltinFunc{Name: "extract", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "extract() missing member")
		}
		name, err := tarMemberName(args[0])
		if err != nil {
			return nil, err
		}
		dest := "."
		if len(args) >= 2 {
			if s, ok := args[1].(*object.Str); ok {
				dest = s.V
			}
		}
		e, ok := st.byName[name]
		if !ok {
			return nil, object.Errorf(i.keyErr, "filename %q not found", name)
		}
		return object.None, tarExtractEntry(e, dest)
	}})

	d.SetStr("extractall", &object.BuiltinFunc{Name: "extractall", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		dest := "."
		if len(args) >= 1 {
			if s, ok := args[0].(*object.Str); ok {
				dest = s.V
			}
		}
		for _, e := range st.entries {
			if err := tarExtractEntry(e, dest); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		}
		return object.None, nil
	}})

	d.SetStr("addfile", &object.BuiltinFunc{Name: "addfile", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if !st.writeMode {
			return nil, object.Errorf(i.valueErr, "addfile on read-only TarFile")
		}
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "addfile() missing tarinfo")
		}
		tiInst, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "addfile() tarinfo must be TarInfo")
		}
		h := tarInfoToHeader(tiInst)
		var data []byte
		if len(args) >= 2 && args[1] != object.None {
			b, err := readAllFromFileLike(args[1])
			if err != nil {
				return nil, err
			}
			data = b
			if h.Size == 0 {
				h.Size = int64(len(data))
			}
		}
		if err := st.writer.WriteHeader(h); err != nil {
			return nil, object.Errorf(readErr, "%v", err)
		}
		if len(data) > 0 {
			if _, err := st.writer.Write(data); err != nil {
				return nil, object.Errorf(readErr, "%v", err)
			}
		}
		e := &tarEntry{header: h, data: data}
		st.entries = append(st.entries, e)
		st.byName[h.Name] = e
		return object.None, nil
	}})

	d.SetStr("add", &object.BuiltinFunc{Name: "add", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if !st.writeMode {
			return nil, object.Errorf(i.valueErr, "add on read-only TarFile")
		}
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "add() missing name")
		}
		fname, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "add() name must be str")
		}
		arcname := fname.V
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				arcname = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("arcname"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					arcname = s.V
				}
			}
		}
		recursive := true
		if len(args) >= 3 {
			recursive = object.Truthy(args[2])
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("recursive"); ok2 {
				recursive = object.Truthy(v)
			}
		}
		return object.None, st.addPath(fname.V, arcname, recursive)
	}})

	d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if st.closed {
			return object.None, nil
		}
		st.closed = true
		if st.writer != nil {
			if err := st.writer.Close(); err != nil {
				return nil, object.Errorf(readErr, "%v", err)
			}
			compressed, err := tarCompress(st.wbuf.Bytes(), st.compType)
			if err != nil {
				return nil, object.Errorf(readErr, "%v", err)
			}
			if err := os.WriteFile(st.filename, compressed, 0644); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		}
		return object.None, nil
	}})

	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closeFn, ok := d.GetStr("close"); ok {
			if bf, ok2 := closeFn.(*object.BuiltinFunc); ok2 {
				bf.Call(nil, nil, nil) //nolint:errcheck
			}
		}
		return object.False, nil
	}})

	d.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(st.entries) {
				return nil, false, nil
			}
			e := st.entries[idx]
			idx++
			return newTarInfoInstance(tarInfoCls, e.header), true, nil
		}}, nil
	}})

	d.SetStr("next", &object.BuiltinFunc{Name: "next", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil // convenience; our reader already parsed everything
	}})

	d.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────

func tarMemberName(o object.Object) (string, error) {
	switch v := o.(type) {
	case *object.Str:
		return v.V, nil
	case *object.Instance:
		if name, ok := v.Dict.GetStr("name"); ok {
			if s, ok2 := name.(*object.Str); ok2 {
				return s.V, nil
			}
		}
	}
	return "", object.Errorf(nil, "member must be str or TarInfo")
}

func tarDecompress(raw []byte, comp string) ([]byte, error) {
	switch comp {
	case "", "tar":
		return raw, nil
	case "gz":
		return gunzipBytes(raw)
	case "bz2":
		return bunzipBytes(raw)
	case "xz":
		return unxzBytes(raw)
	case "*":
		// Try each in turn.
		for _, fn := range []func([]byte) ([]byte, error){
			func(b []byte) ([]byte, error) {
				if _, err := tar.NewReader(bytes.NewReader(b)).Next(); err == nil {
					return b, nil
				}
				return nil, errTarUnknownFormat
			},
			gunzipBytes, bunzipBytes, unxzBytes,
		} {
			if out, err := fn(raw); err == nil {
				return out, nil
			}
		}
		return nil, errTarUnknownFormat
	}
	return nil, errTarUnknownFormat
}

func tarCompress(raw []byte, comp string) ([]byte, error) {
	switch comp {
	case "", "tar":
		return raw, nil
	case "gz":
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(raw); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "bz2":
		return bzip2.Encode(raw, 9), nil
	case "xz":
		return lzma.EncodeXZ(raw, lzma.CheckCRC64), nil
	}
	return nil, errTarUnknownFormat
}

func gunzipBytes(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func bunzipBytes(b []byte) ([]byte, error) {
	return io.ReadAll(stdbzip2.NewReader(bytes.NewReader(b)))
}

func unxzBytes(b []byte) ([]byte, error) {
	out, _, err := lzma.DecodeXZ(b)
	return out, err
}

var errTarUnknownFormat = &simpleErr{"unknown tar format"}

func tarExtractEntry(e *tarEntry, dest string) error {
	// Basic data-filter safety: reject absolute paths and .. escapes.
	name := e.header.Name
	if filepath.IsAbs(name) {
		return &simpleErr{"absolute path rejected: " + name}
	}
	cleaned := filepath.Clean(filepath.Join(dest, filepath.FromSlash(name)))
	absDest, _ := filepath.Abs(dest)
	absTarget, _ := filepath.Abs(cleaned)
	if absDest != "" && !strings.HasPrefix(absTarget, absDest) {
		return &simpleErr{"path escapes destination: " + name}
	}
	switch e.header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(cleaned, os.FileMode(e.header.Mode)|0700)
	case tar.TypeSymlink:
		_ = os.Remove(cleaned)
		if err := os.MkdirAll(filepath.Dir(cleaned), 0755); err != nil {
			return err
		}
		return os.Symlink(e.header.Linkname, cleaned)
	case tar.TypeReg, tar.TypeRegA:
		if err := os.MkdirAll(filepath.Dir(cleaned), 0755); err != nil {
			return err
		}
		mode := os.FileMode(e.header.Mode)
		if mode == 0 {
			mode = 0644
		}
		return os.WriteFile(cleaned, e.data, mode)
	}
	return nil
}

func (s *tarFileState) addPath(path, arcname string, recursive bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	h.Name = arcname
	if info.Mode()&os.ModeSymlink != 0 {
		link, lerr := os.Readlink(path)
		if lerr != nil {
			return lerr
		}
		h.Linkname = link
	}
	if err := s.writer.WriteHeader(h); err != nil {
		return err
	}
	var data []byte
	if info.Mode().IsRegular() {
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if _, werr := s.writer.Write(b); werr != nil {
			return werr
		}
		data = b
	}
	e := &tarEntry{header: h, data: data}
	s.entries = append(s.entries, e)
	s.byName[h.Name] = e

	if info.IsDir() && recursive {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, child := range entries {
			childPath := filepath.Join(path, child.Name())
			childArc := arcname + "/" + child.Name()
			if err := s.addPath(childPath, childArc, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// readAllFromFileLike drains a Python file-like object's read() method.
func readAllFromFileLike(o object.Object) ([]byte, error) {
	switch v := o.(type) {
	case *object.Bytes:
		return v.V, nil
	case *object.Str:
		return []byte(v.V), nil
	case *object.BytesIO:
		return append([]byte(nil), v.V...), nil
	case *object.Instance:
		if readFn, ok := v.Dict.GetStr("read"); ok {
			if bf, ok2 := readFn.(*object.BuiltinFunc); ok2 {
				res, err := bf.Call(nil, nil, nil)
				if err != nil {
					return nil, err
				}
				if b, ok3 := res.(*object.Bytes); ok3 {
					return b.V, nil
				}
				if s, ok3 := res.(*object.Str); ok3 {
					return []byte(s.V), nil
				}
			}
		}
	}
	return nil, object.Errorf(nil, "cannot read from fileobj")
}
