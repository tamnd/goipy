package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

// argparseAction holds metadata for one registered argument.
type argparseAction struct {
	dest         string
	optStrings   []string // e.g. ["-f", "--foo"]
	isPositional bool
	action       string        // store, store_true, store_false, store_const, append, count, help, version
	nargs        object.Object // None, "?", "*", "+", int, or REMAINDER
	const_       object.Object
	default_     object.Object
	type_        object.Object // callable or nil
	choices      object.Object // list or nil
	required     bool
	help         string
	metavar      string
	version      string
}

// argparseParser holds all state for one ArgumentParser instance.
type argparseParser struct {
	prog            string
	usage           string
	description     string
	epilog          string
	prefixChars     string
	addHelp         bool
	allowAbbrev     bool
	exitOnError     bool
	argumentDefault object.Object
	actions         []*argparseAction
	defaults        map[string]object.Object
	subparsers      *argparseSubparsers
}

// argparseSubparsers holds subparser state.
type argparseSubparsers struct {
	dest     string
	parsers  map[string]*argparseParser
	required bool
}

// buildArgparse constructs the argparse module.
func (i *Interp) buildArgparse() *object.Module {
	m := &object.Module{Name: "argparse", Dict: object.NewDict()}

	// --- Constants (CPython values) ---
	m.Dict.SetStr("SUPPRESS", &object.Str{V: "==SUPPRESS=="})
	m.Dict.SetStr("OPTIONAL", &object.Str{V: "?"})
	m.Dict.SetStr("ZERO_OR_MORE", &object.Str{V: "*"})
	m.Dict.SetStr("ONE_OR_MORE", &object.Str{V: "+"})
	m.Dict.SetStr("REMAINDER", &object.Str{V: "..."})
	m.Dict.SetStr("PARSER", &object.Str{V: "..."})

	// --- Exception classes ---
	argErrCls := &object.Class{Name: "ArgumentError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	argTypeErrCls := &object.Class{Name: "ArgumentTypeError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("ArgumentError", argErrCls)
	m.Dict.SetStr("ArgumentTypeError", argTypeErrCls)

	// --- Formatter stubs ---
	for _, name := range []string{
		"HelpFormatter",
		"RawDescriptionHelpFormatter",
		"RawTextHelpFormatter",
		"ArgumentDefaultsHelpFormatter",
		"MetavarTypeHelpFormatter",
	} {
		name := name
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		m.Dict.SetStr(name, cls)
	}

	// --- FileType ---
	fileTypeCls := &object.Class{Name: "FileType", Dict: object.NewDict()}
	fileTypeCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		mode := "r"
		if len(a) > 1 {
			if s, ok2 := a[1].(*object.Str); ok2 {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
		}
		self.Dict.SetStr("_mode", &object.Str{V: mode})
		return object.None, nil
	}})
	fileTypeCls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return &object.Str{V: fmt.Sprintf("<file %s>", object.Str_(a[1]))}, nil
	}})
	m.Dict.SetStr("FileType", fileTypeCls)

	// --- Namespace class ---
	namespaceCls := i.buildArgparseNamespace()
	m.Dict.SetStr("Namespace", namespaceCls)

	// --- ArgumentParser class ---
	parserCls := i.buildArgumentParserClass(namespaceCls)
	m.Dict.SetStr("ArgumentParser", parserCls)

	return m
}

// buildArgparseNamespace builds the Namespace class.
func (i *Interp) buildArgparseNamespace() *object.Class {
	cls := &object.Class{Name: "Namespace", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				if ks, ok2 := k.(*object.Str); ok2 {
					self.Dict.SetStr(ks.V, vals[idx])
				}
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Namespace()"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Namespace()"}, nil
		}
		var parts []string
		keys, vals := self.Dict.Items()
		for idx, k := range keys {
			if ks, ok2 := k.(*object.Str); ok2 {
				parts = append(parts, fmt.Sprintf("%s=%s", ks.V, object.Repr(vals[idx])))
			}
		}
		return &object.Str{V: "Namespace(" + strings.Join(parts, ", ") + ")"}, nil
	}})

	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self, ok1 := a[0].(*object.Instance)
		other, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return object.False, nil
		}
		// Compare all keys/vals
		sk, sv := self.Dict.Items()
		ok3, _ := other.Dict.Items()
		if len(sk) != len(ok3) {
			return object.False, nil
		}
		for idx, k := range sk {
			ks, ok3 := k.(*object.Str)
			if !ok3 {
				return object.False, nil
			}
			ov, found := other.Dict.GetStr(ks.V)
			if !found {
				return object.False, nil
			}
			eq, err := object.Eq(sv[idx], ov)
			if err != nil || !eq {
				return object.False, nil
			}
		}
		return object.True, nil
	}})

	cls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		key, ok2 := a[1].(*object.Str)
		if !ok2 {
			return object.False, nil
		}
		_, found := self.Dict.GetStr(key.V)
		return object.BoolOf(found), nil
	}})

	return cls
}

// buildArgumentParserClass builds the ArgumentParser class with closure-captured state.
func (i *Interp) buildArgumentParserClass(namespaceCls *object.Class) *object.Class {
	cls := &object.Class{Name: "ArgumentParser", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		// Create new parser state captured by all closures.
		parser := &argparseParser{
			prefixChars: "-",
			addHelp:     true,
			allowAbbrev: true,
			exitOnError: true,
			defaults:    map[string]object.Object{},
		}

		// Parse kwargs.
		if kw != nil {
			if v, ok2 := kw.GetStr("prog"); ok2 {
				parser.prog = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("usage"); ok2 {
				parser.usage = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("description"); ok2 {
				parser.description = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("epilog"); ok2 {
				parser.epilog = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("prefix_chars"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					parser.prefixChars = s.V
				}
			}
			if v, ok2 := kw.GetStr("add_help"); ok2 {
				parser.addHelp = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("allow_abbrev"); ok2 {
				parser.allowAbbrev = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("exit_on_error"); ok2 {
				parser.exitOnError = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("argument_default"); ok2 {
				parser.argumentDefault = v
			}
		}

		// Install instance methods as closures over parser.
		argparseInstallMethods(i, self, parser, namespaceCls)

		return object.None, nil
	}})

	return cls
}

// argparseInstallMethods sets all ArgumentParser methods on self.
// These BuiltinFuncs are stored in inst.Dict and called WITHOUT self prepended.
// All state is captured via closure over *argparseParser.
func argparseInstallMethods(interp *Interp, self *object.Instance, parser *argparseParser, namespaceCls *object.Class) {
	// add_argument: a = [name_or_flags...]
	self.Dict.SetStr("add_argument", &object.BuiltinFunc{Name: "add_argument", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return argparseAddArgument(parser, a, kw)
	}})

	// parse_args: a = [args_list?]
	self.Dict.SetStr("parse_args", &object.BuiltinFunc{Name: "parse_args", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		argList, ns := argparseExtractParseArgs(interp, a, kw)
		result, extra, err := argparseParse(interp, parser, argList, ns, namespaceCls)
		if err != nil {
			return nil, err
		}
		if len(extra) > 0 {
			return nil, object.Errorf(interp.runtimeErr, "unrecognized arguments: %s", strings.Join(extra, " "))
		}
		return result, nil
	}})

	// parse_known_args: a = [args_list?]
	self.Dict.SetStr("parse_known_args", &object.BuiltinFunc{Name: "parse_known_args", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		argList, ns := argparseExtractParseArgs(interp, a, kw)
		result, extra, err := argparseParse(interp, parser, argList, ns, namespaceCls)
		if err != nil {
			return nil, err
		}
		extraObjs := make([]object.Object, len(extra))
		for idx, s := range extra {
			extraObjs[idx] = &object.Str{V: s}
		}
		return &object.Tuple{V: []object.Object{result, &object.List{V: extraObjs}}}, nil
	}})

	// set_defaults: only kwargs
	self.Dict.SetStr("set_defaults", &object.BuiltinFunc{Name: "set_defaults", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				if ks, ok := k.(*object.Str); ok {
					parser.defaults[ks.V] = vals[idx]
					// Also update any action default.
					for _, act := range parser.actions {
						if act.dest == ks.V {
							act.default_ = vals[idx]
						}
					}
				}
			}
		}
		return object.None, nil
	}})

	// get_default: a = [dest]
	self.Dict.SetStr("get_default", &object.BuiltinFunc{Name: "get_default", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		dest := object.Str_(a[0])
		if v, ok := parser.defaults[dest]; ok {
			return v, nil
		}
		for _, act := range parser.actions {
			if act.dest == dest {
				if act.default_ != nil {
					return act.default_, nil
				}
				return object.None, nil
			}
		}
		return object.None, nil
	}})

	// add_argument_group: a = [title?]
	self.Dict.SetStr("add_argument_group", &object.BuiltinFunc{Name: "add_argument_group", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return argparseNewGroup(interp, parser), nil
	}})

	// add_mutually_exclusive_group: a = []
	self.Dict.SetStr("add_mutually_exclusive_group", &object.BuiltinFunc{Name: "add_mutually_exclusive_group", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return argparseNewGroup(interp, parser), nil
	}})

	// add_subparsers — stub returning a subparsers action group
	self.Dict.SetStr("add_subparsers", &object.BuiltinFunc{Name: "add_subparsers", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
		subInst := &object.Instance{Class: &object.Class{Name: "_SubParsersAction", Dict: object.NewDict()}, Dict: object.NewDict()}
		subInst.Dict.SetStr("add_parser", &object.BuiltinFunc{Name: "add_parser", Call: func(_ any, a2 []object.Object, kw2 *object.Dict) (object.Object, error) {
			subParser := &argparseParser{
				prefixChars: "-",
				addHelp:     true,
				allowAbbrev: true,
				exitOnError: true,
				defaults:    map[string]object.Object{},
			}
			subInst2 := &object.Instance{Class: &object.Class{Name: "ArgumentParser", Dict: object.NewDict()}, Dict: object.NewDict()}
			argparseInstallMethods(interp, subInst2, subParser, namespaceCls)
			return subInst2, nil
		}})
		return subInst, nil
	}})

	// format_usage: a = []
	self.Dict.SetStr("format_usage", &object.BuiltinFunc{Name: "format_usage", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := parser.prog
		if prog == "" {
			prog = "prog"
		}
		return &object.Str{V: "usage: " + prog + " [options]\n"}, nil
	}})

	// format_help: a = []
	self.Dict.SetStr("format_help", &object.BuiltinFunc{Name: "format_help", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := parser.prog
		if prog == "" {
			prog = "prog"
		}
		s := "usage: " + prog + " [options]\n"
		if parser.description != "" {
			s += "\n" + parser.description + "\n"
		}
		return &object.Str{V: s}, nil
	}})

	// print_usage: a = []
	self.Dict.SetStr("print_usage", &object.BuiltinFunc{Name: "print_usage", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := parser.prog
		if prog == "" {
			prog = "prog"
		}
		fmt.Fprintf(interp.Stdout, "usage: %s [options]\n", prog)
		return object.None, nil
	}})

	// print_help: a = []
	self.Dict.SetStr("print_help", &object.BuiltinFunc{Name: "print_help", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := parser.prog
		if prog == "" {
			prog = "prog"
		}
		fmt.Fprintf(interp.Stdout, "usage: %s [options]\n", prog)
		if parser.description != "" {
			fmt.Fprintf(interp.Stdout, "\n%s\n", parser.description)
		}
		return object.None, nil
	}})

	// error: a = [message]
	self.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		msg := ""
		if len(a) > 0 {
			msg = object.Str_(a[0])
		}
		return nil, object.Errorf(interp.systemExit, "error: %s", msg)
	}})

	// exit: a = [status?, message?]
	self.Dict.SetStr("exit", &object.BuiltinFunc{Name: "exit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		status := int64(0)
		if len(a) > 0 {
			if n, ok := a[0].(*object.Int); ok {
				status = n.Int64()
			}
		}
		exc := object.NewException(interp.systemExit, "")
		exc.Args = &object.Tuple{V: []object.Object{object.NewInt(status)}}
		return nil, exc
	}})
}

// argparseNewGroup creates a group whose add_argument delegates to parser.
// Methods stored in inst.Dict are called without self prepended.
func argparseNewGroup(interp *Interp, parser *argparseParser) *object.Instance {
	cls := &object.Class{Name: "_ArgumentGroup", Dict: object.NewDict()}
	grp := &object.Instance{Class: cls, Dict: object.NewDict()}
	grp.Dict.SetStr("add_argument", &object.BuiltinFunc{Name: "add_argument", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return argparseAddArgument(parser, a, kw)
	}})
	return grp
}

// argparseAddArgument registers a new argument on the parser.
func argparseAddArgument(parser *argparseParser, nameOrFlags []object.Object, kw *object.Dict) (object.Object, error) {
	act := &argparseAction{
		action: "store",
		nargs:  object.None,
	}

	// Collect name_or_flags
	var flags []string
	for _, f := range nameOrFlags {
		flags = append(flags, object.Str_(f))
	}
	if len(flags) == 0 {
		return object.None, nil
	}

	// Determine if positional or optional.
	prefixChars := parser.prefixChars
	isPositional := true
	for _, f := range flags {
		if len(f) > 0 && strings.ContainsRune(prefixChars, rune(f[0])) {
			isPositional = false
			break
		}
	}
	act.isPositional = isPositional
	act.optStrings = flags

	// Parse kwargs.
	if kw != nil {
		if v, ok := kw.GetStr("action"); ok {
			act.action = object.Str_(v)
		}
		if v, ok := kw.GetStr("nargs"); ok {
			act.nargs = v
		}
		if v, ok := kw.GetStr("const"); ok {
			act.const_ = v
		}
		if v, ok := kw.GetStr("default"); ok {
			act.default_ = v
		}
		if v, ok := kw.GetStr("type"); ok {
			act.type_ = v
		}
		if v, ok := kw.GetStr("choices"); ok {
			act.choices = v
		}
		if v, ok := kw.GetStr("required"); ok {
			act.required = object.Truthy(v)
		}
		if v, ok := kw.GetStr("help"); ok {
			act.help = object.Str_(v)
		}
		if v, ok := kw.GetStr("metavar"); ok {
			act.metavar = object.Str_(v)
		}
		if v, ok := kw.GetStr("dest"); ok {
			act.dest = object.Str_(v)
		}
		if v, ok := kw.GetStr("version"); ok {
			act.version = object.Str_(v)
		}
	}

	// Derive dest if not explicit.
	if act.dest == "" {
		if isPositional {
			act.dest = flags[0]
		} else {
			act.dest = argparseDerivedDest(flags, prefixChars)
		}
	}

	// Set smart defaults for action type.
	switch act.action {
	case "store_true":
		if act.default_ == nil {
			act.default_ = object.False
		}
		if act.const_ == nil {
			act.const_ = object.True
		}
		act.nargs = object.NewInt(0) // consumes no values
	case "store_false":
		if act.default_ == nil {
			act.default_ = object.True
		}
		if act.const_ == nil {
			act.const_ = object.False
		}
		act.nargs = object.NewInt(0)
	case "store_const":
		act.nargs = object.NewInt(0)
	case "count":
		if act.default_ == nil {
			act.default_ = object.NewInt(0)
		}
		act.nargs = object.NewInt(0)
	case "append":
		// nargs stays as-is (defaults to None = 1 value)
	}

	parser.actions = append(parser.actions, act)
	return object.None, nil
}

// argparseDerivedDest derives the dest from option strings.
func argparseDerivedDest(optStrings []string, prefixChars string) string {
	for _, s := range optStrings {
		if strings.HasPrefix(s, "--") {
			return strings.ReplaceAll(strings.TrimLeft(s, "-"), "-", "_")
		}
	}
	// fallback: first, strip prefix chars
	return strings.ReplaceAll(strings.TrimLeft(optStrings[0], prefixChars), "-", "_")
}

// argparseExtractParseArgs gets the args list and optional namespace from callsite args.
// a[0] = args list (optional), a[1] = namespace (optional). No self in a.
func argparseExtractParseArgs(interp *Interp, a []object.Object, kw *object.Dict) ([]string, *object.Instance) {
	var argList object.Object = object.None
	if len(a) > 0 {
		argList = a[0]
	}
	if kw != nil {
		if v, ok := kw.GetStr("args"); ok {
			argList = v
		}
	}

	var args []string
	if argList == nil || argList == object.None {
		// Use sys.argv[1:] or empty.
		if interp.Argv != nil && len(interp.Argv) > 1 {
			args = interp.Argv[1:]
		} else {
			args = []string{}
		}
	} else {
		// Expect a list.
		if lst, ok := argList.(*object.List); ok {
			for _, elem := range lst.V {
				args = append(args, object.Str_(elem))
			}
		}
	}

	var ns *object.Instance
	if len(a) > 1 {
		if inst, ok := a[1].(*object.Instance); ok {
			ns = inst
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr("namespace"); ok {
			if inst, ok2 := v.(*object.Instance); ok2 {
				ns = inst
			}
		}
	}

	return args, ns
}

// argparseParse implements the actual parsing algorithm.
func argparseParse(interp *Interp, parser *argparseParser, args []string, namespace *object.Instance, namespaceCls *object.Class) (*object.Instance, []string, error) {
	// Build namespace.
	ns := namespace
	if ns == nil {
		ns = &object.Instance{Class: namespaceCls, Dict: object.NewDict()}
	}

	// Set all defaults.
	for _, act := range parser.actions {
		if act.action == "help" || act.action == "version" {
			continue
		}
		def := act.default_
		if def == nil {
			def = object.None
		}
		// Only set if not already set (inherited namespace case).
		if _, exists := ns.Dict.GetStr(act.dest); !exists {
			ns.Dict.SetStr(act.dest, def)
		}
	}
	// Apply parser-level defaults (set_defaults).
	for k, v := range parser.defaults {
		ns.Dict.SetStr(k, v)
	}

	// Build map of optionals by string for fast lookup.
	optByStr := map[string]*argparseAction{}
	for _, act := range parser.actions {
		if !act.isPositional {
			for _, s := range act.optStrings {
				optByStr[s] = act
			}
		}
	}

	// Collect positionals in order.
	var positionals []*argparseAction
	for _, act := range parser.actions {
		if act.isPositional {
			positionals = append(positionals, act)
		}
	}

	remaining := []string{}
	positionalIdx := 0
	idx := 0

	for idx < len(args) {
		arg := args[idx]

		// End of options marker.
		if arg == "--" {
			idx++
			// Rest are positionals.
			for idx < len(args) {
				if positionalIdx < len(positionals) {
					act := positionals[positionalIdx]
					val, consumed := argparseConsumeNargs(args[idx:], act)
					if consumed == 0 {
						break
					}
					convVal, err := argparseConvertVal(interp, act, val)
					if err != nil {
						return nil, nil, err
					}
					ns.Dict.SetStr(act.dest, convVal)
					idx += consumed
					positionalIdx++
				} else {
					remaining = append(remaining, args[idx])
					idx++
				}
			}
			break
		}

		// Check if optional.
		isOpt := false
		if len(arg) > 1 && strings.ContainsRune(parser.prefixChars, rune(arg[0])) {
			// Could be optional — at least one prefix char.
			// Not if it's just a negative number like -3 and no matching action.
			isOpt = true
		}

		if isOpt && strings.HasPrefix(arg, "--") {
			// Long option.
			key := arg
			val := ""
			hasInlineVal := false
			if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
				key = arg[:eqIdx]
				val = arg[eqIdx+1:]
				hasInlineVal = true
			}
			act, found := optByStr[key]
			if !found && parser.allowAbbrev {
				act = argparseAbbrevMatch(key, optByStr)
				found = act != nil
			}
			if !found {
				remaining = append(remaining, arg)
				idx++
				continue
			}
			idx++
			switch act.action {
			case "store_true", "store_false":
				ns.Dict.SetStr(act.dest, act.const_)
			case "store_const":
				ns.Dict.SetStr(act.dest, act.const_)
			case "count":
				cur, _ := ns.Dict.GetStr(act.dest)
				ns.Dict.SetStr(act.dest, argparseIncrement(cur))
			case "help":
				// skip
			case "version":
				fmt.Fprintf(interp.Stdout, "%s\n", act.version)
			default:
				// Need a value.
				if hasInlineVal {
					convVal, err := argparseApplyType(interp, act.type_, val)
					if err != nil {
						return nil, nil, err
					}
					ns.Dict.SetStr(act.dest, argparseApplyNargsOpt(act, convVal, ns))
				} else {
					vals, consumed := argparseConsumeNargsOptional(act, args[idx:])
					convVal, err := argparseConvertVal(interp, act, vals)
					if err != nil {
						return nil, nil, err
					}
					ns.Dict.SetStr(act.dest, argparseApplyAppend(act, convVal, ns))
					idx += consumed
				}
			}
		} else if isOpt && len(arg) > 1 {
			// Short option(s): -v or -vvv or -f value
			chars := arg[1:] // everything after first prefix char
			consumed := 0
			allHandled := true
			for ci, c := range chars {
				shortFlag := string(parser.prefixChars[0]) + string(c)
				act, found := optByStr[shortFlag]
				if !found {
					// Try full arg if first char.
					if ci == 0 {
						// unknown option
						remaining = append(remaining, arg)
						consumed = 0
						allHandled = false
					}
					break
				}
				switch act.action {
				case "store_true", "store_false":
					ns.Dict.SetStr(act.dest, act.const_)
				case "store_const":
					ns.Dict.SetStr(act.dest, act.const_)
				case "count":
					cur, _ := ns.Dict.GetStr(act.dest)
					ns.Dict.SetStr(act.dest, argparseIncrement(cur))
				case "help":
					// skip
				case "version":
					fmt.Fprintf(interp.Stdout, "%s\n", act.version)
				default:
					// Consume rest of chars as value or next arg.
					rest := chars[ci+1:]
					if rest != "" {
						// -fVALUE
						convVal, err := argparseApplyType(interp, act.type_, rest)
						if err != nil {
							return nil, nil, err
						}
						ns.Dict.SetStr(act.dest, argparseApplyAppend(act, convVal, ns))
					} else {
						// Value is next arg.
						vals, c2 := argparseConsumeNargsOptional(act, args[idx+1:])
						convVal, err := argparseConvertVal(interp, act, vals)
						if err != nil {
							return nil, nil, err
						}
						ns.Dict.SetStr(act.dest, argparseApplyAppend(act, convVal, ns))
						consumed += c2
					}
					goto nextArg
				}
			}
			if allHandled {
				idx++
			} else {
				idx++
			}
			idx += consumed
			continue
		nextArg:
			idx++
			idx += consumed
		} else {
			// Positional.
			if positionalIdx < len(positionals) {
				act := positionals[positionalIdx]
				val, consumed := argparseConsumeNargs(args[idx:], act)
				if consumed == 0 {
					// nargs='*' or '?' with no match; already defaulted.
					positionalIdx++
					continue
				}
				convVal, err := argparseConvertVal(interp, act, val)
				if err != nil {
					return nil, nil, err
				}
				ns.Dict.SetStr(act.dest, convVal)
				idx += consumed
				positionalIdx++
			} else {
				remaining = append(remaining, arg)
				idx++
			}
		}
	}

	// Fill remaining positionals with defaults or empty lists.
	for positionalIdx < len(positionals) {
		act := positionals[positionalIdx]
		switch v := act.nargs.(type) {
		case *object.Str:
			switch v.V {
			case "*":
				if _, exists := ns.Dict.GetStr(act.dest); !exists {
					ns.Dict.SetStr(act.dest, &object.List{V: nil})
				} else {
					// Already has default; overwrite with empty if default is None.
					cur, _ := ns.Dict.GetStr(act.dest)
					if cur == object.None {
						ns.Dict.SetStr(act.dest, &object.List{V: nil})
					}
				}
			case "?":
				// Already has default.
			}
		}
		positionalIdx++
	}

	return ns, remaining, nil
}

// argparseConsumeNargs consumes args based on nargs setting, returns (raw value object, count consumed).
func argparseConsumeNargs(args []string, act *argparseAction) (object.Object, int) {
	switch v := act.nargs.(type) {
	case *object.NoneType:
		// None = consume exactly one.
		if len(args) == 0 {
			return act.default_, 0
		}
		return &object.Str{V: args[0]}, 1
	case *object.Int:
		n := v.Int64()
		if n == 0 {
			// store_true etc — no value consumed.
			return act.const_, 0
		}
		// Consume exactly n.
		if int64(len(args)) < n {
			return act.default_, 0
		}
		items := make([]object.Object, n)
		for j := int64(0); j < n; j++ {
			items[j] = &object.Str{V: args[j]}
		}
		return &object.List{V: items}, int(n)
	case *object.Str:
		switch v.V {
		case "?":
			if len(args) == 0 {
				return act.default_, 0
			}
			// For positional, consume one.
			return &object.Str{V: args[0]}, 1
		case "*":
			items := make([]object.Object, len(args))
			for j, s := range args {
				items[j] = &object.Str{V: s}
			}
			return &object.List{V: items}, len(args)
		case "+":
			if len(args) == 0 {
				return act.default_, 0
			}
			items := make([]object.Object, len(args))
			for j, s := range args {
				items[j] = &object.Str{V: s}
			}
			return &object.List{V: items}, len(args)
		}
	}
	// default: consume one.
	if len(args) == 0 {
		return act.default_, 0
	}
	return &object.Str{V: args[0]}, 1
}

// argparseConsumeNargsOptional consumes values for an optional argument.
// Returns (raw values as object, count consumed from args).
func argparseConsumeNargsOptional(act *argparseAction, args []string) (object.Object, int) {
	switch v := act.nargs.(type) {
	case *object.NoneType:
		// Consume one.
		if len(args) == 0 {
			return act.default_, 0
		}
		return &object.Str{V: args[0]}, 1
	case *object.Int:
		n := v.Int64()
		if n == 0 {
			return act.const_, 0
		}
		if int64(len(args)) < n {
			return act.default_, 0
		}
		items := make([]object.Object, n)
		for j := int64(0); j < n; j++ {
			items[j] = &object.Str{V: args[j]}
		}
		return &object.List{V: items}, int(n)
	case *object.Str:
		switch v.V {
		case "?":
			// For optional: if next arg is not an option, consume it; else use const.
			if len(args) == 0 {
				return act.const_, 0
			}
			next := args[0]
			if len(next) > 0 && next[0] == '-' {
				return act.const_, 0
			}
			return &object.Str{V: next}, 1
		case "*":
			// Consume until next option.
			var items []object.Object
			count := 0
			for _, s := range args {
				if len(s) > 0 && s[0] == '-' {
					break
				}
				items = append(items, &object.Str{V: s})
				count++
			}
			return &object.List{V: items}, count
		case "+":
			// Consume at least one until next option.
			var items []object.Object
			count := 0
			for _, s := range args {
				if len(s) > 0 && s[0] == '-' {
					break
				}
				items = append(items, &object.Str{V: s})
				count++
			}
			if count == 0 {
				return act.default_, 0
			}
			return &object.List{V: items}, count
		}
	}
	// default: consume one.
	if len(args) == 0 {
		return act.default_, 0
	}
	return &object.Str{V: args[0]}, 1
}

// argparseConvertVal applies type conversion to a parsed raw value.
func argparseConvertVal(interp *Interp, act *argparseAction, raw object.Object) (object.Object, error) {
	if raw == nil {
		return object.None, nil
	}
	switch v := raw.(type) {
	case *object.List:
		// Convert each element.
		result := make([]object.Object, len(v.V))
		for j, elem := range v.V {
			conv, err := argparseApplyType(interp, act.type_, object.Str_(elem))
			if err != nil {
				return nil, err
			}
			result[j] = conv
		}
		return &object.List{V: result}, nil
	case *object.Str:
		return argparseApplyType(interp, act.type_, v.V)
	default:
		return raw, nil
	}
}

// argparseApplyType applies a type callable to a string value.
func argparseApplyType(interp *Interp, typeObj object.Object, val string) (object.Object, error) {
	if typeObj == nil || typeObj == object.None {
		return &object.Str{V: val}, nil
	}
	return interp.callObject(typeObj, []object.Object{&object.Str{V: val}}, nil)
}

// argparseApplyNargsOpt handles nargs='?' for optional: single value.
func argparseApplyNargsOpt(act *argparseAction, val object.Object, ns *object.Instance) object.Object {
	if act.action == "append" {
		cur, exists := ns.Dict.GetStr(act.dest)
		if !exists || cur == object.None {
			return &object.List{V: []object.Object{val}}
		}
		if lst, ok := cur.(*object.List); ok {
			return &object.List{V: append(lst.V, val)}
		}
		return &object.List{V: []object.Object{val}}
	}
	return val
}

// argparseApplyAppend handles append action.
func argparseApplyAppend(act *argparseAction, val object.Object, ns *object.Instance) object.Object {
	if act.action == "append" {
		cur, exists := ns.Dict.GetStr(act.dest)
		if !exists || cur == object.None {
			return &object.List{V: []object.Object{val}}
		}
		if lst, ok := cur.(*object.List); ok {
			return &object.List{V: append(lst.V, val)}
		}
		return &object.List{V: []object.Object{val}}
	}
	return val
}

// argparseIncrement increments a count value.
func argparseIncrement(cur object.Object) *object.Int {
	if n, ok := cur.(*object.Int); ok && n.IsInt64() {
		return object.NewInt(n.Int64() + 1)
	}
	return object.NewInt(1)
}

// argparseAbbrevMatch finds a unique action matching an abbreviated long option.
func argparseAbbrevMatch(prefix string, optByStr map[string]*argparseAction) *argparseAction {
	var matches []*argparseAction
	for k, act := range optByStr {
		if strings.HasPrefix(k, prefix) {
			matches = append(matches, act)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return nil
}
