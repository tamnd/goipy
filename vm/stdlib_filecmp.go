package vm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildFilecmp() *object.Module {
	m := &object.Module{Name: "filecmp", Dict: object.NewDict()}

	m.Dict.SetStr("BUFSIZE", object.NewInt(8192))

	defaultIgnores := &object.List{V: []object.Object{
		&object.Str{V: "RCS"},
		&object.Str{V: "CVS"},
		&object.Str{V: "tags"},
		&object.Str{V: ".git"},
		&object.Str{V: ".hg"},
		&object.Str{V: ".bzr"},
		&object.Str{V: "_darcs"},
		&object.Str{V: "__pycache__"},
	}}
	m.Dict.SetStr("DEFAULT_IGNORES", defaultIgnores)

	// filecmpSig is the (type, size, mtime) signature used for shallow compare.
	type filecmpSig struct {
		fileType int64
		size     int64
		mtime    float64
	}
	type cacheKey struct {
		f1, f2                 string
		t1, t2                 int64
		sz1, sz2               int64
		mt1, mt2               float64
	}
	cache := make(map[cacheKey]bool)

	fileSig := func(path string) (filecmpSig, error) {
		info, err := os.Stat(path)
		if err != nil {
			return filecmpSig{}, err
		}
		ft := goModeToPosix(info.Mode()) & 0o170000
		return filecmpSig{
			fileType: ft,
			size:     info.Size(),
			mtime:    float64(info.ModTime().UnixNano()) / 1e9,
		}, nil
	}

	doCmp := func(f1, f2 string) bool {
		fp1, err := os.Open(f1)
		if err != nil {
			return false
		}
		defer fp1.Close()
		fp2, err := os.Open(f2)
		if err != nil {
			return false
		}
		defer fp2.Close()
		buf1 := make([]byte, 8192)
		buf2 := make([]byte, 8192)
		for {
			n1, e1 := io.ReadFull(fp1, buf1)
			n2, e2 := io.ReadFull(fp2, buf2)
			if !bytes.Equal(buf1[:n1], buf2[:n2]) {
				return false
			}
			eof1 := e1 == io.EOF || e1 == io.ErrUnexpectedEOF
			eof2 := e2 == io.EOF || e2 == io.ErrUnexpectedEOF
			if eof1 && eof2 {
				return true
			}
			if e1 != nil || e2 != nil {
				return false
			}
		}
	}

	// clear_cache()
	m.Dict.SetStr("clear_cache", &object.BuiltinFunc{Name: "clear_cache", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for k := range cache {
			delete(cache, k)
		}
		return object.None, nil
	}})

	// cmp(f1, f2, shallow=True) — raises OSError for non-existent files (matches CPython).
	cmpFunc := &object.BuiltinFunc{Name: "cmp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "cmp() requires at least 2 arguments")
		}
		f1s, ok1 := a[0].(*object.Str)
		f2s, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "cmp() arguments must be str")
		}
		shallow := true
		if len(a) >= 3 {
			shallow = object.Truthy(a[2])
		}
		if kw != nil {
			if v, ok := kw.GetStr("shallow"); ok {
				shallow = object.Truthy(v)
			}
		}
		f1, f2 := f1s.V, f2s.V

		s1, err := fileSig(f1)
		if err != nil {
			return nil, object.Errorf(i.osErr, "[Errno 2] No such file or directory: '%s'", f1)
		}
		s2, err := fileSig(f2)
		if err != nil {
			return nil, object.Errorf(i.osErr, "[Errno 2] No such file or directory: '%s'", f2)
		}

		// Non-regular files → False
		const ifreg = int64(0o100000)
		if s1.fileType != ifreg || s2.fileType != ifreg {
			return object.False, nil
		}

		// Shallow + identical sig → True without reading
		if shallow && s1 == s2 {
			return object.True, nil
		}

		// Different sizes → False
		if s1.size != s2.size {
			return object.False, nil
		}

		// Check cache
		key := cacheKey{f1, f2, s1.fileType, s2.fileType, s1.size, s2.size, s1.mtime, s2.mtime}
		if outcome, ok := cache[key]; ok {
			return object.BoolOf(outcome), nil
		}

		outcome := doCmp(f1, f2)
		if len(cache) > 100 {
			for k := range cache {
				delete(cache, k)
			}
		}
		cache[key] = outcome
		return object.BoolOf(outcome), nil
	}}
	m.Dict.SetStr("cmp", cmpFunc)

	// cmpfiles(a, b, common, shallow=True) → (match, mismatch, errors)
	m.Dict.SetStr("cmpfiles", &object.BuiltinFunc{Name: "cmpfiles", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "cmpfiles() requires 3 arguments")
		}
		dirA, ok1 := a[0].(*object.Str)
		dirB, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "cmpfiles() dir arguments must be str")
		}
		shallow := true
		if len(a) >= 4 {
			shallow = object.Truthy(a[3])
		}
		if kw != nil {
			if v, ok := kw.GetStr("shallow"); ok {
				shallow = object.Truthy(v)
			}
		}

		var common []string
		switch v := a[2].(type) {
		case *object.List:
			for _, item := range v.V {
				if s, ok := item.(*object.Str); ok {
					common = append(common, s.V)
				}
			}
		case *object.Tuple:
			for _, item := range v.V {
				if s, ok := item.(*object.Str); ok {
					common = append(common, s.V)
				}
			}
		default:
			return nil, object.Errorf(i.typeErr, "cmpfiles() common must be a sequence")
		}

		match := &object.List{}
		mismatch := &object.List{}
		errors := &object.List{}
		shallowObj := object.BoolOf(shallow)

		for _, name := range common {
			ax := filepath.Join(dirA.V, name)
			bx := filepath.Join(dirB.V, name)
			nameObj := &object.Str{V: name}

			result, err := cmpFunc.Call(nil, []object.Object{
				&object.Str{V: ax}, &object.Str{V: bx}, shallowObj,
			}, nil)
			if err != nil {
				errors.V = append(errors.V, nameObj)
				continue
			}
			if object.Truthy(result) {
				match.V = append(match.V, nameObj)
			} else {
				mismatch.V = append(mismatch.V, nameObj)
			}
		}

		return &object.Tuple{V: []object.Object{match, mismatch, errors}}, nil
	}})

	// dircmp class
	m.Dict.SetStr("dircmp", i.buildDircmpClass(cmpFunc, defaultIgnores))

	return m
}

// buildDircmpClass constructs the dircmp class backed by Go logic.
func (i *Interp) buildDircmpClass(cmpFunc *object.BuiltinFunc, defaultIgnores *object.List) *object.Class {
	cls := &object.Class{Name: "dircmp", Bases: nil, Dict: object.NewDict()}

	// __init__(self, a, b, ignore=None, hide=None, *, shallow=True)
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "dircmp() requires 2 positional arguments")
		}
		self, ok0 := a[0].(*object.Instance)
		left, ok1 := a[1].(*object.Str)
		right, ok2 := a[2].(*object.Str)
		if !ok0 || !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "dircmp() a and b must be str")
		}

		self.Dict.SetStr("left", left)
		self.Dict.SetStr("right", right)

		// ignore
		ignore := object.Object(defaultIgnores)
		if len(a) >= 4 && a[3] != object.None {
			ignore = a[3]
		}
		if kw != nil {
			if v, ok := kw.GetStr("ignore"); ok && v != object.None {
				ignore = v
			}
		}
		self.Dict.SetStr("ignore", ignore)

		// hide
		hide := object.Object(&object.List{V: []object.Object{
			&object.Str{V: "."}, &object.Str{V: ".."},
		}})
		if len(a) >= 5 && a[4] != object.None {
			hide = a[4]
		}
		if kw != nil {
			if v, ok := kw.GetStr("hide"); ok && v != object.None {
				hide = v
			}
		}
		self.Dict.SetStr("hide", hide)

		// shallow
		shallow := true
		if kw != nil {
			if v, ok := kw.GetStr("shallow"); ok {
				shallow = object.Truthy(v)
			}
		}
		self.Dict.SetStr("shallow", object.BoolOf(shallow))

		return object.None, nil
	}})

	// phase0: compute left_list and right_list
	cls.Dict.SetStr("phase0", &object.BuiltinFunc{Name: "phase0", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		leftObj, _ := self.Dict.GetStr("left")
		rightObj, _ := self.Dict.GetStr("right")
		hideObj, _ := self.Dict.GetStr("hide")
		ignoreObj, _ := self.Dict.GetStr("ignore")

		skip := dircmpSkipSet(hideObj, ignoreObj)
		ll := dircmpListDir(leftObj.(*object.Str).V, skip)
		rl := dircmpListDir(rightObj.(*object.Str).V, skip)

		self.Dict.SetStr("left_list", strSliceToList(ll))
		self.Dict.SetStr("right_list", strSliceToList(rl))
		return object.None, nil
	}})

	// phase1: compute common, left_only, right_only
	cls.Dict.SetStr("phase1", &object.BuiltinFunc{Name: "phase1", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		dircmpEnsurePhase(i, self, "left_list")

		ll := listToStrSlice(mustList(self, "left_list"))
		rl := listToStrSlice(mustList(self, "right_list"))

		lb := make(map[string]bool, len(ll))
		rb := make(map[string]bool, len(rl))
		for _, s := range ll {
			lb[s] = true
		}
		for _, s := range rl {
			rb[s] = true
		}

		var common, leftOnly, rightOnly []string
		for _, s := range ll {
			if rb[s] {
				common = append(common, s)
			} else {
				leftOnly = append(leftOnly, s)
			}
		}
		for _, s := range rl {
			if !lb[s] {
				rightOnly = append(rightOnly, s)
			}
		}

		self.Dict.SetStr("common", strSliceToList(common))
		self.Dict.SetStr("left_only", strSliceToList(leftOnly))
		self.Dict.SetStr("right_only", strSliceToList(rightOnly))
		return object.None, nil
	}})

	// phase2: categorise common names into common_dirs, common_files, common_funny
	cls.Dict.SetStr("phase2", &object.BuiltinFunc{Name: "phase2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		dircmpEnsurePhase(i, self, "common")

		leftObj, _ := self.Dict.GetStr("left")
		rightObj, _ := self.Dict.GetStr("right")
		leftStr := leftObj.(*object.Str).V
		rightStr := rightObj.(*object.Str).V
		common := listToStrSlice(mustList(self, "common"))

		var dirs, files, funny []string
		for _, name := range common {
			aPath := filepath.Join(leftStr, name)
			bPath := filepath.Join(rightStr, name)

			aInfo, aErr := os.Stat(aPath)
			bInfo, bErr := os.Stat(bPath)
			if aErr != nil || bErr != nil {
				funny = append(funny, name)
				continue
			}

			aMode := goModeToPosix(aInfo.Mode()) & 0o170000
			bMode := goModeToPosix(bInfo.Mode()) & 0o170000
			if aMode != bMode {
				funny = append(funny, name)
			} else if aMode == 0o040000 {
				dirs = append(dirs, name)
			} else if aMode == 0o100000 {
				files = append(files, name)
			} else {
				funny = append(funny, name)
			}
		}

		self.Dict.SetStr("common_dirs", strSliceToList(dirs))
		self.Dict.SetStr("common_files", strSliceToList(files))
		self.Dict.SetStr("common_funny", strSliceToList(funny))
		return object.None, nil
	}})

	// phase3: compare common_files → same_files, diff_files, funny_files
	cls.Dict.SetStr("phase3", &object.BuiltinFunc{Name: "phase3", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		dircmpEnsurePhase(i, self, "common_files")

		leftObj2, _ := self.Dict.GetStr("left")
		rightObj2, _ := self.Dict.GetStr("right")
		leftStr := leftObj2.(*object.Str).V
		rightStr := rightObj2.(*object.Str).V
		shallowObj, _ := self.Dict.GetStr("shallow")
		files := listToStrSlice(mustList(self, "common_files"))

		var same, diff, funnyFiles []string
		for _, name := range files {
			ax := filepath.Join(leftStr, name)
			bx := filepath.Join(rightStr, name)
			result, err := cmpFunc.Call(nil, []object.Object{
				&object.Str{V: ax}, &object.Str{V: bx}, shallowObj,
			}, nil)
			if err != nil {
				funnyFiles = append(funnyFiles, name)
			} else if object.Truthy(result) {
				same = append(same, name)
			} else {
				diff = append(diff, name)
			}
		}

		self.Dict.SetStr("same_files", strSliceToList(same))
		self.Dict.SetStr("diff_files", strSliceToList(diff))
		self.Dict.SetStr("funny_files", strSliceToList(funnyFiles))
		return object.None, nil
	}})

	// phase4: build subdirs dict of dircmp instances for common_dirs
	cls.Dict.SetStr("phase4", &object.BuiltinFunc{Name: "phase4", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		dircmpEnsurePhase(i, self, "common_dirs")

		leftObj3, _ := self.Dict.GetStr("left")
		rightObj3, _ := self.Dict.GetStr("right")
		leftStr := leftObj3.(*object.Str).V
		rightStr := rightObj3.(*object.Str).V
		ignoreObj, _ := self.Dict.GetStr("ignore")
		hideObj, _ := self.Dict.GetStr("hide")
		shallowObj, _ := self.Dict.GetStr("shallow")
		dirs := listToStrSlice(mustList(self, "common_dirs"))

		subdirs := object.NewDict()
		for _, name := range dirs {
			aPath := filepath.Join(leftStr, name)
			bPath := filepath.Join(rightStr, name)

			subInst := &object.Instance{Class: cls, Dict: object.NewDict()}
			subInst.Dict.SetStr("left", &object.Str{V: aPath})
			subInst.Dict.SetStr("right", &object.Str{V: bPath})
			subInst.Dict.SetStr("ignore", ignoreObj)
			subInst.Dict.SetStr("hide", hideObj)
			subInst.Dict.SetStr("shallow", shallowObj)
			subdirs.SetStr(name, subInst)
		}
		self.Dict.SetStr("subdirs", subdirs)
		return object.None, nil
	}})

	// phase4_closure: recursively call phase4 on subdirs
	cls.Dict.SetStr("phase4_closure", &object.BuiltinFunc{Name: "phase4_closure", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if err := dircmpCallMethod(i, self, "phase4"); err != nil {
			return nil, err
		}
		subdirsObj, _ := self.Dict.GetStr("subdirs")
		if d, ok := subdirsObj.(*object.Dict); ok {
			_, vals := d.Items()
			for _, v := range vals {
				if sub, ok := v.(*object.Instance); ok {
					if err := dircmpCallMethod(i, sub, "phase4_closure"); err != nil {
						return nil, err
					}
				}
			}
		}
		return object.None, nil
	}})

	// __getattr__ — lazy phase dispatch
	methodmap := map[string]string{
		"left_list":    "phase0",
		"right_list":   "phase0",
		"common":       "phase1",
		"left_only":    "phase1",
		"right_only":   "phase1",
		"common_dirs":  "phase2",
		"common_files": "phase2",
		"common_funny": "phase2",
		"same_files":   "phase3",
		"diff_files":   "phase3",
		"funny_files":  "phase3",
		"subdirs":      "phase4",
	}
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "__getattr__ requires 2 arguments")
		}
		self := a[0].(*object.Instance)
		attrName, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "__getattr__ attr must be str")
		}
		phase, known := methodmap[attrName.V]
		if !known {
			return nil, object.Errorf(i.attrErr, "'dircmp' object has no attribute '%s'", attrName.V)
		}
		if err := dircmpCallMethod(i, self, phase); err != nil {
			return nil, err
		}
		v, ok := self.Dict.GetStr(attrName.V)
		if !ok {
			return nil, object.Errorf(i.attrErr, "'dircmp' object has no attribute '%s'", attrName.V)
		}
		return v, nil
	}})

	// report()
	cls.Dict.SetStr("report", &object.BuiltinFunc{Name: "report", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if err := dircmpEnsureAll(i, self); err != nil {
			return nil, err
		}
		return nil, dircmpReport(i, self, false, false)
	}})

	// report_partial_closure()
	cls.Dict.SetStr("report_partial_closure", &object.BuiltinFunc{Name: "report_partial_closure", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if err := dircmpEnsureAll(i, self); err != nil {
			return nil, err
		}
		if err := dircmpReport(i, self, false, false); err != nil {
			return nil, err
		}
		subdirsObj, _ := self.Dict.GetStr("subdirs")
		if d, ok := subdirsObj.(*object.Dict); ok {
			_, vals := d.Items()
			for _, v := range vals {
				if sub, ok := v.(*object.Instance); ok {
					if err := dircmpEnsureAll(i, sub); err != nil {
						return nil, err
					}
					fmt.Fprintln(i.Stdout)
					if err := dircmpReport(i, sub, false, false); err != nil {
						return nil, err
					}
				}
			}
		}
		return object.None, nil
	}})

	// report_full_closure()
	cls.Dict.SetStr("report_full_closure", &object.BuiltinFunc{Name: "report_full_closure", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		return nil, dircmpReportFull(i, self)
	}})

	return cls
}

// dircmpEnsurePhase calls the phase that computes the given attribute if not yet set.
func dircmpEnsurePhase(i *Interp, self *object.Instance, attr string) {
	if _, ok := self.Dict.GetStr(attr); ok {
		return
	}
	phaseMap := map[string]string{
		"left_list":    "phase0",
		"right_list":   "phase0",
		"common":       "phase1",
		"left_only":    "phase1",
		"right_only":   "phase1",
		"common_dirs":  "phase2",
		"common_files": "phase2",
		"common_funny": "phase2",
		"same_files":   "phase3",
		"diff_files":   "phase3",
		"funny_files":  "phase3",
		"subdirs":      "phase4",
	}
	if phase, ok := phaseMap[attr]; ok {
		_ = dircmpCallMethod(i, self, phase)
	}
}

// dircmpEnsureAll ensures all lazily-computed attributes are populated.
func dircmpEnsureAll(i *Interp, self *object.Instance) error {
	for _, phase := range []string{"phase0", "phase1", "phase2", "phase3", "phase4"} {
		if err := dircmpCallMethod(i, self, phase); err != nil {
			return err
		}
	}
	return nil
}

// dircmpCallMethod calls a named method on a dircmp instance.
func dircmpCallMethod(i *Interp, self *object.Instance, method string) error {
	fn, ok := self.Class.Dict.GetStr(method)
	if !ok {
		return fmt.Errorf("dircmp: no method %s", method)
	}
	bf, ok := fn.(*object.BuiltinFunc)
	if !ok {
		return fmt.Errorf("dircmp: %s is not a builtin", method)
	}
	_, err := bf.Call(nil, []object.Object{self}, nil)
	return err
}

// dircmpReport prints the diff report for a single dircmp instance.
func dircmpReport(i *Interp, self *object.Instance, _, _ bool) error {
	leftO, _ := self.Dict.GetStr("left")
	rightO, _ := self.Dict.GetStr("right")
	left := leftO.(*object.Str).V
	right := rightO.(*object.Str).V

	fmt.Fprintf(i.Stdout, "diff %s %s\n", left, right)

	if leftOnly := listToStrSlice(mustList(self, "left_only")); len(leftOnly) > 0 {
		sort.Strings(leftOnly)
		fmt.Fprintf(i.Stdout, "Only in %s : %s\n", left, formatStrList(leftOnly))
	}
	if rightOnly := listToStrSlice(mustList(self, "right_only")); len(rightOnly) > 0 {
		sort.Strings(rightOnly)
		fmt.Fprintf(i.Stdout, "Only in %s : %s\n", right, formatStrList(rightOnly))
	}
	if same := listToStrSlice(mustList(self, "same_files")); len(same) > 0 {
		sort.Strings(same)
		fmt.Fprintf(i.Stdout, "Identical files : %s\n", formatStrList(same))
	}
	if diff := listToStrSlice(mustList(self, "diff_files")); len(diff) > 0 {
		sort.Strings(diff)
		fmt.Fprintf(i.Stdout, "Differing files : %s\n", formatStrList(diff))
	}
	if funny := listToStrSlice(mustList(self, "funny_files")); len(funny) > 0 {
		sort.Strings(funny)
		fmt.Fprintf(i.Stdout, "Trouble with common files : %s\n", formatStrList(funny))
	}
	if dirs := listToStrSlice(mustList(self, "common_dirs")); len(dirs) > 0 {
		sort.Strings(dirs)
		fmt.Fprintf(i.Stdout, "Common subdirectories : %s\n", formatStrList(dirs))
	}
	if funny := listToStrSlice(mustList(self, "common_funny")); len(funny) > 0 {
		sort.Strings(funny)
		fmt.Fprintf(i.Stdout, "Common funny cases : %s\n", formatStrList(funny))
	}
	return nil
}

// dircmpReportFull recursively prints the diff report.
func dircmpReportFull(i *Interp, self *object.Instance) error {
	if err := dircmpEnsureAll(i, self); err != nil {
		return err
	}
	if err := dircmpReport(i, self, false, false); err != nil {
		return err
	}
	subdirsObj, _ := self.Dict.GetStr("subdirs")
	if d, ok := subdirsObj.(*object.Dict); ok {
		_, vals := d.Items()
		for _, v := range vals {
			if sub, ok := v.(*object.Instance); ok {
				fmt.Fprintln(i.Stdout)
				if err := dircmpReportFull(i, sub); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// dircmpSkipSet builds a set of names to exclude from directory listings.
func dircmpSkipSet(hideObj, ignoreObj object.Object) map[string]bool {
	skip := make(map[string]bool)
	for _, obj := range []object.Object{hideObj, ignoreObj} {
		if lst, ok := obj.(*object.List); ok {
			for _, item := range lst.V {
				if s, ok := item.(*object.Str); ok {
					skip[s.V] = true
				}
			}
		}
	}
	return skip
}

// dircmpListDir returns a sorted list of entries in dir, excluding skip names.
func dircmpListDir(dir string, skip map[string]bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !skip[e.Name()] {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// mustList returns the []string from a named attribute, or nil.
func mustList(self *object.Instance, name string) *object.List {
	v, _ := self.Dict.GetStr(name)
	if l, ok := v.(*object.List); ok {
		return l
	}
	return &object.List{}
}

// strSliceToList converts a []string to *object.List.
func strSliceToList(ss []string) *object.List {
	v := make([]object.Object, len(ss))
	for idx, s := range ss {
		v[idx] = &object.Str{V: s}
	}
	return &object.List{V: v}
}

// listToStrSlice converts *object.List to []string.
func listToStrSlice(lst *object.List) []string {
	if lst == nil {
		return nil
	}
	out := make([]string, 0, len(lst.V))
	for _, item := range lst.V {
		if s, ok := item.(*object.Str); ok {
			out = append(out, s.V)
		}
	}
	return out
}

// formatStrList formats a []string as a Python list repr e.g. ['a', 'b'].
func formatStrList(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b := []byte{'['}
	for idx, s := range ss {
		if idx > 0 {
			b = append(b, ',', ' ')
		}
		b = append(b, '\'')
		b = append(b, s...)
		b = append(b, '\'')
	}
	b = append(b, ']')
	return string(b)
}
