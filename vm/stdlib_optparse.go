package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

// optparseOption holds metadata for one registered option.
type optparseOption struct {
	optStrings []string
	action     string
	typeName   string // "string", "int", "long", "float", "complex", "choice"
	dest       string
	default_   object.Object
	nargs      int // 1 default; 0 for no-arg actions
	const_     object.Object
	choices    []string
	help       string
	metavar    string
	callback   object.Object
}

// optparseGroup holds an OptionGroup.
type optparseGroup struct {
	title       string
	description string
	options     []*optparseOption
}

// optparseParser holds all state for one OptionParser instance.
type optparseParser struct {
	prog             string
	usage            string
	description      string
	epilog           string
	version          string
	addHelpOption    bool
	interspersedArgs bool
	options          []*optparseOption
	defaults         map[string]object.Object
	optionGroups     []*optparseGroup
}

// allOptions returns all options on the parser.
func (p *optparseParser) allOptions() []*optparseOption {
	return p.options
}

// optparseDest derives dest from option strings.
func optparseDest(optStrings []string) string {
	for _, s := range optStrings {
		if strings.HasPrefix(s, "--") {
			return strings.ReplaceAll(strings.TrimLeft(s, "-"), "-", "_")
		}
	}
	if len(optStrings) > 0 {
		return strings.TrimLeft(optStrings[0], "-")
	}
	return ""
}

// buildOptparse constructs the optparse module.
func (i *Interp) buildOptparse() *object.Module {
	m := &object.Module{Name: "optparse", Dict: object.NewDict()}

	// Constants
	m.Dict.SetStr("SUPPRESS_HELP", &object.Str{V: "SUPPRESS HELP"})
	m.Dict.SetStr("SUPPRESS_USAGE", &object.Str{V: "SUPPRESS USAGE"})
	noDefault := &object.Tuple{V: []object.Object{&object.Str{V: "NO"}, &object.Str{V: "DEFAULT"}}}
	m.Dict.SetStr("NO_DEFAULT", noDefault)

	// Exception classes
	optErrCls := &object.Class{Name: "OptionError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	optValErrCls := &object.Class{Name: "OptionValueError", Bases: []*object.Class{optErrCls}, Dict: object.NewDict()}
	badOptCls := &object.Class{Name: "BadOptionError", Bases: []*object.Class{optErrCls}, Dict: object.NewDict()}
	ambigOptCls := &object.Class{Name: "AmbiguousOptionError", Bases: []*object.Class{badOptCls}, Dict: object.NewDict()}
	m.Dict.SetStr("OptionError", optErrCls)
	m.Dict.SetStr("OptionValueError", optValErrCls)
	m.Dict.SetStr("BadOptionError", badOptCls)
	m.Dict.SetStr("AmbiguousOptionError", ambigOptCls)

	// Values class
	valuesCls := i.buildOptparseValues()
	m.Dict.SetStr("Values", valuesCls)

	// Option class
	optionCls := i.buildOptparseOption()
	m.Dict.SetStr("Option", optionCls)

	// make_option is an alias for Option(...)
	m.Dict.SetStr("make_option", &object.BuiltinFunc{Name: "make_option", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return i.callObject(optionCls, a, kw)
	}})

	// check_builtin stub
	m.Dict.SetStr("check_builtin", &object.BuiltinFunc{Name: "check_builtin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// OptionGroup class
	optionGroupCls := &object.Class{Name: "OptionGroup", Dict: object.NewDict()}
	optionGroupCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("OptionGroup", optionGroupCls)

	// OptionParser class
	optionParserCls := i.buildOptparseParserClass(valuesCls, optionCls, optionGroupCls)
	m.Dict.SetStr("OptionParser", optionParserCls)

	return m
}

// buildOptparseValues builds the Values class.
func (i *Interp) buildOptparseValues() *object.Class {
	cls := &object.Class{Name: "Values", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			if d, ok2 := a[1].(*object.Dict); ok2 {
				keys, vals := d.Items()
				for k, key := range keys {
					if ks, ok3 := key.(*object.Str); ok3 {
						self.Dict.SetStr(ks.V, vals[k])
					}
				}
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("ensure_value", &object.BuiltinFunc{Name: "ensure_value", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		attr := object.Str_(a[1])
		defVal := a[2]
		cur, exists := self.Dict.GetStr(attr)
		if !exists || cur == object.None {
			self.Dict.SetStr(attr, defVal)
			return defVal, nil
		}
		return cur, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Values()"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Values()"}, nil
		}
		var parts []string
		keys, vals := self.Dict.Items()
		for idx, k := range keys {
			if ks, ok2 := k.(*object.Str); ok2 {
				parts = append(parts, fmt.Sprintf("%s=%s", ks.V, object.Repr(vals[idx])))
			}
		}
		return &object.Str{V: "{" + strings.Join(parts, ", ") + "}"}, nil
	}})

	return cls
}

// buildOptparseOption builds the Option class.
// Each constructed instance stores no Go state — the state is tracked by the
// caller (buildOptparseParserClass) which extracts attrs from the instance dict.
func (i *Interp) buildOptparseOption() *object.Class {
	cls := &object.Class{Name: "Option", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// Positional args are option strings: '-f', '--foo'
		var optStrings []string
		for _, v := range a[1:] {
			s := object.Str_(v)
			if strings.HasPrefix(s, "-") {
				optStrings = append(optStrings, s)
			}
		}
		// Store in instance dict for extraction
		self.Dict.SetStr("_optstrings_list", optStringsToList(optStrings))
		// kwargs
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				if ks, ok2 := k.(*object.Str); ok2 {
					self.Dict.SetStr(ks.V, vals[idx])
				}
			}
		}
		// Derive dest if not given
		if _, hasDest := self.Dict.GetStr("dest"); !hasDest {
			dest := optparseDest(optStrings)
			if dest != "" {
				self.Dict.SetStr("dest", &object.Str{V: dest})
			}
		}
		return object.None, nil
	}})

	return cls
}

// optStringsToList converts a []string to an *object.List of *object.Str.
func optStringsToList(ss []string) *object.List {
	items := make([]object.Object, len(ss))
	for i, s := range ss {
		items[i] = &object.Str{V: s}
	}
	return &object.List{V: items}
}

// optionFromInstance extracts an optparseOption from an Option class instance.
func optionFromInstance(inst *object.Instance) *optparseOption {
	opt := &optparseOption{
		action:   "store",
		typeName: "string",
		nargs:    1,
	}
	// optstrings
	if v, ok := inst.Dict.GetStr("_optstrings_list"); ok {
		if lst, ok2 := v.(*object.List); ok2 {
			for _, elem := range lst.V {
				opt.optStrings = append(opt.optStrings, object.Str_(elem))
			}
		}
	}
	if v, ok := inst.Dict.GetStr("action"); ok {
		opt.action = object.Str_(v)
	}
	if v, ok := inst.Dict.GetStr("type"); ok {
		opt.typeName = object.Str_(v)
	}
	if v, ok := inst.Dict.GetStr("dest"); ok {
		opt.dest = object.Str_(v)
	}
	if v, ok := inst.Dict.GetStr("default"); ok {
		opt.default_ = v
	}
	if v, ok := inst.Dict.GetStr("nargs"); ok {
		if n, ok2 := v.(*object.Int); ok2 {
			opt.nargs = int(n.Int64())
		}
	}
	if v, ok := inst.Dict.GetStr("const"); ok {
		opt.const_ = v
	}
	if v, ok := inst.Dict.GetStr("choices"); ok {
		if lst, ok2 := v.(*object.List); ok2 {
			for _, c := range lst.V {
				opt.choices = append(opt.choices, object.Str_(c))
			}
		}
	}
	if v, ok := inst.Dict.GetStr("help"); ok {
		opt.help = object.Str_(v)
	}
	if v, ok := inst.Dict.GetStr("metavar"); ok {
		opt.metavar = object.Str_(v)
	}
	if v, ok := inst.Dict.GetStr("callback"); ok {
		opt.callback = v
	}
	// Adjust nargs for zero-arg actions
	switch opt.action {
	case "store_true", "store_false", "store_const", "count", "help", "version", "callback":
		opt.nargs = 0
	}
	// Derive dest if empty
	if opt.dest == "" {
		opt.dest = optparseDest(opt.optStrings)
	}
	return opt
}

// optparseParseAddOptionArgs constructs an optparseOption from add_option call args.
func optparseParseAddOptionArgs(interp *Interp, optionCls *object.Class, a []object.Object, kw *object.Dict) *optparseOption {
	// If first arg is an Option instance, extract from it
	if len(a) >= 1 {
		if inst, ok := a[0].(*object.Instance); ok && inst.Class == optionCls {
			return optionFromInstance(inst)
		}
	}
	// Construct from strings + kwargs
	opt := &optparseOption{
		action:   "store",
		typeName: "string",
		nargs:    1,
	}
	for _, v := range a {
		s := object.Str_(v)
		if strings.HasPrefix(s, "-") {
			opt.optStrings = append(opt.optStrings, s)
		}
	}
	if kw != nil {
		if v, ok2 := kw.GetStr("action"); ok2 {
			opt.action = object.Str_(v)
		}
		if v, ok2 := kw.GetStr("type"); ok2 {
			opt.typeName = object.Str_(v)
		}
		if v, ok2 := kw.GetStr("dest"); ok2 {
			opt.dest = object.Str_(v)
		}
		if v, ok2 := kw.GetStr("default"); ok2 {
			opt.default_ = v
		}
		if v, ok2 := kw.GetStr("nargs"); ok2 {
			if n, ok3 := v.(*object.Int); ok3 {
				opt.nargs = int(n.Int64())
			}
		}
		if v, ok2 := kw.GetStr("const"); ok2 {
			opt.const_ = v
		}
		if v, ok2 := kw.GetStr("choices"); ok2 {
			if lst, ok3 := v.(*object.List); ok3 {
				for _, c := range lst.V {
					opt.choices = append(opt.choices, object.Str_(c))
				}
			}
		}
		if v, ok2 := kw.GetStr("help"); ok2 {
			opt.help = object.Str_(v)
		}
		if v, ok2 := kw.GetStr("metavar"); ok2 {
			opt.metavar = object.Str_(v)
		}
		if v, ok2 := kw.GetStr("callback"); ok2 {
			opt.callback = v
		}
	}
	// Derive dest
	if opt.dest == "" {
		opt.dest = optparseDest(opt.optStrings)
	}
	// Adjust nargs for zero-arg actions
	switch opt.action {
	case "store_true", "store_false", "store_const", "count", "help", "version", "callback":
		opt.nargs = 0
	}
	return opt
}

// optparseGetDefaultValues builds a Values instance with all option defaults.
// Only the first explicit default for a given dest wins (earlier options win).
func optparseGetDefaultValues(parser *optparseParser, valuesCls *object.Class) *object.Instance {
	inst := &object.Instance{Class: valuesCls, Dict: object.NewDict()}
	for _, opt := range parser.allOptions() {
		if opt.dest == "" {
			continue
		}
		if opt.default_ != nil {
			// Has an explicit default — always set it (last explicit wins among options,
			// but set_defaults will override below anyway).
			inst.Dict.SetStr(opt.dest, opt.default_)
		} else {
			// No explicit default — only set None if dest not already in values.
			if _, exists := inst.Dict.GetStr(opt.dest); !exists {
				inst.Dict.SetStr(opt.dest, object.None)
			}
		}
	}
	// set_defaults overrides everything
	for k, v := range parser.defaults {
		inst.Dict.SetStr(k, v)
	}
	return inst
}

// buildOptparseParserClass builds the OptionParser class.
func (i *Interp) buildOptparseParserClass(valuesCls *object.Class, optionCls *object.Class, optionGroupCls *object.Class) *object.Class {
	cls := &object.Class{Name: "OptionParser", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		parser := &optparseParser{
			addHelpOption:    true,
			interspersedArgs: true,
			defaults:         map[string]object.Object{},
		}

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
			if v, ok2 := kw.GetStr("version"); ok2 {
				parser.version = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("add_help_option"); ok2 {
				parser.addHelpOption = object.Truthy(v)
			}
		}

		optparseInstallMethods(i, self, parser, valuesCls, optionCls, optionGroupCls)
		return object.None, nil
	}})

	return cls
}

// optparseInstallMethods installs all OptionParser instance methods as closures.
func optparseInstallMethods(interp *Interp, self *object.Instance, parser *optparseParser, valuesCls *object.Class, optionCls *object.Class, optionGroupCls *object.Class) {
	// add_option
	self.Dict.SetStr("add_option", &object.BuiltinFunc{Name: "add_option", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		opt := optparseParseAddOptionArgs(interp, optionCls, a, kw)
		parser.options = append(parser.options, opt)
		return object.None, nil
	}})

	// parse_args -> (values, args)
	self.Dict.SetStr("parse_args", &object.BuiltinFunc{Name: "parse_args", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		argList := optparseExtractArgList(interp, a, kw)
		var existingVals *object.Instance
		if len(a) > 1 {
			if inst, ok := a[1].(*object.Instance); ok {
				existingVals = inst
			}
		}
		values, positionals, err := optparseDoParse(interp, parser, argList, existingVals, valuesCls)
		if err != nil {
			return nil, err
		}
		posObjs := make([]object.Object, len(positionals))
		for idx, s := range positionals {
			posObjs[idx] = &object.Str{V: s}
		}
		return &object.Tuple{V: []object.Object{values, &object.List{V: posObjs}}}, nil
	}})

	// get_default_values
	self.Dict.SetStr("get_default_values", &object.BuiltinFunc{Name: "get_default_values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return optparseGetDefaultValues(parser, valuesCls), nil
	}})

	// set_defaults (kwargs only)
	self.Dict.SetStr("set_defaults", &object.BuiltinFunc{Name: "set_defaults", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				if ks, ok := k.(*object.Str); ok {
					parser.defaults[ks.V] = vals[idx]
				}
			}
		}
		return object.None, nil
	}})

	// has_option
	self.Dict.SetStr("has_option", &object.BuiltinFunc{Name: "has_option", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		target := object.Str_(a[0])
		for _, opt := range parser.allOptions() {
			for _, s := range opt.optStrings {
				if s == target {
					return object.True, nil
				}
			}
		}
		return object.False, nil
	}})

	// get_option -> Option instance or None
	self.Dict.SetStr("get_option", &object.BuiltinFunc{Name: "get_option", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		target := object.Str_(a[0])
		for _, opt := range parser.allOptions() {
			for _, s := range opt.optStrings {
				if s == target {
					// Return an Option instance representing this option
					inst := &object.Instance{Class: optionCls, Dict: object.NewDict()}
					inst.Dict.SetStr("dest", &object.Str{V: opt.dest})
					inst.Dict.SetStr("_optstrings_list", optStringsToList(opt.optStrings))
					return inst, nil
				}
			}
		}
		return object.None, nil
	}})

	// remove_option
	self.Dict.SetStr("remove_option", &object.BuiltinFunc{Name: "remove_option", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		target := object.Str_(a[0])
		var kept []*optparseOption
		for _, opt := range parser.options {
			found := false
			for _, s := range opt.optStrings {
				if s == target {
					found = true
					break
				}
			}
			if !found {
				kept = append(kept, opt)
			}
		}
		parser.options = kept
		return object.None, nil
	}})

	// add_option_group
	self.Dict.SetStr("add_option_group", &object.BuiltinFunc{Name: "add_option_group", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		title := ""
		description := ""
		if len(a) > 0 {
			title = object.Str_(a[0])
		}
		if len(a) > 1 {
			description = object.Str_(a[1])
		}
		_ = description

		grp := &optparseGroup{title: title, description: description}
		grpInst := &object.Instance{Class: optionGroupCls, Dict: object.NewDict()}
		grpInst.Dict.SetStr("title", &object.Str{V: title})

		// add_option on group also registers with parser
		grpInst.Dict.SetStr("add_option", &object.BuiltinFunc{Name: "add_option", Call: func(_ any, a2 []object.Object, kw2 *object.Dict) (object.Object, error) {
			opt := optparseParseAddOptionArgs(interp, optionCls, a2, kw2)
			grp.options = append(grp.options, opt)
			parser.options = append(parser.options, opt)
			return object.None, nil
		}})

		parser.optionGroups = append(parser.optionGroups, grp)
		return grpInst, nil
	}})

	// format_usage
	self.Dict.SetStr("format_usage", &object.BuiltinFunc{Name: "format_usage", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := optparseProg(parser)
		usage := parser.usage
		if usage == "" {
			usage = fmt.Sprintf("Usage: %s [options]\n", prog)
		} else {
			usage = strings.ReplaceAll(usage, "%prog", prog) + "\n"
		}
		return &object.Str{V: usage}, nil
	}})

	// format_help
	self.Dict.SetStr("format_help", &object.BuiltinFunc{Name: "format_help", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := optparseProg(parser)
		s := fmt.Sprintf("Usage: %s [options]\n", prog)
		if parser.description != "" {
			s += "\n" + parser.description + "\n"
		}
		s += "\nOptions:\n"
		for _, opt := range parser.options {
			optStr := strings.Join(opt.optStrings, ", ")
			if opt.help != "" {
				s += fmt.Sprintf("  %-20s %s\n", optStr, opt.help)
			} else {
				s += fmt.Sprintf("  %s\n", optStr)
			}
		}
		if parser.epilog != "" {
			s += "\n" + parser.epilog + "\n"
		}
		return &object.Str{V: s}, nil
	}})

	// print_help
	self.Dict.SetStr("print_help", &object.BuiltinFunc{Name: "print_help", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := optparseProg(parser)
		fmt.Fprintf(interp.Stdout, "Usage: %s [options]\n", prog)
		if parser.description != "" {
			fmt.Fprintf(interp.Stdout, "\n%s\n", parser.description)
		}
		return object.None, nil
	}})

	// print_usage
	self.Dict.SetStr("print_usage", &object.BuiltinFunc{Name: "print_usage", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		prog := optparseProg(parser)
		fmt.Fprintf(interp.Stdout, "Usage: %s [options]\n", prog)
		return object.None, nil
	}})

	// print_version
	self.Dict.SetStr("print_version", &object.BuiltinFunc{Name: "print_version", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if parser.version != "" {
			fmt.Fprintf(interp.Stdout, "%s\n", parser.version)
		}
		return object.None, nil
	}})

	// error
	self.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		msg := ""
		if len(a) > 0 {
			msg = object.Str_(a[0])
		}
		return nil, object.Errorf(interp.systemExit, "error: %s", msg)
	}})

	// get_prog_name
	self.Dict.SetStr("get_prog_name", &object.BuiltinFunc{Name: "get_prog_name", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: optparseProg(parser)}, nil
	}})

	// expand_prog_name
	self.Dict.SetStr("expand_prog_name", &object.BuiltinFunc{Name: "expand_prog_name", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		s := object.Str_(a[0])
		return &object.Str{V: strings.ReplaceAll(s, "%prog", optparseProg(parser))}, nil
	}})

	// get_description
	self.Dict.SetStr("get_description", &object.BuiltinFunc{Name: "get_description", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: parser.description}, nil
	}})

	// set_usage
	self.Dict.SetStr("set_usage", &object.BuiltinFunc{Name: "set_usage", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			parser.usage = object.Str_(a[0])
		}
		return object.None, nil
	}})

	// enable_interspersed_args
	self.Dict.SetStr("enable_interspersed_args", &object.BuiltinFunc{Name: "enable_interspersed_args", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		parser.interspersedArgs = true
		return object.None, nil
	}})

	// disable_interspersed_args
	self.Dict.SetStr("disable_interspersed_args", &object.BuiltinFunc{Name: "disable_interspersed_args", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		parser.interspersedArgs = false
		return object.None, nil
	}})
}

// optparseProg returns the prog string or a default.
func optparseProg(parser *optparseParser) string {
	if parser.prog != "" {
		return parser.prog
	}
	return "prog"
}

// optparseExtractArgList extracts the args list from call args.
func optparseExtractArgList(interp *Interp, a []object.Object, kw *object.Dict) []string {
	var argObj object.Object = object.None
	if len(a) > 0 {
		argObj = a[0]
	}
	if kw != nil {
		if v, ok := kw.GetStr("args"); ok {
			argObj = v
		}
	}
	if argObj == nil || argObj == object.None {
		if interp.Argv != nil && len(interp.Argv) > 1 {
			return interp.Argv[1:]
		}
		return nil
	}
	if lst, ok := argObj.(*object.List); ok {
		var result []string
		for _, elem := range lst.V {
			result = append(result, object.Str_(elem))
		}
		return result
	}
	return nil
}

// optparseDoParse implements the optparse parsing algorithm.
func optparseDoParse(interp *Interp, parser *optparseParser, args []string, existingVals *object.Instance, valuesCls *object.Class) (*object.Instance, []string, error) {
	// Build Values with defaults
	var values *object.Instance
	if existingVals != nil {
		values = existingVals
		// Apply missing defaults
		for _, opt := range parser.allOptions() {
			if opt.dest != "" {
				if _, exists := values.Dict.GetStr(opt.dest); !exists {
					def := opt.default_
					if def == nil {
						def = object.None
					}
					values.Dict.SetStr(opt.dest, def)
				}
			}
		}
		for k, v := range parser.defaults {
			values.Dict.SetStr(k, v)
		}
	} else {
		values = optparseGetDefaultValues(parser, valuesCls)
	}

	// Build lookup map: optStr -> option
	optByStr := map[string]*optparseOption{}
	for _, opt := range parser.allOptions() {
		for _, s := range opt.optStrings {
			optByStr[s] = opt
		}
	}

	var positionals []string
	stopProcessing := false
	idx := 0

	for idx < len(args) {
		arg := args[idx]

		if stopProcessing {
			positionals = append(positionals, arg)
			idx++
			continue
		}

		// -- stops option processing
		if arg == "--" {
			stopProcessing = true
			idx++
			continue
		}

		// Check if it looks like an option
		isOpt := len(arg) >= 2 && arg[0] == '-'

		if isOpt && strings.HasPrefix(arg, "--") {
			// Long option
			key := arg
			inlineVal := ""
			hasInline := false
			if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
				key = arg[:eqIdx]
				inlineVal = arg[eqIdx+1:]
				hasInline = true
			}
			opt, found := optByStr[key]
			if !found {
				// Unknown long option -> positional
				positionals = append(positionals, arg)
				idx++
				continue
			}
			idx++
			if opt.nargs == 0 {
				err := optparseApplyAction(interp, parser, opt, "", values)
				if err != nil {
					return nil, nil, err
				}
			} else if opt.nargs == 1 {
				val := ""
				if hasInline {
					val = inlineVal
				} else if idx < len(args) {
					val = args[idx]
					idx++
				}
				err := optparseApplyAction(interp, parser, opt, val, values)
				if err != nil {
					return nil, nil, err
				}
			} else {
				// nargs > 1: consume N values
				var rawVals []string
				for n := 0; n < opt.nargs && idx < len(args); n++ {
					rawVals = append(rawVals, args[idx])
					idx++
				}
				err := optparseApplyActionMulti(interp, parser, opt, rawVals, values)
				if err != nil {
					return nil, nil, err
				}
			}
		} else if isOpt {
			// Short option(s): -v or -vvv or -fVALUE
			chars := arg[1:] // everything after '-'
			ci := 0
			for ci < len(chars) {
				c := string(chars[ci])
				shortFlag := "-" + c
				opt, found := optByStr[shortFlag]
				if !found {
					// Unknown short option -> treat whole arg as positional
					positionals = append(positionals, arg)
					break
				}
				ci++
				if opt.nargs == 0 {
					err := optparseApplyAction(interp, parser, opt, "", values)
					if err != nil {
						return nil, nil, err
					}
				} else if opt.nargs == 1 {
					val := ""
					if ci < len(chars) {
						// -fVALUE: rest of chars is value
						val = chars[ci:]
						ci = len(chars) // consumed all remaining
					} else {
						// Next arg is value
						idx++
						if idx < len(args) {
							val = args[idx]
						}
					}
					err := optparseApplyAction(interp, parser, opt, val, values)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// nargs > 1: consume N args after current flag
					idx++ // advance past current -f arg
					var rawVals []string
					for n := 0; n < opt.nargs && idx < len(args); n++ {
						rawVals = append(rawVals, args[idx])
						idx++
					}
					err := optparseApplyActionMulti(interp, parser, opt, rawVals, values)
					if err != nil {
						return nil, nil, err
					}
					ci = len(chars) // done with this flag cluster
					idx-- // will be incremented at end of outer loop
				}
			}
			idx++
		} else {
			// Positional arg
			if !parser.interspersedArgs {
				// All remaining are positionals
				positionals = append(positionals, args[idx:]...)
				break
			}
			positionals = append(positionals, arg)
			idx++
		}
	}

	return values, positionals, nil
}

// optparseApplyAction applies a zero-or-one-value action.
func optparseApplyAction(interp *Interp, parser *optparseParser, opt *optparseOption, rawVal string, values *object.Instance) error {
	switch opt.action {
	case "store":
		converted, err := optparseConvertType(interp, parser, opt, rawVal)
		if err != nil {
			return err
		}
		values.Dict.SetStr(opt.dest, converted)
	case "store_true":
		values.Dict.SetStr(opt.dest, object.True)
	case "store_false":
		values.Dict.SetStr(opt.dest, object.False)
	case "store_const":
		c := opt.const_
		if c == nil {
			c = object.None
		}
		values.Dict.SetStr(opt.dest, c)
	case "append":
		converted, err := optparseConvertType(interp, parser, opt, rawVal)
		if err != nil {
			return err
		}
		cur, exists := values.Dict.GetStr(opt.dest)
		if !exists || cur == object.None {
			values.Dict.SetStr(opt.dest, &object.List{V: []object.Object{converted}})
		} else if lst, ok := cur.(*object.List); ok {
			values.Dict.SetStr(opt.dest, &object.List{V: append(lst.V, converted)})
		} else {
			values.Dict.SetStr(opt.dest, &object.List{V: []object.Object{converted}})
		}
	case "count":
		cur, exists := values.Dict.GetStr(opt.dest)
		if !exists || cur == object.None {
			values.Dict.SetStr(opt.dest, object.NewInt(1))
		} else if n, ok := cur.(*object.Int); ok && n.IsInt64() {
			values.Dict.SetStr(opt.dest, object.NewInt(n.Int64()+1))
		} else {
			values.Dict.SetStr(opt.dest, object.NewInt(1))
		}
	case "callback":
		if opt.callback != nil {
			_, _ = interp.callObject(opt.callback, []object.Object{
				object.None,
				&object.Str{V: rawVal},
				&object.Str{V: rawVal},
				object.None,
			}, nil)
		}
	case "help":
		prog := optparseProg(parser)
		fmt.Fprintf(interp.Stdout, "Usage: %s [options]\n", prog)
		exc := object.NewException(interp.systemExit, "")
		exc.Args = &object.Tuple{V: []object.Object{object.NewInt(0)}}
		return exc
	case "version":
		fmt.Fprintf(interp.Stdout, "%s\n", parser.version)
		exc := object.NewException(interp.systemExit, "")
		exc.Args = &object.Tuple{V: []object.Object{object.NewInt(0)}}
		return exc
	}
	return nil
}

// optparseApplyActionMulti applies a multi-value (nargs>1) action.
func optparseApplyActionMulti(interp *Interp, parser *optparseParser, opt *optparseOption, rawVals []string, values *object.Instance) error {
	items := make([]object.Object, len(rawVals))
	for k, s := range rawVals {
		converted, err := optparseConvertType(interp, parser, opt, s)
		if err != nil {
			return err
		}
		items[k] = converted
	}
	switch opt.action {
	case "store":
		values.Dict.SetStr(opt.dest, &object.Tuple{V: items})
	case "append":
		cur, exists := values.Dict.GetStr(opt.dest)
		if !exists || cur == object.None {
			values.Dict.SetStr(opt.dest, &object.List{V: []object.Object{&object.Tuple{V: items}}})
		} else if lst, ok := cur.(*object.List); ok {
			values.Dict.SetStr(opt.dest, &object.List{V: append(lst.V, &object.Tuple{V: items})})
		}
	}
	return nil
}

// optparseConvertType converts a raw string based on the option's typeName.
func optparseConvertType(interp *Interp, parser *optparseParser, opt *optparseOption, rawVal string) (object.Object, error) {
	switch opt.typeName {
	case "int", "long":
		intFn, ok := interp.Builtins.GetStr("int")
		if !ok {
			return &object.Str{V: rawVal}, nil
		}
		result, err := interp.callObject(intFn, []object.Object{&object.Str{V: rawVal}}, nil)
		if err != nil {
			return nil, object.Errorf(interp.runtimeErr, "option %s: invalid integer value: %q", strings.Join(opt.optStrings, "/"), rawVal)
		}
		return result, nil
	case "float":
		floatFn, ok := interp.Builtins.GetStr("float")
		if !ok {
			return &object.Str{V: rawVal}, nil
		}
		result, err := interp.callObject(floatFn, []object.Object{&object.Str{V: rawVal}}, nil)
		if err != nil {
			return nil, object.Errorf(interp.runtimeErr, "option %s: invalid float value: %q", strings.Join(opt.optStrings, "/"), rawVal)
		}
		return result, nil
	case "complex":
		complexFn, ok := interp.Builtins.GetStr("complex")
		if !ok {
			return &object.Str{V: rawVal}, nil
		}
		result, err := interp.callObject(complexFn, []object.Object{&object.Str{V: rawVal}}, nil)
		if err != nil {
			return nil, object.Errorf(interp.runtimeErr, "option %s: invalid complex value: %q", strings.Join(opt.optStrings, "/"), rawVal)
		}
		return result, nil
	case "choice":
		if len(opt.choices) > 0 {
			for _, c := range opt.choices {
				if c == rawVal {
					return &object.Str{V: rawVal}, nil
				}
			}
			return nil, object.Errorf(interp.runtimeErr, "option %s: invalid choice: %q (choose from %s)",
				strings.Join(opt.optStrings, "/"), rawVal, strings.Join(opt.choices, ", "))
		}
		return &object.Str{V: rawVal}, nil
	default:
		// "string" or unset: no conversion
		return &object.Str{V: rawVal}, nil
	}
}
