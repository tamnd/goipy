package vm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildCmd() *object.Module {
	m := &object.Module{Name: "cmd", Dict: object.NewDict()}

	identchars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

	cmdCls := &object.Class{Name: "Cmd", Dict: object.NewDict()}

	// Class-level defaults (not set per-instance by __init__).
	cmdCls.Dict.SetStr("prompt", &object.Str{V: "(Cmd) "})
	cmdCls.Dict.SetStr("identchars", &object.Str{V: identchars})
	cmdCls.Dict.SetStr("ruler", &object.Str{V: "="})
	cmdCls.Dict.SetStr("lastcmd", &object.Str{V: ""})
	cmdCls.Dict.SetStr("intro", object.None)
	cmdCls.Dict.SetStr("doc_leader", &object.Str{V: ""})
	cmdCls.Dict.SetStr("doc_header", &object.Str{V: "Documented commands (type help <topic>):"})
	cmdCls.Dict.SetStr("misc_header", &object.Str{V: "Miscellaneous help topics:"})
	cmdCls.Dict.SetStr("undoc_header", &object.Str{V: "Undocumented commands:"})
	cmdCls.Dict.SetStr("nohelp", &object.Str{V: "*** No help on %s"})
	cmdCls.Dict.SetStr("use_rawinput", object.True)

	// --- helpers (local closures capturing i) ---

	// cmdStr extracts the string value from an object, "" if not a Str.
	cmdStr := func(o object.Object) string {
		if o == nil || o == object.None {
			return ""
		}
		if s, ok := o.(*object.Str); ok {
			return s.V
		}
		return ""
	}

	// cmdGetAttrStr reads a string attribute from obj via getAttr.
	cmdGetAttrStr := func(obj object.Object, name string) string {
		v, err := i.getAttr(obj, name)
		if err != nil {
			return ""
		}
		return cmdStr(v)
	}

	// cmdWrite calls obj.stdout.write(s).
	cmdWrite := func(self *object.Instance, s string) error {
		stdout, err := i.getAttr(self, "stdout")
		if err != nil {
			return err
		}
		writeFn, err := i.getAttr(stdout, "write")
		if err != nil {
			return err
		}
		_, err = i.callObject(writeFn, []object.Object{&object.Str{V: s}}, nil)
		return err
	}

	// docOf extracts the Python docstring from a callable object.
	docOf := func(fn object.Object) string {
		switch f := fn.(type) {
		case *object.BoundMethod:
			if pf, ok := f.Fn.(*object.Function); ok {
				if pf.Doc != nil && pf.Doc != object.None {
					return cmdStr(pf.Doc)
				}
			}
			if bf, ok := f.Fn.(*object.BuiltinFunc); ok && bf.Attrs != nil {
				if d, ok2 := bf.Attrs.GetStr("__doc__"); ok2 {
					return cmdStr(d)
				}
			}
		case *object.Function:
			if f.Doc != nil && f.Doc != object.None {
				return cmdStr(f.Doc)
			}
		case *object.BuiltinFunc:
			if f.Attrs != nil {
				if d, ok := f.Attrs.GetStr("__doc__"); ok {
					return cmdStr(d)
				}
			}
		}
		return ""
	}

	// --- __init__ ---
	cmdCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		completekey := "tab"
		var stdin, stdout object.Object = object.None, object.None

		if len(a) > 1 && a[1] != object.None {
			completekey = cmdStr(a[1])
		}
		if len(a) > 2 {
			stdin = a[2]
		}
		if len(a) > 3 {
			stdout = a[3]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("completekey"); ok2 {
				completekey = cmdStr(v)
			}
			if v, ok2 := kw.GetStr("stdin"); ok2 {
				stdin = v
			}
			if v, ok2 := kw.GetStr("stdout"); ok2 {
				stdout = v
			}
		}

		if stdin == object.None {
			if sysMod, ok2 := i.builtinModule("sys"); ok2 {
				if v, ok3 := sysMod.Dict.GetStr("stdin"); ok3 {
					stdin = v
				}
			}
			if stdin == object.None {
				stdin = &object.TextStream{Name: "stdin"}
			}
		}
		if stdout == object.None {
			if sysMod, ok2 := i.builtinModule("sys"); ok2 {
				if v, ok3 := sysMod.Dict.GetStr("stdout"); ok3 {
					stdout = v
				}
			}
			if stdout == object.None {
				stdout = &object.TextStream{Name: "stdout", W: i.Stdout}
			}
		}

		self.Dict.SetStr("stdin", stdin)
		self.Dict.SetStr("stdout", stdout)
		self.Dict.SetStr("completekey", &object.Str{V: completekey})
		self.Dict.SetStr("cmdqueue", &object.List{V: []object.Object{}})
		return object.None, nil
	}})

	// --- parseline ---
	cmdCls.Dict.SetStr("parseline", &object.BuiltinFunc{Name: "parseline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("parseline() requires line argument")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, fmt.Errorf("parseline() self must be Cmd instance")
		}
		line := strings.TrimSpace(cmdStr(a[1]))
		if line == "" {
			return &object.Tuple{V: []object.Object{object.None, object.None, &object.Str{V: line}}}, nil
		}
		if line[0] == '?' {
			line = "help " + line[1:]
		} else if line[0] == '!' {
			if _, err := i.getAttr(self, "do_shell"); err == nil {
				line = "shell " + line[1:]
			} else {
				return &object.Tuple{V: []object.Object{object.None, object.None, &object.Str{V: line}}}, nil
			}
		}

		idStr := cmdGetAttrStr(self, "identchars")
		idSet := make(map[rune]bool, len(idStr))
		for _, r := range idStr {
			idSet[r] = true
		}
		runes := []rune(line)
		idx := 0
		for idx < len(runes) && idSet[runes[idx]] {
			idx++
		}
		cmd := string(runes[:idx])
		arg := strings.TrimSpace(string(runes[idx:]))
		return &object.Tuple{V: []object.Object{&object.Str{V: cmd}, &object.Str{V: arg}, &object.Str{V: line}}}, nil
	}})

	// --- emptyline ---
	cmdCls.Dict.SetStr("emptyline", &object.BuiltinFunc{Name: "emptyline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		last := cmdGetAttrStr(self, "lastcmd")
		if last != "" {
			onecmdFn, err := i.getAttr(self, "onecmd")
			if err != nil {
				return object.None, nil
			}
			return i.callObject(onecmdFn, []object.Object{&object.Str{V: last}}, nil)
		}
		return object.None, nil
	}})

	// --- default ---
	cmdCls.Dict.SetStr("default", &object.BuiltinFunc{Name: "default", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		line := cmdStr(a[1])
		return object.None, cmdWrite(self, "*** Unknown syntax: "+line+"\n")
	}})

	// --- precmd: hook, default returns line unchanged ---
	cmdCls.Dict.SetStr("precmd", &object.BuiltinFunc{Name: "precmd", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return a[1], nil // self=a[0], line=a[1]
	}})

	// --- postcmd: hook, default returns stop unchanged ---
	cmdCls.Dict.SetStr("postcmd", &object.BuiltinFunc{Name: "postcmd", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return a[1], nil // self=a[0], stop=a[1], line=a[2]
	}})

	// --- preloop / postloop: no-op hooks ---
	cmdCls.Dict.SetStr("preloop", &object.BuiltinFunc{Name: "preloop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cmdCls.Dict.SetStr("postloop", &object.BuiltinFunc{Name: "postloop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// --- completedefault ---
	cmdCls.Dict.SetStr("completedefault", &object.BuiltinFunc{Name: "completedefault", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: []object.Object{}}, nil
	}})

	// --- onecmd ---
	cmdCls.Dict.SetStr("onecmd", &object.BuiltinFunc{Name: "onecmd", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		lineStr := cmdStr(a[1])

		// parseline
		parselineFn, err := i.getAttr(self, "parseline")
		if err != nil {
			return nil, err
		}
		res, err := i.callObject(parselineFn, []object.Object{&object.Str{V: lineStr}}, nil)
		if err != nil {
			return nil, err
		}
		tup, ok2 := res.(*object.Tuple)
		if !ok2 || len(tup.V) < 3 {
			return object.None, nil
		}
		cmdPart := tup.V[0]
		argPart := tup.V[1]
		linePart := cmdStr(tup.V[2])

		// self.lastcmd = line
		self.Dict.SetStr("lastcmd", &object.Str{V: linePart})
		if linePart == "EOF" {
			self.Dict.SetStr("lastcmd", &object.Str{V: ""})
		}

		// empty line
		if linePart == "" {
			emptyFn, err2 := i.getAttr(self, "emptyline")
			if err2 != nil {
				return object.None, nil
			}
			return i.callObject(emptyFn, nil, nil)
		}

		// cmd is None (e.g. '!' without do_shell)
		if cmdPart == object.None {
			defaultFn, err2 := i.getAttr(self, "default")
			if err2 != nil {
				return object.None, nil
			}
			return i.callObject(defaultFn, []object.Object{&object.Str{V: linePart}}, nil)
		}

		// Reset lastcmd before dispatch (matches CPython behaviour).
		self.Dict.SetStr("lastcmd", &object.Str{V: ""})

		cmdName := cmdStr(cmdPart)
		if cmdName == "" {
			defaultFn, err2 := i.getAttr(self, "default")
			if err2 != nil {
				return object.None, nil
			}
			return i.callObject(defaultFn, []object.Object{&object.Str{V: linePart}}, nil)
		}

		doFn, err := i.getAttr(self, "do_"+cmdName)
		if err != nil {
			defaultFn, err2 := i.getAttr(self, "default")
			if err2 != nil {
				return object.None, nil
			}
			return i.callObject(defaultFn, []object.Object{&object.Str{V: linePart}}, nil)
		}
		argStr := cmdStr(argPart)
		return i.callObject(doFn, []object.Object{&object.Str{V: argStr}}, nil)
	}})

	// --- get_names: return all attribute names from the instance's class MRO ---
	cmdCls.Dict.SetStr("get_names", &object.BuiltinFunc{Name: "get_names", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.List{}, nil
		}
		mro := computeMRO(self.Class)
		seen := make(map[string]bool)
		var names []object.Object
		for _, cls := range mro {
			keys, _ := cls.Dict.Items()
			for _, k := range keys {
				if sk, ok2 := k.(*object.Str); ok2 {
					name := sk.V
					if !seen[name] {
						seen[name] = true
						names = append(names, &object.Str{V: name})
					}
				}
			}
		}
		return &object.List{V: names}, nil
	}})

	// --- print_topics ---
	cmdCls.Dict.SetStr("print_topics", &object.BuiltinFunc{Name: "print_topics", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 4 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		header := cmdStr(a[1])
		cmdList, ok2 := a[2].(*object.List)
		if !ok2 || len(cmdList.V) == 0 {
			return object.None, nil
		}
		ruler := cmdGetAttrStr(self, "ruler")

		var items []string
		for _, v := range cmdList.V {
			items = append(items, cmdStr(v))
		}
		sort.Strings(items)

		_ = cmdWrite(self, header+"\n")
		if ruler != "" {
			_ = cmdWrite(self, strings.Repeat(ruler, len(header))+"\n")
		}
		_ = cmdWrite(self, strings.Join(items, "  ")+"\n")
		_ = cmdWrite(self, "\n")
		return object.None, nil
	}})

	// --- do_help ---
	cmdCls.Dict.SetStr("do_help", &object.BuiltinFunc{Name: "do_help", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		arg := cmdStr(a[1])

		if arg != "" {
			// help_<arg> method takes priority.
			if helpFn, err := i.getAttr(self, "help_"+arg); err == nil {
				return i.callObject(helpFn, nil, nil)
			}
			// Docstring of do_<arg>.
			if doFn, err := i.getAttr(self, "do_"+arg); err == nil {
				if doc := docOf(doFn); doc != "" {
					return object.None, cmdWrite(self, doc+"\n")
				}
			}
			nohelp := cmdGetAttrStr(self, "nohelp")
			if nohelp == "" {
				nohelp = "*** No help on %s"
			}
			return object.None, cmdWrite(self, fmt.Sprintf(nohelp, arg)+"\n")
		}

		// List all do_* commands, split into documented / undocumented.
		getNsFn, err := i.getAttr(self, "get_names")
		if err != nil {
			return object.None, nil
		}
		namesObj, err := i.callObject(getNsFn, nil, nil)
		if err != nil {
			return object.None, nil
		}
		namesList, ok2 := namesObj.(*object.List)
		if !ok2 {
			return object.None, nil
		}

		var cmdsDoc, cmdsUndoc []string
		for _, v := range namesList.V {
			name := cmdStr(v)
			if !strings.HasPrefix(name, "do_") {
				continue
			}
			cmdName := name[3:]
			if cmdName == "" {
				continue
			}
			doFn, err2 := i.getAttr(self, name)
			if err2 != nil {
				continue
			}
			if docOf(doFn) != "" {
				cmdsDoc = append(cmdsDoc, cmdName)
			} else {
				cmdsUndoc = append(cmdsUndoc, cmdName)
			}
		}
		sort.Strings(cmdsDoc)
		sort.Strings(cmdsUndoc)

		docHeader := cmdGetAttrStr(self, "doc_header")
		undocHeader := cmdGetAttrStr(self, "undoc_header")
		ruler := cmdGetAttrStr(self, "ruler")

		if len(cmdsDoc) > 0 {
			_ = cmdWrite(self, docHeader+"\n")
			if ruler != "" {
				_ = cmdWrite(self, strings.Repeat(ruler, len(docHeader))+"\n")
			}
			_ = cmdWrite(self, strings.Join(cmdsDoc, "  ")+"\n")
			_ = cmdWrite(self, "\n")
		}
		if len(cmdsUndoc) > 0 {
			_ = cmdWrite(self, undocHeader+"\n")
			if ruler != "" {
				_ = cmdWrite(self, strings.Repeat(ruler, len(undocHeader))+"\n")
			}
			_ = cmdWrite(self, strings.Join(cmdsUndoc, "  ")+"\n")
			_ = cmdWrite(self, "\n")
		}
		return object.None, nil
	}})

	// --- columnize ---
	cmdCls.Dict.SetStr("columnize", &object.BuiltinFunc{Name: "columnize", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		lst, ok2 := a[1].(*object.List)
		if !ok2 || len(lst.V) == 0 {
			return object.None, cmdWrite(self, "<empty>\n")
		}
		var items []string
		for _, v := range lst.V {
			items = append(items, cmdStr(v))
		}
		return object.None, cmdWrite(self, strings.Join(items, "\n")+"\n")
	}})

	// --- cmdloop ---
	cmdCls.Dict.SetStr("cmdloop", &object.BuiltinFunc{Name: "cmdloop", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		// Resolve intro: explicit arg overrides self.intro.
		var intro object.Object = object.None
		if len(a) >= 2 {
			intro = a[1]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("intro"); ok2 {
				intro = v
			}
		}

		// preloop()
		if fn, err := i.getAttr(self, "preloop"); err == nil {
			if _, err2 := i.callObject(fn, nil, nil); err2 != nil {
				return nil, err2
			}
		}

		// Print intro.
		introStr := ""
		if intro != object.None {
			introStr = cmdStr(intro)
		} else {
			introStr = cmdGetAttrStr(self, "intro")
		}
		if introStr != "" {
			_ = cmdWrite(self, introStr+"\n")
		}

		var stop object.Object = object.None

		for {
			var line string

			// Pop from cmdqueue first.
			queueVal, err := i.getAttr(self, "cmdqueue")
			if err == nil {
				if q, ok2 := queueVal.(*object.List); ok2 && len(q.V) > 0 {
					line = cmdStr(q.V[0])
					q.V = q.V[1:]
					goto dispatched
				}
			}

			// Read from input.
			{
				useRaw, _ := i.getAttr(self, "use_rawinput")
				prompt := cmdGetAttrStr(self, "prompt")
				if object.Truthy(useRaw) {
					if inputFn, ok2 := i.Builtins.GetStr("input"); ok2 {
						res, err2 := i.callObject(inputFn, []object.Object{&object.Str{V: prompt}}, nil)
						if err2 != nil {
							line = "EOF"
						} else {
							line = cmdStr(res)
						}
					} else {
						line = "EOF"
					}
				} else {
					stdout, _ := i.getAttr(self, "stdout")
					if writeFn, err2 := i.getAttr(stdout, "write"); err2 == nil {
						_, _ = i.callObject(writeFn, []object.Object{&object.Str{V: prompt}}, nil)
					}
					stdinObj, _ := i.getAttr(self, "stdin")
					if readFn, err2 := i.getAttr(stdinObj, "readline"); err2 == nil {
						res, err3 := i.callObject(readFn, nil, nil)
						if err3 != nil || cmdStr(res) == "" {
							line = "EOF"
						} else {
							line = strings.TrimRight(cmdStr(res), "\r\n")
						}
					} else {
						line = "EOF"
					}
				}
			}

		dispatched:
			// precmd
			precmdFn, err := i.getAttr(self, "precmd")
			if err != nil {
				break
			}
			lineObj, err := i.callObject(precmdFn, []object.Object{&object.Str{V: line}}, nil)
			if err != nil {
				break
			}
			line = cmdStr(lineObj)

			// onecmd
			onecmdFn, err := i.getAttr(self, "onecmd")
			if err != nil {
				break
			}
			stop, err = i.callObject(onecmdFn, []object.Object{&object.Str{V: line}}, nil)
			if err != nil {
				break
			}

			// postcmd
			postcmdFn, err := i.getAttr(self, "postcmd")
			if err != nil {
				break
			}
			stop, err = i.callObject(postcmdFn, []object.Object{stop, &object.Str{V: line}}, nil)
			if err != nil {
				break
			}

			if object.Truthy(stop) {
				break
			}
		}

		// postloop
		if fn, err := i.getAttr(self, "postloop"); err == nil {
			_, _ = i.callObject(fn, nil, nil)
		}

		return object.None, nil
	}})

	m.Dict.SetStr("Cmd", cmdCls)
	return m
}
