package vm

import (
	"bufio"
	"os"

	"github.com/tamnd/goipy/object"
)

// fileinputState holds all mutable state for one FileInput instance.
type fileinputState struct {
	files           []string
	fileIdx         int
	currentFile     *os.File
	scanner         *bufio.Scanner
	lineno          int
	filelineno      int
	filename        string
	isStdin         bool
	closed          bool
	nextFilePending bool
}

// openNextFile advances to the next file in the list and opens it.
func (s *fileinputState) openNextFile() {
	name := s.files[s.fileIdx]
	s.fileIdx++
	s.filelineno = 0
	s.filename = name
	s.isStdin = (name == "-")
	if s.isStdin {
		s.currentFile = nil
		s.scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(name)
		if err != nil {
			s.currentFile = nil
			s.scanner = nil
			return
		}
		s.currentFile = f
		s.scanner = bufio.NewScanner(f)
	}
}

// closeCurrentFile closes the currently open file (not stdin).
func (s *fileinputState) closeCurrentFile() {
	if s.currentFile != nil {
		s.currentFile.Close()
		s.currentFile = nil
	}
	s.scanner = nil
}

// nextLine advances the iterator and returns the next line with newline.
// Returns ("", false, nil) when exhausted.
func (s *fileinputState) nextLine() (object.Object, bool, error) {
	if s.closed {
		return nil, false, nil
	}
	for {
		if s.scanner != nil && !s.nextFilePending {
			if s.scanner.Scan() {
				s.lineno++
				s.filelineno++
				return &object.Str{V: s.scanner.Text() + "\n"}, true, nil
			}
		}
		// advance to next file
		s.nextFilePending = false
		s.closeCurrentFile()
		if s.fileIdx >= len(s.files) {
			return nil, false, nil
		}
		s.openNextFile()
	}
}

// fileinputFilenames normalises the files argument into a []string.
func fileinputFilenames(filesArg object.Object) []string {
	if filesArg == nil || filesArg == object.None {
		return []string{"-"}
	}
	switch v := filesArg.(type) {
	case *object.Str:
		return []string{v.V}
	case *object.List:
		out := make([]string, 0, len(v.V))
		for _, el := range v.V {
			if s, ok := el.(*object.Str); ok {
				out = append(out, s.V)
			}
		}
		if len(out) == 0 {
			return []string{"-"}
		}
		return out
	case *object.Tuple:
		out := make([]string, 0, len(v.V))
		for _, el := range v.V {
			if s, ok := el.(*object.Str); ok {
				out = append(out, s.V)
			}
		}
		if len(out) == 0 {
			return []string{"-"}
		}
		return out
	}
	return []string{"-"}
}

// buildFileinputInstance creates a new FileInput *object.Instance with all
// per-instance methods installed as closures over state.
func (i *Interp) buildFileinputInstance(cls *object.Class, filesArg object.Object) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	state := &fileinputState{
		files: fileinputFilenames(filesArg),
	}

	// __iter__ returns an *object.Iter that reads from state.
	// This makes `for line in fi` work via GET_ITER dispatch.
	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Iter{Next: func() (object.Object, bool, error) {
			return state.nextLine()
		}}, nil
	}})

	// __next__ is for next(fi) direct calls.
	inst.Dict.SetStr("__next__", &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		v, ok, err := state.nextLine()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.NewException(i.stopIter, "")
		}
		return v, nil
	}})

	// __enter__ returns self.
	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})

	// __exit__ closes the FileInput.
	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		state.closed = true
		state.closeCurrentFile()
		return object.False, nil
	}})

	// filename() → current filename or None.
	inst.Dict.SetStr("filename", &object.BuiltinFunc{Name: "filename", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if state.filename == "" {
			return object.None, nil
		}
		return &object.Str{V: state.filename}, nil
	}})

	// fileno() → current fd or -1.
	inst.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if state.currentFile != nil {
			return object.NewInt(int64(state.currentFile.Fd())), nil
		}
		return object.NewInt(-1), nil
	}})

	// lineno() → cumulative line number.
	inst.Dict.SetStr("lineno", &object.BuiltinFunc{Name: "lineno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(state.lineno)), nil
	}})

	// filelineno() → line number within current file.
	inst.Dict.SetStr("filelineno", &object.BuiltinFunc{Name: "filelineno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(state.filelineno)), nil
	}})

	// isfirstline() → filelineno() == 1.
	inst.Dict.SetStr("isfirstline", &object.BuiltinFunc{Name: "isfirstline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(state.filelineno == 1), nil
	}})

	// isstdin() → reading from stdin.
	inst.Dict.SetStr("isstdin", &object.BuiltinFunc{Name: "isstdin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(state.isStdin), nil
	}})

	// nextfile() → skip to next file.
	inst.Dict.SetStr("nextfile", &object.BuiltinFunc{Name: "nextfile", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		state.nextFilePending = true
		return object.None, nil
	}})

	// close() → close all files.
	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		state.closed = true
		state.closeCurrentFile()
		return object.None, nil
	}})

	return inst
}

// buildFileinput constructs the fileinput module.
func (i *Interp) buildFileinput() *object.Module {
	m := &object.Module{Name: "fileinput", Dict: object.NewDict()}

	// FileInput class — __init__ delegates to buildFileinputInstance logic.
	cls := &object.Class{Name: "FileInput", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		// Resolve files argument: positional arg[1] or keyword "files".
		var filesArg object.Object = object.None
		if len(a) >= 2 {
			filesArg = a[1]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("files"); ok2 {
				filesArg = v
			}
		}

		// Build a temporary instance to extract the closures, then copy them
		// into self. This lets us reuse buildFileinputInstance cleanly.
		tmp := i.buildFileinputInstance(cls, filesArg)
		keys, vals := tmp.Dict.Items()
		for k, key := range keys {
			if ks, ok2 := key.(*object.Str); ok2 {
				self.Dict.SetStr(ks.V, vals[k])
			}
		}
		return object.None, nil
	}})

	m.Dict.SetStr("FileInput", cls)

	// Module-level global instance (set by input()).
	var globalInst *object.Instance

	// input(files=None, ...) → FileInput
	m.Dict.SetStr("input", &object.BuiltinFunc{Name: "input", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var filesArg object.Object = object.None
		if len(a) >= 1 {
			filesArg = a[0]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("files"); ok2 {
				filesArg = v
			}
		}
		inst := i.buildFileinputInstance(cls, filesArg)
		globalInst = inst
		return inst, nil
	}})

	// Helper to call a zero-arg method on globalInst.
	callGlobal := func(name string) (object.Object, error) {
		if globalInst == nil {
			return object.None, nil
		}
		if fn, ok := globalInst.Dict.GetStr(name); ok {
			return i.callObject(fn, nil, nil)
		}
		return object.None, nil
	}

	m.Dict.SetStr("filename", &object.BuiltinFunc{Name: "filename", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("filename")
	}})
	m.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("fileno")
	}})
	m.Dict.SetStr("lineno", &object.BuiltinFunc{Name: "lineno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("lineno")
	}})
	m.Dict.SetStr("filelineno", &object.BuiltinFunc{Name: "filelineno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("filelineno")
	}})
	m.Dict.SetStr("isfirstline", &object.BuiltinFunc{Name: "isfirstline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("isfirstline")
	}})
	m.Dict.SetStr("isstdin", &object.BuiltinFunc{Name: "isstdin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("isstdin")
	}})
	m.Dict.SetStr("nextfile", &object.BuiltinFunc{Name: "nextfile", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callGlobal("nextfile")
	}})
	m.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		v, err := callGlobal("close")
		if err == nil {
			globalInst = nil
		}
		return v, err
	}})

	// hook_encoded(encoding, errors=None) → callable that opens with encoding.
	m.Dict.SetStr("hook_encoded", &object.BuiltinFunc{Name: "hook_encoded", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		encoding := "utf-8"
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				encoding = s.V
			}
		}
		_ = encoding
		hook := &object.BuiltinFunc{Name: "hook_encoded_opener", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return object.None, nil
			}
			filename := ""
			if s, ok := args[0].(*object.Str); ok {
				filename = s.V
			}
			f, err := os.Open(filename)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return &object.File{F: f}, nil
		}}
		return hook, nil
	}})

	// hook_compressed(filename, mode) → open file (stub: just uses os.Open).
	m.Dict.SetStr("hook_compressed", &object.BuiltinFunc{Name: "hook_compressed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		filename := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				filename = s.V
			}
		}
		f, err := os.Open(filename)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.File{F: f}, nil
	}})

	return m
}
