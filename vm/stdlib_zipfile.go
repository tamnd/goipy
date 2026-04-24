package vm

import (
	"archive/zip"
	"bytes"
	stdbzip2 "compress/bzip2"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/goipy/lib/bzip2"
	"github.com/tamnd/goipy/lib/lzma"
	"github.com/tamnd/goipy/object"
)

// Python `zipfile` stdlib module.
//
// Read and write ZIP archives. STORED and DEFLATE are handled by Go's
// archive/zip. BZIP2 and LZMA entries are compressed through our
// lib/bzip2 / lib/lzma packages and stored as raw-data entries with
// the correct compress_type stamp.

const (
	zipStored   = 0
	zipDeflated = 8
	zipBzip2    = 12
	zipLzma     = 14
)

func (i *Interp) buildZipfile() *object.Module {
	m := &object.Module{Name: "zipfile", Dict: object.NewDict()}

	badZipCls := &object.Class{Name: "BadZipFile", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	largeZipCls := &object.Class{Name: "LargeZipFile", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	m.Dict.SetStr("BadZipFile", badZipCls)
	m.Dict.SetStr("BadZipfile", badZipCls) // CPython alias
	m.Dict.SetStr("LargeZipFile", largeZipCls)
	m.Dict.SetStr("error", badZipCls)

	for name, v := range map[string]int64{
		"ZIP_STORED":          zipStored,
		"ZIP_DEFLATED":        zipDeflated,
		"ZIP_BZIP2":           zipBzip2,
		"ZIP_LZMA":            zipLzma,
		"ZIP64_LIMIT":         (1 << 31) - 1,
		"ZIP_FILECOUNT_LIMIT": 0xFFFF,
		"ZIP_MAX_COMMENT":     0xFFFF,
	} {
		m.Dict.SetStr(name, object.NewInt(v))
	}

	zipInfoCls := i.buildZipInfoClass()
	m.Dict.SetStr("ZipInfo", zipInfoCls)

	zipFileCls := i.buildZipFileClass(badZipCls, zipInfoCls)
	m.Dict.SetStr("ZipFile", zipFileCls)

	m.Dict.SetStr("Path", i.buildZipPathClass(zipFileCls))

	m.Dict.SetStr("is_zipfile", &object.BuiltinFunc{Name: "is_zipfile", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
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
		if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
			return object.False, nil
		}
		return object.True, nil
	}})

	return m
}

// ── ZipInfo ───────────────────────────────────────────────────────────────

func (i *Interp) buildZipInfoClass() *object.Class {
	cls := &object.Class{Name: "ZipInfo", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "ZipInfo() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZipInfo() self must be instance")
		}
		name := "NoName"
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				name = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("filename"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					name = s.V
				}
			}
		}
		dt := []object.Object{
			object.NewInt(1980), object.NewInt(1), object.NewInt(1),
			object.NewInt(0), object.NewInt(0), object.NewInt(0),
		}
		if len(args) >= 3 {
			if t, ok2 := args[2].(*object.Tuple); ok2 && len(t.V) == 6 {
				dt = t.V
			}
		}
		return nil, initZipInfo(self, name, dt, 0, 0, 0, 0, "", nil)
	}})
	return cls
}

func initZipInfo(inst *object.Instance, name string, dt []object.Object, crc uint32, compSize, fileSize uint64, compType int, comment string, extra []byte) error {
	d := inst.Dict
	d.SetStr("filename", &object.Str{V: name})
	d.SetStr("orig_filename", &object.Str{V: name})
	d.SetStr("date_time", &object.Tuple{V: dt})
	d.SetStr("compress_type", object.NewInt(int64(compType)))
	d.SetStr("compress_size", object.NewInt(int64(compSize)))
	d.SetStr("file_size", object.NewInt(int64(fileSize)))
	d.SetStr("CRC", object.NewInt(int64(crc)))
	d.SetStr("comment", &object.Bytes{V: []byte(comment)})
	if extra == nil {
		extra = []byte{}
	}
	d.SetStr("extra", &object.Bytes{V: extra})
	d.SetStr("header_offset", object.NewInt(0))
	d.SetStr("external_attr", object.NewInt(0))
	d.SetStr("internal_attr", object.NewInt(0))
	d.SetStr("create_system", object.NewInt(3))
	d.SetStr("create_version", object.NewInt(20))
	d.SetStr("extract_version", object.NewInt(20))
	d.SetStr("flag_bits", object.NewInt(0))
	d.SetStr("volume", object.NewInt(0))
	d.SetStr("reserved", object.NewInt(0))

	isDir := strings.HasSuffix(name, "/")
	d.SetStr("is_dir", &object.BuiltinFunc{Name: "is_dir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if isDir {
			return object.True, nil
		}
		return object.False, nil
	}})

	return nil
}

// newZipInfoInstance creates a ZipInfo populated from Go-side metadata.
func newZipInfoInstance(cls *object.Class, h *zip.FileHeader) *object.Instance {
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}
	t := h.Modified
	if t.IsZero() {
		t = h.ModTime()
	}
	dt := []object.Object{
		object.NewInt(int64(t.Year())),
		object.NewInt(int64(t.Month())),
		object.NewInt(int64(t.Day())),
		object.NewInt(int64(t.Hour())),
		object.NewInt(int64(t.Minute())),
		object.NewInt(int64(t.Second())),
	}
	compType := int(h.Method)
	initZipInfo(inst, h.Name, dt, h.CRC32, h.CompressedSize64, h.UncompressedSize64, compType, h.Comment, h.Extra)
	return inst
}

// ── ZipFile ───────────────────────────────────────────────────────────────

func (i *Interp) buildZipFileClass(badZipCls, zipInfoCls *object.Class) *object.Class {
	cls := &object.Class{Name: "ZipFile", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "ZipFile() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZipFile() self must be instance")
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "ZipFile() requires file argument")
		}
		filename := ""
		switch f := args[1].(type) {
		case *object.Str:
			filename = f.V
		default:
			return nil, object.Errorf(i.typeErr, "ZipFile() file must be str")
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
		compression := zipStored
		if len(args) >= 4 {
			if n, ok2 := toInt64(args[3]); ok2 {
				compression = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compression"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					compression = int(n)
				}
			}
		}
		return nil, i.initZipFileInstance(self, filename, mode, compression, badZipCls, zipInfoCls)
	}})
	return cls
}

// zipFileState holds the Go-side backing for an open ZipFile instance.
type zipFileState struct {
	filename    string
	mode        string
	compression int
	closed      bool

	// Read side: full archive bytes + parsed reader.
	raw    []byte
	reader *zip.Reader

	// Write side: in-memory writer that flushes to disk on close.
	wbuf           *bytes.Buffer
	writer         *zip.Writer
	nameSet        map[string]struct{}
	comment        []byte
	origRead       []*zip.File        // preserved entries for append mode (source for reads)
	pendingHeaders []*zip.FileHeader  // headers for new writes (order preserved)
	pendingReads   map[string][]byte  // uncompressed payload for new writes
	entryComments  map[string]string
}

func (i *Interp) initZipFileInstance(inst *object.Instance, filename, mode string, compression int, badZipCls, zipInfoCls *object.Class) error {
	st := &zipFileState{
		filename:      filename,
		mode:          mode,
		compression:   compression,
		nameSet:       map[string]struct{}{},
		pendingReads:  map[string][]byte{},
		entryComments: map[string]string{},
	}
	d := inst.Dict

	d.SetStr("filename", &object.Str{V: filename})
	d.SetStr("mode", &object.Str{V: mode})
	d.SetStr("compression", object.NewInt(int64(compression)))
	d.SetStr("comment", &object.Bytes{V: []byte{}})
	d.SetStr("debug", object.NewInt(0))
	d.SetStr("pwd", object.None)
	d.SetStr("NameToInfo", object.NewDict())

	switch mode {
	case "r":
		raw, err := os.ReadFile(filename)
		if err != nil {
			return object.Errorf(i.osErr, "%v", err)
		}
		zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			return object.Errorf(badZipCls, "%v", err)
		}
		st.raw = raw
		st.reader = zr
		if zr.Comment != "" {
			st.comment = []byte(zr.Comment)
			d.SetStr("comment", &object.Bytes{V: st.comment})
		}
	case "w", "x":
		st.wbuf = &bytes.Buffer{}
		st.writer = zip.NewWriter(st.wbuf)
	case "a":
		raw, err := os.ReadFile(filename)
		if err == nil {
			if zr, zerr := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); zerr == nil {
				st.origRead = zr.File
				for _, f := range zr.File {
					st.nameSet[f.Name] = struct{}{}
				}
			}
		}
		st.wbuf = &bytes.Buffer{}
		st.writer = zip.NewWriter(st.wbuf)
		// Copy existing entries verbatim.
		for _, f := range st.origRead {
			if err := st.writer.Copy(f); err != nil {
				return object.Errorf(badZipCls, "%v", err)
			}
		}
	default:
		return object.Errorf(i.valueErr, "ZipFile mode must be 'r', 'w', 'x', or 'a'")
	}

	d.SetStr("namelist", &object.BuiltinFunc{Name: "namelist", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: st.allNames()}, nil
	}})

	d.SetStr("infolist", &object.BuiltinFunc{Name: "infolist", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var out []object.Object
		for _, h := range st.allHeaders() {
			out = append(out, newZipInfoInstance(zipInfoCls, h))
		}
		return &object.List{V: out}, nil
	}})

	d.SetStr("getinfo", &object.BuiltinFunc{Name: "getinfo", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "getinfo() missing name")
		}
		name, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getinfo() name must be str")
		}
		for _, h := range st.allHeaders() {
			if h.Name == name.V {
				return newZipInfoInstance(zipInfoCls, h), nil
			}
		}
		return nil, object.Errorf(i.keyErr, "there is no item named %q in the archive", name.V)
	}})

	d.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "read() missing name")
		}
		name, err := zipMemberName(args[0])
		if err != nil {
			return nil, err
		}
		data, err := st.readMember(name)
		if err != nil {
			return nil, object.Errorf(badZipCls, "%v", err)
		}
		return &object.Bytes{V: data}, nil
	}})

	d.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "open() missing name")
		}
		name, err := zipMemberName(args[0])
		if err != nil {
			return nil, err
		}
		data, err := st.readMember(name)
		if err != nil {
			return nil, object.Errorf(badZipCls, "%v", err)
		}
		return i.makeBytesIOLike(data), nil
	}})

	d.SetStr("extract", &object.BuiltinFunc{Name: "extract", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "extract() missing name")
		}
		name, err := zipMemberName(args[0])
		if err != nil {
			return nil, err
		}
		dest := "."
		if len(args) >= 2 {
			if s, ok := args[1].(*object.Str); ok {
				dest = s.V
			}
		}
		p, err := st.extractMember(name, dest)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: p}, nil
	}})

	d.SetStr("extractall", &object.BuiltinFunc{Name: "extractall", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		dest := "."
		if len(args) >= 1 {
			if s, ok := args[0].(*object.Str); ok {
				dest = s.V
			}
		}
		for _, h := range st.allHeaders() {
			if _, err := st.extractMember(h.Name, dest); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		}
		return object.None, nil
	}})

	d.SetStr("writestr", &object.BuiltinFunc{Name: "writestr", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if mode == "r" {
			return nil, object.Errorf(i.valueErr, "writestr on read-only ZipFile")
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "writestr() requires zinfo_or_arcname and data")
		}
		var arcname string
		var userMethod = -1
		var dateTime time.Time
		switch z := args[0].(type) {
		case *object.Str:
			arcname = z.V
		case *object.Instance:
			if v, ok := z.Dict.GetStr("filename"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					arcname = s.V
				}
			}
			if v, ok := z.Dict.GetStr("compress_type"); ok {
				if n, ok2 := toInt64(v); ok2 {
					userMethod = int(n)
				}
			}
			if v, ok := z.Dict.GetStr("date_time"); ok {
				if t, ok2 := v.(*object.Tuple); ok2 && len(t.V) == 6 {
					parts := make([]int, 6)
					for i, o := range t.V {
						if n, ok := toInt64(o); ok {
							parts[i] = int(n)
						}
					}
					dateTime = time.Date(parts[0], time.Month(parts[1]), parts[2], parts[3], parts[4], parts[5], 0, time.UTC)
				}
			}
		default:
			return nil, object.Errorf(i.typeErr, "writestr() zinfo_or_arcname must be str or ZipInfo")
		}
		var data []byte
		switch d := args[1].(type) {
		case *object.Bytes:
			data = d.V
		case *object.Str:
			data = []byte(d.V)
		default:
			b, err := asBytes(args[1])
			if err != nil {
				return nil, err
			}
			data = b
		}
		method := compression
		if userMethod >= 0 {
			method = userMethod
		}
		if len(args) >= 3 {
			if n, ok := toInt64(args[2]); ok {
				method = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("compress_type"); ok {
				if n, ok2 := toInt64(v); ok2 {
					method = int(n)
				}
			}
		}
		if err := st.writeMember(arcname, data, method, dateTime); err != nil {
			return nil, object.Errorf(badZipCls, "%v", err)
		}
		return object.None, nil
	}})

	d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if mode == "r" {
			return nil, object.Errorf(i.valueErr, "write on read-only ZipFile")
		}
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "write() missing filename")
		}
		filenameArg, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "write() filename must be str")
		}
		data, err := os.ReadFile(filenameArg.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		arcname := filenameArg.V
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
		method := compression
		if len(args) >= 3 {
			if n, ok2 := toInt64(args[2]); ok2 {
				method = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compress_type"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					method = int(n)
				}
			}
		}
		info, err := os.Stat(filenameArg.V)
		var modTime time.Time
		if err == nil {
			modTime = info.ModTime()
		}
		if err := st.writeMember(arcname, data, method, modTime); err != nil {
			return nil, object.Errorf(badZipCls, "%v", err)
		}
		return object.None, nil
	}})

	d.SetStr("testzip", &object.BuiltinFunc{Name: "testzip", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for _, h := range st.allHeaders() {
			if _, err := st.readMember(h.Name); err != nil {
				return &object.Str{V: h.Name}, nil
			}
		}
		return object.None, nil
	}})

	d.SetStr("setpassword", &object.BuiltinFunc{Name: "setpassword", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) > 0 {
			d.SetStr("pwd", args[0])
		}
		return object.None, nil
	}})

	d.SetStr("printdir", &object.BuiltinFunc{Name: "printdir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if st.closed {
			return object.None, nil
		}
		st.closed = true
		if st.writer != nil {
			if len(st.comment) > 0 {
				st.writer.SetComment(string(st.comment))
			}
			if err := st.writer.Close(); err != nil {
				return nil, object.Errorf(badZipCls, "%v", err)
			}
			if err := os.WriteFile(st.filename, st.wbuf.Bytes(), 0644); err != nil {
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
		names := st.allNames()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(names) {
				return nil, false, nil
			}
			v := names[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	d.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return object.False, nil
		}
		s, ok := args[0].(*object.Str)
		if !ok {
			return object.False, nil
		}
		for _, n := range st.allNames() {
			if sn, ok := n.(*object.Str); ok && sn.V == s.V {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	// comment setter
	d.SetStr("set_comment", &object.BuiltinFunc{Name: "set_comment", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return object.None, nil
		}
		b, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		st.comment = b
		d.SetStr("comment", &object.Bytes{V: b})
		return object.None, nil
	}})

	return nil
}

// allHeaders returns the union of read-side + pending-write headers, in
// insertion order (original entries first, then new pending writes).
func (s *zipFileState) allHeaders() []*zip.FileHeader {
	var out []*zip.FileHeader
	if s.reader != nil {
		for _, f := range s.reader.File {
			h := f.FileHeader
			out = append(out, &h)
		}
	}
	for _, f := range s.origRead {
		// Skip entries that were re-written (same name appears in pendingHeaders).
		if _, replaced := s.pendingReads[f.Name]; replaced {
			continue
		}
		h := f.FileHeader
		out = append(out, &h)
	}
	out = append(out, s.pendingHeaders...)
	return out
}

func (s *zipFileState) allNames() []object.Object {
	var names []object.Object
	for _, h := range s.allHeaders() {
		names = append(names, &object.Str{V: h.Name})
	}
	return names
}

func (s *zipFileState) readMember(name string) ([]byte, error) {
	if data, ok := s.pendingReads[name]; ok {
		return append([]byte(nil), data...), nil
	}
	if s.reader != nil {
		for _, f := range s.reader.File {
			if f.Name == name {
				return readZipFile(f)
			}
		}
	}
	for _, f := range s.origRead {
		if f.Name == name {
			return readZipFile(f)
		}
	}
	return nil, errZipNoSuchEntry
}

// ── helpers ───────────────────────────────────────────────────────────────

var errZipNoSuchEntry = &simpleErr{"no such entry"}

func readZipFile(f *zip.File) ([]byte, error) {
	switch f.Method {
	case zipStored, zipDeflated:
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	case zipBzip2:
		r, err := f.OpenRaw()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return io.ReadAll(stdbzip2.NewReader(bytes.NewReader(raw)))
	case zipLzma:
		r, err := f.OpenRaw()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if len(raw) < 9 {
			return nil, errZipLZMARead
		}
		// raw layout: [0..2)=version [2..4)=propsSize [4..9)=props+dict
		// [9..]=stream. Re-frame as ALONE: props(1)+dict(4)+uSize(8 FF)
		// + stream, so our decoder can take it.
		alone := make([]byte, 0, 13+len(raw)-9)
		alone = append(alone, raw[4:9]...)
		var sz [8]byte
		for k := 0; k < 8; k++ {
			sz[k] = byte(f.UncompressedSize64 >> (8 * uint(k)))
		}
		alone = append(alone, sz[:]...)
		alone = append(alone, raw[9:]...)
		return lzma.DecodeAlone(alone)
	default:
		return nil, errZipUnsupportedMethod
	}
}

var (
	errZipLZMARead          = &simpleErr{"LZMA zip entries not supported for reading"}
	errZipUnsupportedMethod = &simpleErr{"unsupported compression method"}
)

func (s *zipFileState) extractMember(name, dest string) (string, error) {
	data, err := s.readMember(name)
	if err != nil {
		return "", err
	}
	target := filepath.Join(dest, filepath.FromSlash(name))
	if strings.HasSuffix(name, "/") {
		if err := os.MkdirAll(target, 0755); err != nil {
			return "", err
		}
		return target, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", err
	}
	return target, nil
}

func (s *zipFileState) writeMember(name string, data []byte, method int, modTime time.Time) error {
	s.nameSet[name] = struct{}{}
	if s.pendingReads == nil {
		s.pendingReads = map[string][]byte{}
	}
	s.pendingReads[name] = append([]byte(nil), data...)

	h := &zip.FileHeader{Name: name}
	if !modTime.IsZero() {
		h.Modified = modTime
	} else {
		h.Modified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	hdrCopy := *h
	hdrCopy.Method = uint16(method)
	hdrCopy.CRC32 = crc32.ChecksumIEEE(data)
	hdrCopy.UncompressedSize64 = uint64(len(data))
	s.pendingHeaders = append(s.pendingHeaders, &hdrCopy)
	switch method {
	case zipStored, zipDeflated:
		h.Method = uint16(method)
		w, err := s.writer.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		return nil
	case zipBzip2:
		compressed := bzip2.Encode(data, 9)
		return s.writeStoredRaw(h, data, compressed, uint16(method))
	case zipLzma:
		compressed := lzmaZipEntry(data)
		return s.writeStoredRaw(h, data, compressed, uint16(method))
	default:
		return errZipUnsupportedMethod
	}
}

// writeStoredRaw writes an entry whose compressed bytes we already
// produced out-of-band. archive/zip offers CreateRaw for this.
func (s *zipFileState) writeStoredRaw(h *zip.FileHeader, uncompressed, compressed []byte, method uint16) error {
	h.Method = method
	h.CRC32 = crc32.ChecksumIEEE(uncompressed)
	h.UncompressedSize64 = uint64(len(uncompressed))
	h.CompressedSize64 = uint64(len(compressed))
	w, err := s.writer.CreateRaw(h)
	if err != nil {
		return err
	}
	_, err = w.Write(compressed)
	return err
}

// lzmaZipEntry wraps a payload in the per-entry LZMA framing that zip
// uses: 2 bytes version (currently 0x10 0x02), 2 bytes props-size (LE),
// then 5 bytes raw LZMA properties + dict-size, then the raw stream.
// Our lib/lzma's FORMAT_ALONE produces 13-byte header + stream; the
// ZIP entry header takes 4+5=9 bytes before the stream, so we strip
// the 13-byte ALONE header's uncompressed-size field (last 8 bytes)
// and emit our own 4-byte prefix.
func lzmaZipEntry(data []byte) []byte {
	alone := lzma.EncodeAlone(data)
	// alone layout: [0]=props [1..5)=dictSize [5..13)=uSize [13..]=stream
	props := alone[0]
	dict := alone[1:5]
	stream := alone[13:]
	var out []byte
	out = append(out, 0x10, 0x02, 0x05, 0x00)
	out = append(out, props)
	out = append(out, dict...)
	out = append(out, stream...)
	return out
}

func zipMemberName(o object.Object) (string, error) {
	switch v := o.(type) {
	case *object.Str:
		return v.V, nil
	case *object.Instance:
		if name, ok := v.Dict.GetStr("filename"); ok {
			if s, ok2 := name.(*object.Str); ok2 {
				return s.V, nil
			}
		}
	}
	return "", object.Errorf(nil, "member must be str or ZipInfo")
}

// makeBytesIOLike produces a minimal file-like object exposing read,
// readline, readlines, close, __enter__, __exit__ over a bytes buffer.
func (i *Interp) makeBytesIOLike(data []byte) *object.Instance {
	cls := &object.Class{Name: "ZipExtFile", Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}
	var pos int

	d.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		size := -1
		if len(args) >= 1 {
			if n, ok := toInt64(args[0]); ok {
				size = int(n)
			}
		}
		remaining := data[pos:]
		if size < 0 || size >= len(remaining) {
			pos = len(data)
			return &object.Bytes{V: append([]byte{}, remaining...)}, nil
		}
		chunk := remaining[:size]
		pos += size
		return &object.Bytes{V: append([]byte{}, chunk...)}, nil
	}})
	d.SetStr("readline", &object.BuiltinFunc{Name: "readline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		remaining := data[pos:]
		if len(remaining) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		nl := bytes.IndexByte(remaining, '\n')
		var line []byte
		if nl < 0 {
			line = append([]byte{}, remaining...)
			pos = len(data)
		} else {
			line = append([]byte{}, remaining[:nl+1]...)
			pos += nl + 1
		}
		return &object.Bytes{V: line}, nil
	}})
	d.SetStr("readlines", &object.BuiltinFunc{Name: "readlines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var lines []object.Object
		for {
			remaining := data[pos:]
			if len(remaining) == 0 {
				break
			}
			nl := bytes.IndexByte(remaining, '\n')
			var line []byte
			if nl < 0 {
				line = append([]byte{}, remaining...)
				pos = len(data)
			} else {
				line = append([]byte{}, remaining[:nl+1]...)
				pos += nl + 1
			}
			lines = append(lines, &object.Bytes{V: line})
		}
		return &object.List{V: lines}, nil
	}})
	d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
	return inst
}

// ── Path ──────────────────────────────────────────────────────────────────

func (i *Interp) buildZipPathClass(zipFileCls *object.Class) *object.Class {
	cls := &object.Class{Name: "Path", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "Path() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "Path() self must be instance")
		}
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "Path() requires root")
		}
		root := args[1]
		at := ""
		if len(args) >= 3 {
			if s, ok2 := args[2].(*object.Str); ok2 {
				at = s.V
			}
		}
		return nil, initZipPath(self, root, at)
	}})
	return cls
}

func initZipPath(inst *object.Instance, root object.Object, at string) error {
	d := inst.Dict
	d.SetStr("root", root)
	d.SetStr("at", &object.Str{V: at})
	name := at
	if idx := strings.LastIndex(strings.TrimSuffix(at, "/"), "/"); idx >= 0 {
		name = strings.TrimSuffix(at, "/")[idx+1:]
	} else {
		name = strings.TrimSuffix(at, "/")
	}
	d.SetStr("name", &object.Str{V: name})

	readBytes := func() ([]byte, error) {
		inst, ok := root.(*object.Instance)
		if !ok {
			return nil, errZipNoSuchEntry
		}
		readFn, ok := inst.Dict.GetStr("read")
		if !ok {
			return nil, errZipNoSuchEntry
		}
		bf, ok := readFn.(*object.BuiltinFunc)
		if !ok {
			return nil, errZipNoSuchEntry
		}
		res, err := bf.Call(nil, []object.Object{&object.Str{V: at}}, nil)
		if err != nil {
			return nil, err
		}
		if b, ok := res.(*object.Bytes); ok {
			return b.V, nil
		}
		return nil, errZipNoSuchEntry
	}

	d.SetStr("read_bytes", &object.BuiltinFunc{Name: "read_bytes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		b, err := readBytes()
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: b}, nil
	}})
	d.SetStr("read_text", &object.BuiltinFunc{Name: "read_text", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		b, err := readBytes()
		if err != nil {
			return nil, err
		}
		return &object.Str{V: string(b)}, nil
	}})
	d.SetStr("is_file", &object.BuiltinFunc{Name: "is_file", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if strings.HasSuffix(at, "/") || at == "" {
			return object.False, nil
		}
		return object.True, nil
	}})
	d.SetStr("is_dir", &object.BuiltinFunc{Name: "is_dir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if at == "" || strings.HasSuffix(at, "/") {
			return object.True, nil
		}
		return object.False, nil
	}})
	d.SetStr("exists", &object.BuiltinFunc{Name: "exists", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if at == "" {
			return object.True, nil
		}
		if _, err := readBytes(); err != nil {
			if strings.HasSuffix(at, "/") {
				return object.True, nil
			}
			return object.False, nil
		}
		return object.True, nil
	}})
	d.SetStr("iterdir", &object.BuiltinFunc{Name: "iterdir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: nil}, nil
	}})
	return nil
}
