package vm

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildBdb() *object.Module {
	m := &object.Module{Name: "bdb", Dict: object.NewDict()}

	// ── Breakpoint class-level state ──────────────────────────────────────
	// bpList: "canonical:line" → []*object.Instance
	// bpByNumList: Python list, index = bp number, element = bp or None
	// bpNext: next breakpoint number (1-based)
	var mu sync.Mutex
	bpNext := 1
	bpList := make(map[string][]*object.Instance)
	bpByNumList := &object.List{V: []object.Object{object.None}} // index 0 is always None

	bpListKey := func(file string, line int) string {
		return fmt.Sprintf("%s:%d", file, line)
	}

	// ── BdbQuit ───────────────────────────────────────────────────────────
	bdbQuit := &object.Class{
		Name:  "BdbQuit",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("BdbQuit", bdbQuit)

	// ── Breakpoint ────────────────────────────────────────────────────────
	bpCls := &object.Class{Name: "Breakpoint", Dict: object.NewDict()}
	bpCls.Dict.SetStr("bpbynumber", bpByNumList)

	// Helper: initialise a Breakpoint instance
	bdbInitBP := func(inst *object.Instance, file string, line int, temporary bool, cond, funcname object.Object) {
		mu.Lock()
		num := bpNext
		bpNext++
		key := bpListKey(file, line)
		bpList[key] = append(bpList[key], inst)
		bpByNumList.V = append(bpByNumList.V, inst)
		mu.Unlock()

		inst.Dict.SetStr("file", &object.Str{V: file})
		inst.Dict.SetStr("line", object.NewInt(int64(line)))
		inst.Dict.SetStr("temporary", object.BoolOf(temporary))
		if cond == nil {
			cond = object.None
		}
		if funcname == nil {
			funcname = object.None
		}
		inst.Dict.SetStr("cond", cond)
		inst.Dict.SetStr("funcname", funcname)
		inst.Dict.SetStr("enabled", object.BoolOf(true))
		inst.Dict.SetStr("hits", object.NewInt(0))
		inst.Dict.SetStr("ignore", object.NewInt(0))
		inst.Dict.SetStr("number", object.NewInt(int64(num)))
		inst.Dict.SetStr("func_first_executable_line", object.None)
	}

	// Breakpoint.__init__
	bpCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "Breakpoint requires file and line")
			}
			inst := a[0].(*object.Instance)
			file := object.Str_(a[1])
			lineObj, ok := a[2].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "Breakpoint line must be int")
			}
			line := int(lineObj.Int64())

			temporary := false
			var cond, funcname object.Object
			cond = object.None
			funcname = object.None
			if len(a) >= 4 {
				temporary = object.Truthy(a[3])
			}
			if len(a) >= 5 {
				cond = a[4]
			}
			if len(a) >= 6 {
				funcname = a[5]
			}
			if kw != nil {
				if v, ok2 := kw.GetStr("temporary"); ok2 {
					temporary = object.Truthy(v)
				}
				if v, ok2 := kw.GetStr("cond"); ok2 {
					cond = v
				}
				if v, ok2 := kw.GetStr("funcname"); ok2 {
					funcname = v
				}
			}
			bdbInitBP(inst, file, line, temporary, cond, funcname)
			return object.None, nil
		},
	})

	// Breakpoint.deleteMe
	bpCls.Dict.SetStr("deleteMe", &object.BuiltinFunc{
		Name: "deleteMe",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			fileObj, _ := inst.Dict.GetStr("file")
			lineObj, _ := inst.Dict.GetStr("line")
			numObj, _ := inst.Dict.GetStr("number")
			file := object.Str_(fileObj)
			line := 0
			if li, ok := lineObj.(*object.Int); ok {
				line = int(li.Int64())
			}
			num := 0
			if ni, ok := numObj.(*object.Int); ok {
				num = int(ni.Int64())
			}
			key := bpListKey(file, line)
			mu.Lock()
			if num < len(bpByNumList.V) {
				bpByNumList.V[num] = object.None
			}
			lst := bpList[key]
			for idx, bp := range lst {
				if bp == inst {
					bpList[key] = append(lst[:idx], lst[idx+1:]...)
					break
				}
			}
			if len(bpList[key]) == 0 {
				delete(bpList, key)
			}
			mu.Unlock()
			return object.None, nil
		},
	})

	// Breakpoint.enable
	bpCls.Dict.SetStr("enable", &object.BuiltinFunc{
		Name: "enable",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a[0].(*object.Instance).Dict.SetStr("enabled", object.BoolOf(true))
			return object.None, nil
		},
	})

	// Breakpoint.disable
	bpCls.Dict.SetStr("disable", &object.BuiltinFunc{
		Name: "disable",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a[0].(*object.Instance).Dict.SetStr("enabled", object.BoolOf(false))
			return object.None, nil
		},
	})

	// Breakpoint.bpformat
	bpCls.Dict.SetStr("bpformat", &object.BuiltinFunc{
		Name: "bpformat",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			numObj, _ := inst.Dict.GetStr("number")
			tmpObj, _ := inst.Dict.GetStr("temporary")
			enaObj, _ := inst.Dict.GetStr("enabled")
			fileObj, _ := inst.Dict.GetStr("file")
			lineObj, _ := inst.Dict.GetStr("line")
			condObj, _ := inst.Dict.GetStr("cond")
			ignObj, _ := inst.Dict.GetStr("ignore")
			hitsObj, _ := inst.Dict.GetStr("hits")

			num := int64(0)
			if ni, ok := numObj.(*object.Int); ok {
				num = ni.Int64()
			}
			line := int64(0)
			if li, ok := lineObj.(*object.Int); ok {
				line = li.Int64()
			}
			ign := int64(0)
			if ii2, ok := ignObj.(*object.Int); ok {
				ign = ii2.Int64()
			}
			hits := int64(0)
			if hi, ok := hitsObj.(*object.Int); ok {
				hits = hi.Int64()
			}
			file := object.Str_(fileObj)

			disp := "keep "
			if object.Truthy(tmpObj) {
				disp = "del  "
			}
			if object.Truthy(enaObj) {
				disp += "yes  "
			} else {
				disp += "no   "
			}

			ret := fmt.Sprintf("%-4dbreakpoint   %sat %s:%d", num, disp, file, line)
			if condObj != nil && condObj != object.None {
				ret += fmt.Sprintf("\n\tstop only if %s", object.Str_(condObj))
			}
			if ign > 0 {
				ret += fmt.Sprintf("\n\tignore next %d hits", ign)
			}
			if hits == 1 {
				ret += "\n\tbreakpoint already hit 1 time"
			} else if hits > 1 {
				ret += fmt.Sprintf("\n\tbreakpoint already hit %d times", hits)
			}
			return &object.Str{V: ret}, nil
		},
	})

	// Breakpoint.bpprint
	bpCls.Dict.SetStr("bpprint", &object.BuiltinFunc{
		Name: "bpprint",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			interp := ii.(*Interp)
			// get formatted string
			fmtFn, ok := bpCls.Dict.GetStr("bpformat")
			if !ok {
				return object.None, nil
			}
			fmtResult, err := fmtFn.(*object.BuiltinFunc).Call(nil, []object.Object{inst}, nil)
			if err != nil {
				return nil, err
			}
			s := object.Str_(fmtResult) + "\n"

			// determine output target
			var outObj object.Object
			if kw != nil {
				if v, ok2 := kw.GetStr("out"); ok2 && v != object.None {
					outObj = v
				}
			}
			if len(a) >= 2 && a[1] != object.None {
				outObj = a[1]
			}
			if outObj != nil {
				writeMethod, werr := interp.getAttr(outObj, "write")
				if werr != nil {
					return nil, werr
				}
				_, werr = interp.callObject(writeMethod, []object.Object{&object.Str{V: s}}, nil)
				return object.None, werr
			}
			fmt.Fprint(interp.Stdout, s)
			return object.None, nil
		},
	})

	m.Dict.SetStr("Breakpoint", bpCls)

	// ── Bdb class ──────────────────────────────────────────────────────────
	bdbCls := &object.Class{Name: "Bdb", Dict: object.NewDict()}

	// Helper: get canonical filename
	bdbCanonic := func(self *object.Instance, filename string) string {
		if len(filename) >= 2 && filename[0] == '<' && filename[len(filename)-1] == '>' {
			return filename
		}
		fncacheObj, _ := self.Dict.GetStr("_fncache")
		fncache, _ := fncacheObj.(*object.Dict)
		if fncache != nil {
			if v, ok := fncache.GetStr(filename); ok {
				return object.Str_(v)
			}
		}
		abs, err := filepath.Abs(filename)
		if err != nil {
			abs = filename
		}
		if fncache != nil {
			fncache.SetStr(filename, &object.Str{V: abs})
		}
		return abs
	}

	// Helper: get or create breaks dict on Bdb instance
	bdbBreaks := func(self *object.Instance) *object.Dict {
		breaksObj, _ := self.Dict.GetStr("breaks")
		d, _ := breaksObj.(*object.Dict)
		if d == nil {
			d = object.NewDict()
			self.Dict.SetStr("breaks", d)
		}
		return d
	}

	// Helper: check if lineno is in the breaks list for a file
	bdbHasBreak := func(self *object.Instance, canonical string, lineno int) bool {
		d := bdbBreaks(self)
		linesObj, ok := d.GetStr(canonical)
		if !ok {
			return false
		}
		lst, ok2 := linesObj.(*object.List)
		if !ok2 {
			return false
		}
		for _, v := range lst.V {
			if n, ok3 := v.(*object.Int); ok3 && int(n.Int64()) == lineno {
				return true
			}
		}
		return false
	}

	// Bdb.__init__
	bdbCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("breaks", object.NewDict())
			inst.Dict.SetStr("_fncache", object.NewDict())
			inst.Dict.SetStr("quitting", object.BoolOf(false))
			inst.Dict.SetStr("botframe", object.None)
			inst.Dict.SetStr("stopframe", object.None)
			inst.Dict.SetStr("returnframe", object.None)
			inst.Dict.SetStr("frame_returning", object.None)

			// skip: list of module patterns to skip
			var skip object.Object = object.None
			if len(a) >= 2 && a[1] != object.None {
				skip = a[1]
			}
			if kw != nil {
				if v, ok := kw.GetStr("skip"); ok {
					skip = v
				}
			}
			inst.Dict.SetStr("_skip", skip)
			return object.None, nil
		},
	})

	// Bdb.canonic
	bdbCls.Dict.SetStr("canonic", &object.BuiltinFunc{
		Name: "canonic",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			return &object.Str{V: bdbCanonic(self, filename)}, nil
		},
	})

	// Bdb.reset
	bdbCls.Dict.SetStr("reset", &object.BuiltinFunc{
		Name: "reset",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			self.Dict.SetStr("botframe", object.None)
			self.Dict.SetStr("stopframe", object.None)
			self.Dict.SetStr("returnframe", object.None)
			self.Dict.SetStr("quitting", object.BoolOf(false))
			return object.None, nil
		},
	})

	// Bdb.is_skipped_module
	bdbCls.Dict.SetStr("is_skipped_module", &object.BuiltinFunc{
		Name: "is_skipped_module",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			modName := object.Str_(a[1])
			skipObj, _ := self.Dict.GetStr("_skip")
			if skipObj == nil || skipObj == object.None {
				return object.BoolOf(false), nil
			}
			lst, ok := skipObj.(*object.List)
			if !ok {
				return object.BoolOf(false), nil
			}
			for _, v := range lst.V {
				if object.Str_(v) == modName {
					return object.BoolOf(true), nil
				}
			}
			return object.BoolOf(false), nil
		},
	})

	// Bdb.set_break
	bdbCls.Dict.SetStr("set_break", &object.BuiltinFunc{
		Name: "set_break",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "set_break() requires filename and lineno")
			}
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			linenoObj, ok := a[2].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "lineno must be int")
			}
			lineno := int(linenoObj.Int64())

			temporary := false
			var cond object.Object = object.None
			var funcname object.Object = object.None
			if len(a) >= 4 {
				temporary = object.Truthy(a[3])
			}
			if len(a) >= 5 {
				cond = a[4]
			}
			if len(a) >= 6 {
				funcname = a[5]
			}
			if kw != nil {
				if v, ok2 := kw.GetStr("temporary"); ok2 {
					temporary = object.Truthy(v)
				}
				if v, ok2 := kw.GetStr("cond"); ok2 {
					cond = v
				}
				if v, ok2 := kw.GetStr("funcname"); ok2 {
					funcname = v
				}
			}

			canonical := bdbCanonic(self, filename)
			d := bdbBreaks(self)

			// Add lineno to breaks[canonical] if not already present
			var linesList *object.List
			if existing, ok2 := d.GetStr(canonical); ok2 {
				linesList, _ = existing.(*object.List)
			}
			if linesList == nil {
				linesList = &object.List{}
			}
			found := false
			for _, v := range linesList.V {
				if n, ok2 := v.(*object.Int); ok2 && int(n.Int64()) == lineno {
					found = true
					break
				}
			}
			if !found {
				linesList.V = append(linesList.V, object.NewInt(int64(lineno)))
			}
			d.SetStr(canonical, linesList)

			// Create Breakpoint
			bp := &object.Instance{Class: bpCls, Dict: object.NewDict()}
			bdbInitBP(bp, canonical, lineno, temporary, cond, funcname)
			return object.None, nil
		},
	})

	// Bdb.clear_break
	bdbCls.Dict.SetStr("clear_break", &object.BuiltinFunc{
		Name: "clear_break",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			linenoObj, ok := a[2].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "lineno must be int")
			}
			lineno := int(linenoObj.Int64())
			canonical := bdbCanonic(self, filename)

			d := bdbBreaks(self)
			if _, ok2 := d.GetStr(canonical); !ok2 {
				return &object.Str{V: fmt.Sprintf("There are no breakpoints in %s", canonical)}, nil
			}
			if !bdbHasBreak(self, canonical, lineno) {
				return &object.Str{V: fmt.Sprintf("There is no breakpoint in %s at line %d", canonical, lineno)}, nil
			}

			// Delete all Breakpoint objects for this location
			key := bpListKey(canonical, lineno)
			mu.Lock()
			for _, bp := range bpList[key] {
				numObj, _ := bp.Dict.GetStr("number")
				if ni, ok3 := numObj.(*object.Int); ok3 {
					num := int(ni.Int64())
					if num < len(bpByNumList.V) {
						bpByNumList.V[num] = object.None
					}
				}
			}
			delete(bpList, key)
			mu.Unlock()

			// Remove lineno from breaks[canonical]
			if linesObj, ok2 := d.GetStr(canonical); ok2 {
				if lst, ok3 := linesObj.(*object.List); ok3 {
					newV := lst.V[:0]
					for _, v := range lst.V {
						if n, ok4 := v.(*object.Int); !ok4 || int(n.Int64()) != lineno {
							newV = append(newV, v)
						}
					}
					if len(newV) == 0 {
						d.Delete(&object.Str{V: canonical})
					} else {
						lst.V = newV
					}
				}
			}
			return object.None, nil
		},
	})

	// Bdb.clear_bpbynumber
	bdbCls.Dict.SetStr("clear_bpbynumber", &object.BuiltinFunc{
		Name: "clear_bpbynumber",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			argObj := a[1]
			numObj, ok := argObj.(*object.Int)
			if !ok {
				return &object.Str{V: fmt.Sprintf("Non-numeric breakpoint number %s", object.Str_(argObj))}, nil
			}
			num := int(numObj.Int64())

			mu.Lock()
			if num <= 0 || num >= len(bpByNumList.V) {
				mu.Unlock()
				return &object.Str{V: fmt.Sprintf("Breakpoint number %d out of range", num)}, nil
			}
			bp, ok2 := bpByNumList.V[num].(*object.Instance)
			if !ok2 || bp == nil {
				mu.Unlock()
				return &object.Str{V: fmt.Sprintf("Breakpoint %d already deleted", num)}, nil
			}
			mu.Unlock()

			// Remove from bpList and bpByNumber
			fileObj, _ := bp.Dict.GetStr("file")
			lineObj, _ := bp.Dict.GetStr("line")
			file := object.Str_(fileObj)
			line := 0
			if li, ok3 := lineObj.(*object.Int); ok3 {
				line = int(li.Int64())
			}
			key := bpListKey(file, line)
			mu.Lock()
			bpByNumList.V[num] = object.None
			lst := bpList[key]
			for idx, b := range lst {
				if b == bp {
					bpList[key] = append(lst[:idx], lst[idx+1:]...)
					break
				}
			}
			if len(bpList[key]) == 0 {
				delete(bpList, key)
			}
			mu.Unlock()

			// Remove lineno from self.breaks[file]
			d := bdbBreaks(self)
			lineno := line
			if linesObj, ok3 := d.GetStr(file); ok3 {
				if lst2, ok4 := linesObj.(*object.List); ok4 {
					newV := lst2.V[:0]
					for _, v := range lst2.V {
						if n, ok5 := v.(*object.Int); !ok5 || int(n.Int64()) != lineno {
							newV = append(newV, v)
						}
					}
					if len(newV) == 0 {
						d.Delete(&object.Str{V: file})
					} else {
						lst2.V = newV
					}
				}
			}
			return object.None, nil
		},
	})

	// Bdb.clear_all_file_breaks
	bdbCls.Dict.SetStr("clear_all_file_breaks", &object.BuiltinFunc{
		Name: "clear_all_file_breaks",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			canonical := bdbCanonic(self, filename)
			d := bdbBreaks(self)
			linesObj, ok := d.GetStr(canonical)
			if !ok {
				return &object.Str{V: fmt.Sprintf("There are no breakpoints in %s", canonical)}, nil
			}
			lst, _ := linesObj.(*object.List)
			mu.Lock()
			if lst != nil {
				for _, v := range lst.V {
					if n, ok2 := v.(*object.Int); ok2 {
						key := bpListKey(canonical, int(n.Int64()))
						for _, bp := range bpList[key] {
							numObj2, _ := bp.Dict.GetStr("number")
							if ni, ok3 := numObj2.(*object.Int); ok3 {
								num := int(ni.Int64())
								if num < len(bpByNumList.V) {
									bpByNumList.V[num] = object.None
								}
							}
						}
						delete(bpList, key)
					}
				}
			}
			mu.Unlock()
			d.Delete(&object.Str{V: canonical})
			return object.None, nil
		},
	})

	// Bdb.clear_all_breaks
	bdbCls.Dict.SetStr("clear_all_breaks", &object.BuiltinFunc{
		Name: "clear_all_breaks",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			d := bdbBreaks(self)
			keys, _ := d.Items()
			if len(keys) == 0 {
				return &object.Str{V: "There are no breakpoints"}, nil
			}
			mu.Lock()
			for key := range bpList {
				for _, bp := range bpList[key] {
					numObj2, _ := bp.Dict.GetStr("number")
					if ni, ok2 := numObj2.(*object.Int); ok2 {
						num := int(ni.Int64())
						if num < len(bpByNumList.V) {
							bpByNumList.V[num] = object.None
						}
					}
				}
			}
			// Only clear bpList entries owned by this Bdb's breaks
			for _, k := range keys {
				canonical := object.Str_(k)
				if linesObj, ok2 := d.GetStr(canonical); ok2 {
					if lst, ok3 := linesObj.(*object.List); ok3 {
						for _, v := range lst.V {
							if n, ok4 := v.(*object.Int); ok4 {
								delete(bpList, bpListKey(canonical, int(n.Int64())))
							}
						}
					}
				}
			}
			mu.Unlock()
			self.Dict.SetStr("breaks", object.NewDict())
			return object.None, nil
		},
	})

	// Bdb.get_bpbynumber
	bdbCls.Dict.SetStr("get_bpbynumber", &object.BuiltinFunc{
		Name: "get_bpbynumber",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			numObj, ok := a[1].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.valueErr, "Non-numeric breakpoint number %s", object.Str_(a[1]))
			}
			num := int(numObj.Int64())
			mu.Lock()
			defer mu.Unlock()
			if num <= 0 || num >= len(bpByNumList.V) {
				return nil, object.Errorf(i.valueErr, "Breakpoint number %d out of range", num)
			}
			bp := bpByNumList.V[num]
			if bp == object.None {
				return nil, object.Errorf(i.valueErr, "Breakpoint %d already deleted", num)
			}
			return bp, nil
		},
	})

	// Bdb.get_breaks
	bdbCls.Dict.SetStr("get_breaks", &object.BuiltinFunc{
		Name: "get_breaks",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			linenoObj, ok := a[2].(*object.Int)
			if !ok {
				return &object.List{}, nil
			}
			lineno := int(linenoObj.Int64())
			canonical := bdbCanonic(self, filename)

			if !bdbHasBreak(self, canonical, lineno) {
				return &object.List{}, nil
			}
			key := bpListKey(canonical, lineno)
			mu.Lock()
			bps := bpList[key]
			result := make([]object.Object, len(bps))
			for j, bp := range bps {
				result[j] = bp
			}
			mu.Unlock()
			return &object.List{V: result}, nil
		},
	})

	// Bdb.get_file_breaks
	bdbCls.Dict.SetStr("get_file_breaks", &object.BuiltinFunc{
		Name: "get_file_breaks",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			canonical := bdbCanonic(self, filename)
			d := bdbBreaks(self)
			linesObj, ok := d.GetStr(canonical)
			if !ok {
				return &object.List{}, nil
			}
			lst, ok2 := linesObj.(*object.List)
			if !ok2 {
				return &object.List{}, nil
			}
			var result []object.Object
			mu.Lock()
			for _, v := range lst.V {
				if n, ok3 := v.(*object.Int); ok3 {
					key := bpListKey(canonical, int(n.Int64()))
					for _, bp := range bpList[key] {
						result = append(result, bp)
					}
				}
			}
			mu.Unlock()
			return &object.List{V: result}, nil
		},
	})

	// Bdb.get_all_breaks
	bdbCls.Dict.SetStr("get_all_breaks", &object.BuiltinFunc{
		Name: "get_all_breaks",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			return bdbBreaks(self), nil
		},
	})

	// Bdb.set_quit
	bdbCls.Dict.SetStr("set_quit", &object.BuiltinFunc{
		Name: "set_quit",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a[0].(*object.Instance).Dict.SetStr("quitting", object.BoolOf(true))
			return object.None, nil
		},
	})

	// Stub control methods (no tracing in goipy)
	for _, name := range []string{"set_step", "set_next", "set_return", "set_until", "set_continue", "set_trace"} {
		n := name
		bdbCls.Dict.SetStr(n, &object.BuiltinFunc{
			Name: n,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		})
	}

	// Stub user hooks
	for _, name := range []string{"user_call", "user_line", "user_return", "user_exception", "do_clear"} {
		n := name
		bdbCls.Dict.SetStr(n, &object.BuiltinFunc{
			Name: n,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		})
	}

	// Stub trace methods
	for _, name := range []string{"trace_dispatch", "dispatch_line", "dispatch_call", "dispatch_return", "dispatch_exception", "break_here", "break_anywhere"} {
		n := name
		bdbCls.Dict.SetStr(n, &object.BuiltinFunc{
			Name: n,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		})
	}

	// Bdb.get_stack — stub returning empty stack
	bdbCls.Dict.SetStr("get_stack", &object.BuiltinFunc{
		Name: "get_stack",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{
				&object.List{},
				object.NewInt(0),
			}}, nil
		},
	})

	// Bdb.format_stack_entry — stub
	bdbCls.Dict.SetStr("format_stack_entry", &object.BuiltinFunc{
		Name: "format_stack_entry",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})

	// Bdb.runcall
	bdbCls.Dict.SetStr("runcall", &object.BuiltinFunc{
		Name: "runcall",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "runcall() requires a callable")
			}
			self := a[0].(*object.Instance)
			fn := a[1]
			fnArgs := a[2:]
			interp := ii.(*Interp)

			// reset
			self.Dict.SetStr("quitting", object.BoolOf(false))

			res, err := interp.callObject(fn, fnArgs, kw)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok {
					if object.IsSubclass(exc.Class, bdbQuit) {
						return object.None, nil
					}
				}
			}
			self.Dict.SetStr("quitting", object.BoolOf(true))
			return res, err
		},
	})

	// Bdb.run — stub (executes without tracing)
	bdbCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// Bdb.runeval — stub
	bdbCls.Dict.SetStr("runeval", &object.BuiltinFunc{
		Name: "runeval",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// Bdb.runctx — stub
	bdbCls.Dict.SetStr("runctx", &object.BuiltinFunc{
		Name: "runctx",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("Bdb", bdbCls)

	return m
}
