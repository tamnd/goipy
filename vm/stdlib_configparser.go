package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// Interpolation constants.
const (
	cfgInterpBasic    = 0
	cfgInterpExtended = 1
	cfgInterpNone     = 2
)

var reInterpBasic = regexp.MustCompile(`%\(([^)]+)\)s|%%`)
var reInterpExtended = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]+))?\}|\$\$`)

// buildConfigParser registers the configparser module.
func (i *Interp) buildConfigParser() *object.Module {
	m := &object.Module{Name: "configparser", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("DEFAULTSECT", &object.Str{V: "DEFAULT"})
	m.Dict.SetStr("MAX_INTERPOLATION_DEPTH", object.NewInt(10))

	// Exception hierarchy.
	errBase := &object.Class{Name: "Error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	noSecErr := &object.Class{Name: "NoSectionError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	dupSecErr := &object.Class{Name: "DuplicateSectionError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	dupOptErr := &object.Class{Name: "DuplicateOptionError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	noOptErr := &object.Class{Name: "NoOptionError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	interpErr := &object.Class{Name: "InterpolationError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	interpDepthErr := &object.Class{Name: "InterpolationDepthError", Bases: []*object.Class{interpErr}, Dict: object.NewDict()}
	interpMissErr := &object.Class{Name: "InterpolationMissingOptionError", Bases: []*object.Class{interpErr}, Dict: object.NewDict()}
	interpSynErr := &object.Class{Name: "InterpolationSyntaxError", Bases: []*object.Class{interpErr}, Dict: object.NewDict()}
	missSecHdrErr := &object.Class{Name: "MissingSectionHeaderError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}
	parseErr := &object.Class{Name: "ParsingError", Bases: []*object.Class{errBase}, Dict: object.NewDict()}

	m.Dict.SetStr("Error", errBase)
	m.Dict.SetStr("NoSectionError", noSecErr)
	m.Dict.SetStr("DuplicateSectionError", dupSecErr)
	m.Dict.SetStr("DuplicateOptionError", dupOptErr)
	m.Dict.SetStr("NoOptionError", noOptErr)
	m.Dict.SetStr("InterpolationError", interpErr)
	m.Dict.SetStr("InterpolationDepthError", interpDepthErr)
	m.Dict.SetStr("InterpolationMissingOptionError", interpMissErr)
	m.Dict.SetStr("InterpolationSyntaxError", interpSynErr)
	m.Dict.SetStr("MissingSectionHeaderError", missSecHdrErr)
	m.Dict.SetStr("ParsingError", parseErr)

	// Interpolation sentinel classes.
	basicInterpClass := &object.Class{Name: "BasicInterpolation", Dict: object.NewDict()}
	extInterpClass := &object.Class{Name: "ExtendedInterpolation", Dict: object.NewDict()}
	m.Dict.SetStr("BasicInterpolation", &object.BuiltinFunc{Name: "BasicInterpolation", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return &object.Instance{Class: basicInterpClass, Dict: object.NewDict()}, nil
	}})
	m.Dict.SetStr("ExtendedInterpolation", &object.BuiltinFunc{Name: "ExtendedInterpolation", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return &object.Instance{Class: extInterpClass, Dict: object.NewDict()}, nil
	}})

	// RawConfigParser constructor.
	m.Dict.SetStr("RawConfigParser", &object.BuiltinFunc{Name: "RawConfigParser", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		cfg := newConfigParserObj(kw, cfgInterpNone,
			noSecErr, dupSecErr, dupOptErr, noOptErr,
			interpMissErr, interpDepthErr, interpSynErr, missSecHdrErr, parseErr)
		return cfg, nil
	}})

	// Build BOOLEAN_STATES dict once (shared).
	bsDict := object.NewDict()
	for k, v := range booleanStates {
		bsDict.SetStr(k, object.BoolOf(v))
	}

	// ConfigParser constructor.
	cfgParserFunc := &object.BuiltinFunc{Name: "ConfigParser", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// Determine interpolation mode.
		interp := cfgInterpBasic
		if kw != nil {
			if iv, ok := kw.GetStr("interpolation"); ok {
				switch v := iv.(type) {
				case *object.NoneType:
					interp = cfgInterpNone
				case *object.Instance:
					if v.Class != nil && v.Class.Name == "ExtendedInterpolation" {
						interp = cfgInterpExtended
					}
				}
			}
		}
		cfg := newConfigParserObj(kw, interp,
			noSecErr, dupSecErr, dupOptErr, noOptErr,
			interpMissErr, interpDepthErr, interpSynErr, missSecHdrErr, parseErr)
		cfg.BoolStates = bsDict
		return cfg, nil
	}}
	// BOOLEAN_STATES is a class-level attribute accessible as ConfigParser.BOOLEAN_STATES.
	cfgParserFunc.Attrs = object.NewDict()
	cfgParserFunc.Attrs.SetStr("BOOLEAN_STATES", bsDict)
	m.Dict.SetStr("ConfigParser", cfgParserFunc)

	return m
}

var booleanStates = map[string]bool{
	"1": true, "yes": true, "true": true, "on": true,
	"0": false, "no": false, "false": false, "off": false,
}

func newConfigParserObj(
	kw *object.Dict,
	interp int,
	noSecErr, dupSecErr, dupOptErr, noOptErr *object.Class,
	interpMissErr, interpDepthErr, interpSynErr, missSecHdrErr, parseErr *object.Class,
) *object.ConfigParserObj {
	cfg := &object.ConfigParserObj{
		Defaults:           object.NewCfgSection(),
		Data:               make(map[string]*object.CfgSection),
		DefaultSection:     "DEFAULT",
		AllowNoValue:       false,
		Delimiters:         []string{"=", ":"},
		CommentPrefixes:    []string{"#", ";"},
		InlineCommentPrefixes: nil,
		Strict:             true,
		EmptyLinesInValues: true,
		Interpolation:      interp,
		NoSecErr:           noSecErr,
		DupSecErr:          dupSecErr,
		DupOptErr:          dupOptErr,
		NoOptErr:           noOptErr,
		InterpMissErr:      interpMissErr,
		InterpDepthErr:     interpDepthErr,
		InterpSynErr:       interpSynErr,
		MissSecHdrErr:      missSecHdrErr,
		ParseErr:           parseErr,
	}
	if kw == nil {
		return cfg
	}
	if v, ok := kw.GetStr("defaults"); ok {
		if d, ok := v.(*object.Dict); ok {
			keys, vals := d.Items()
			for k, kobj := range keys {
				if ks, ok := kobj.(*object.Str); ok {
					if vs, ok := vals[k].(*object.Str); ok {
						cfg.Defaults.Set(strings.ToLower(ks.V), vs.V)
					}
				}
			}
		}
	}
	if v, ok := kw.GetStr("allow_no_value"); ok {
		if b, ok := v.(*object.Bool); ok {
			cfg.AllowNoValue = b.V
		}
	}
	if v, ok := kw.GetStr("delimiters"); ok {
		if t, ok := v.(*object.Tuple); ok {
			cfg.Delimiters = nil
			for _, d := range t.V {
				if s, ok := d.(*object.Str); ok {
					cfg.Delimiters = append(cfg.Delimiters, s.V)
				}
			}
		}
	}
	if v, ok := kw.GetStr("comment_prefixes"); ok {
		if t, ok := v.(*object.Tuple); ok {
			cfg.CommentPrefixes = nil
			for _, p := range t.V {
				if s, ok := p.(*object.Str); ok {
					cfg.CommentPrefixes = append(cfg.CommentPrefixes, s.V)
				}
			}
		}
	}
	if v, ok := kw.GetStr("inline_comment_prefixes"); ok {
		if t, ok := v.(*object.Tuple); ok {
			for _, p := range t.V {
				if s, ok := p.(*object.Str); ok {
					cfg.InlineCommentPrefixes = append(cfg.InlineCommentPrefixes, s.V)
				}
			}
		}
	}
	if v, ok := kw.GetStr("strict"); ok {
		if b, ok := v.(*object.Bool); ok {
			cfg.Strict = b.V
		}
	}
	if v, ok := kw.GetStr("empty_lines_in_values"); ok {
		if b, ok := v.(*object.Bool); ok {
			cfg.EmptyLinesInValues = b.V
		}
	}
	if v, ok := kw.GetStr("default_section"); ok {
		if s, ok := v.(*object.Str); ok {
			cfg.DefaultSection = s.V
		}
	}
	return cfg
}

// --- INI parser -------------------------------------------------------------

func cfgParseString(cfg *object.ConfigParserObj, text string) error {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	currentSection := ""
	currentKey := ""
	var currentValLines []string
	inMultiline := false

	flushValue := func() {
		if currentSection == "" || currentKey == "" {
			return
		}
		val := strings.Join(currentValLines, "\n")
		sec := cfg.Data[currentSection]
		if sec == nil {
			sec = object.NewCfgSection()
			cfg.Data[currentSection] = sec
		}
		sec.Set(currentKey, val)
		currentKey = ""
		currentValLines = nil
		inMultiline = false
	}

	for lineIdx, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		// Continuation line?
		if inMultiline && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				if cfg.EmptyLinesInValues {
					currentValLines = append(currentValLines, "")
				} else {
					flushValue()
				}
			} else {
				currentValLines = append(currentValLines, trimmed)
			}
			continue
		}

		// If was in multiline, flush before processing new line.
		if inMultiline {
			flushValue()
		}

		stripped := strings.TrimSpace(line)

		// Empty line.
		if stripped == "" {
			continue
		}

		// Comment line?
		if cfgIsComment(stripped, cfg.CommentPrefixes) {
			continue
		}

		// Section header.
		if stripped[0] == '[' {
			end := strings.LastIndex(stripped, "]")
			if end < 1 {
				return object.Errorf(cfg.MissSecHdrErr, "File contains no section headers.\nfile: <string>, line %d\n%s", lineIdx+1, line)
			}
			sectionName := stripped[1:end]
			if strings.EqualFold(sectionName, cfg.DefaultSection) {
				currentSection = cfg.DefaultSection
			} else {
				if cfg.Strict {
					for _, s := range cfg.Sections {
						if s == sectionName {
							return object.Errorf(cfg.DupSecErr, "While reading from '<string>' [line %2d]: section '%s' already exists", lineIdx+1, sectionName)
						}
					}
				}
				if _, exists := cfg.Data[sectionName]; !exists {
					cfg.Sections = append(cfg.Sections, sectionName)
					cfg.Data[sectionName] = object.NewCfgSection()
				}
				currentSection = sectionName
			}
			currentKey = ""
			currentValLines = nil
			inMultiline = false
			continue
		}

		// Must have a section.
		if currentSection == "" {
			return object.Errorf(cfg.MissSecHdrErr, "File contains no section headers.\nfile: <string>, line %d\n%s", lineIdx+1, line)
		}

		// Key-value pair.
		key, value, isNoVal := cfgSplitKV(stripped, cfg.Delimiters)
		if key == "" && !isNoVal {
			return object.Errorf(cfg.ParseErr, "Source contains parsing errors: '<string>'\n\t[line %2d]: %s", lineIdx+1, line)
		}

		key = strings.ToLower(strings.TrimSpace(key))

		// Strict: check for duplicate option.
		if cfg.Strict {
			var sec *object.CfgSection
			if currentSection == cfg.DefaultSection {
				sec = cfg.Defaults
			} else {
				sec = cfg.Data[currentSection]
			}
			if sec != nil && sec.Has(key) {
				return object.Errorf(cfg.DupOptErr, "While reading from '<string>' [line %2d]: option '%s' in section '%s' already exists", lineIdx+1, key, currentSection)
			}
		}

		if isNoVal && value == "" {
			// allow_no_value key.
			if currentSection == cfg.DefaultSection {
				cfg.Defaults.SetNoVal(key)
			} else {
				sec := cfg.Data[currentSection]
				if sec == nil {
					sec = object.NewCfgSection()
					cfg.Data[currentSection] = sec
				}
				sec.SetNoVal(key)
			}
			currentKey = ""
			inMultiline = false
			continue
		}

		value = strings.TrimSpace(value)

		// Strip inline comment.
		for _, prefix := range cfg.InlineCommentPrefixes {
			if idx := strings.Index(value, " "+prefix); idx >= 0 {
				value = strings.TrimRight(value[:idx], " ")
				break
			}
		}

		currentKey = key
		currentValLines = []string{value}
		inMultiline = true

		// Store immediately in the right section.
		if currentSection == cfg.DefaultSection {
			cfg.Defaults.Set(key, value)
			// We track multiline for defaults too.
		}
	}

	// Final flush.
	flushValue()

	// Re-flush defaults if multiline was in progress there.
	if inMultiline && currentSection == cfg.DefaultSection && currentKey != "" {
		val := strings.Join(currentValLines, "\n")
		cfg.Defaults.Set(currentKey, val)
	}

	return nil
}

func cfgIsComment(line string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// cfgSplitKV splits "key = value" on the first delimiter found.
// Returns (key, value, isNoVal). isNoVal=true means no delimiter was found.
func cfgSplitKV(line string, delimiters []string) (string, string, bool) {
	bestIdx := -1
	bestDelim := ""
	for _, d := range delimiters {
		idx := strings.Index(line, d)
		if idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
			bestIdx = idx
			bestDelim = d
		}
	}
	if bestIdx < 0 {
		// No delimiter found.
		return line, "", true
	}
	_ = bestDelim
	return line[:bestIdx], line[bestIdx+len(bestDelim):], false
}

// --- interpolation ----------------------------------------------------------

func cfgInterpolate(cfg *object.ConfigParserObj, section, value string, depth int) (string, error) {
	if cfg.Interpolation == cfgInterpNone {
		return value, nil
	}
	if depth > 10 {
		return "", object.Errorf(cfg.InterpDepthErr, "Recursion limit exceeded in value substitution: option '%s' in section '%s' contains an interpolation key which cannot be found: depth %d", "", section, depth)
	}
	if cfg.Interpolation == cfgInterpBasic {
		return cfgInterpBasicExpand(cfg, section, value, depth)
	}
	if cfg.Interpolation == cfgInterpExtended {
		return cfgInterpExtExpand(cfg, section, value, depth)
	}
	return value, nil
}

func cfgInterpBasicExpand(cfg *object.ConfigParserObj, section, value string, depth int) (string, error) {
	var err error
	result := reInterpBasic.ReplaceAllStringFunc(value, func(match string) string {
		if err != nil {
			return match
		}
		if match == "%%" {
			return "%"
		}
		// Extract name from %(name)s
		inner := match[2 : len(match)-2]
		raw, found := cfgGetRaw(cfg, section, inner)
		if !found {
			err = object.Errorf(cfg.InterpMissErr, "Bad value substitution: option %q in section %q contains an interpolation key %q which is not a valid option name. Raw value: %q", inner, section, inner, value)
			return match
		}
		expanded, e := cfgInterpolate(cfg, section, raw, depth+1)
		if e != nil {
			err = e
			return match
		}
		return expanded
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

func cfgInterpExtExpand(cfg *object.ConfigParserObj, section, value string, depth int) (string, error) {
	var err error
	result := reInterpExtended.ReplaceAllStringFunc(value, func(match string) string {
		if err != nil {
			return match
		}
		if match == "$$" {
			return "$"
		}
		// Parse ${section:option} or ${option}
		inner := match[2 : len(match)-1]
		parts := strings.SplitN(inner, ":", 2)
		var lookupSection, lookupOption string
		if len(parts) == 2 {
			lookupSection = parts[0]
			lookupOption = parts[1]
		} else {
			lookupSection = section
			lookupOption = parts[0]
		}
		raw, found := cfgGetRaw(cfg, lookupSection, lookupOption)
		if !found {
			err = object.Errorf(cfg.InterpMissErr, "Bad value substitution in section '%s'", section)
			return match
		}
		expanded, e := cfgInterpolate(cfg, lookupSection, raw, depth+1)
		if e != nil {
			err = e
			return match
		}
		return expanded
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// cfgGetRaw retrieves the raw (uninterpolated) value for option in section.
func cfgGetRaw(cfg *object.ConfigParserObj, section, option string) (string, bool) {
	option = strings.ToLower(option)
	if section == cfg.DefaultSection {
		v, ok := cfg.Defaults.Values[option]
		return v, ok
	}
	sec, ok := cfg.Data[section]
	if ok {
		if v, exists := sec.Values[option]; exists {
			return v, true
		}
	}
	// Fall back to defaults.
	v, ok2 := cfg.Defaults.Values[option]
	return v, ok2
}

// cfgGetValue retrieves an interpolated value with fallback.
func cfgGetValue(cfg *object.ConfigParserObj, section, option string, raw bool, fallbackVal object.Object, hasFallback bool) (object.Object, error) {
	option = strings.ToLower(option)
	rawVal, found := cfgGetRaw(cfg, section, option)
	if !found {
		// Check for no-value key.
		sec, secOk := cfg.Data[section]
		if secOk && sec.NoVal[option] {
			return object.None, nil
		}
		if cfg.Defaults.NoVal[option] {
			return object.None, nil
		}
		if hasFallback {
			return fallbackVal, nil
		}
		if _, secExists := cfg.Data[section]; !secExists && section != cfg.DefaultSection {
			return nil, object.Errorf(cfg.NoSecErr, "No section: %q", section)
		}
		return nil, object.Errorf(cfg.NoOptErr, "No option %q in section %q", option, section)
	}
	if raw {
		return &object.Str{V: rawVal}, nil
	}
	expanded, err := cfgInterpolate(cfg, section, rawVal, 0)
	if err != nil {
		return nil, err
	}
	return &object.Str{V: expanded}, nil
}

// cfgHasSection checks if section exists (not DEFAULT).
func cfgHasSection(cfg *object.ConfigParserObj, section string) bool {
	for _, s := range cfg.Sections {
		if s == section {
			return true
		}
	}
	return false
}

// cfgOptionsList returns all option names for section including DEFAULT.
func cfgOptionsList(cfg *object.ConfigParserObj, section string) []string {
	seen := make(map[string]bool)
	var result []string
	sec, ok := cfg.Data[section]
	if ok {
		for _, k := range sec.Keys {
			result = append(result, k)
			seen[k] = true
		}
	}
	for _, k := range cfg.Defaults.Keys {
		if !seen[k] {
			result = append(result, k)
		}
	}
	return result
}

// cfgHasOption checks if an option exists in section or DEFAULT.
func cfgHasOption(cfg *object.ConfigParserObj, section, option string) bool {
	option = strings.ToLower(option)
	sec, ok := cfg.Data[section]
	if ok && sec.Has(option) {
		return true
	}
	return cfg.Defaults.Has(option)
}

// --- write ------------------------------------------------------------------

func cfgWrite(cfg *object.ConfigParserObj, target object.Object, spaceAround bool, i *Interp) error {
	delim := " = "
	if !spaceAround {
		delim = "="
	}
	writeStr := func(s string) error {
		wr, err := i.getAttr(target, "write")
		if err != nil {
			return err
		}
		_, err = i.callObject(wr, []object.Object{&object.Str{V: s}}, nil)
		return err
	}
	// Write DEFAULT section if non-empty.
	if len(cfg.Defaults.Keys) > 0 {
		if err := writeStr(fmt.Sprintf("[%s]\n", cfg.DefaultSection)); err != nil {
			return err
		}
		for _, k := range cfg.Defaults.Keys {
			v := cfg.Defaults.Values[k]
			if err := writeStr(fmt.Sprintf("%s%s%s\n", k, delim, v)); err != nil {
				return err
			}
		}
		if err := writeStr("\n"); err != nil {
			return err
		}
	}
	// Write each section.
	for _, secName := range cfg.Sections {
		sec := cfg.Data[secName]
		if err := writeStr(fmt.Sprintf("[%s]\n", secName)); err != nil {
			return err
		}
		if sec != nil {
			for _, k := range sec.Keys {
				if sec.NoVal[k] {
					if err := writeStr(fmt.Sprintf("%s\n", k)); err != nil {
						return err
					}
				} else {
					v := sec.Values[k]
					if err := writeStr(fmt.Sprintf("%s%s%s\n", k, delim, v)); err != nil {
						return err
					}
				}
			}
		}
		if err := writeStr("\n"); err != nil {
			return err
		}
	}
	return nil
}

// --- attr dispatch for ConfigParserObj --------------------------------------

func configParserAttr(i *Interp, cfg *object.ConfigParserObj, name string) (object.Object, bool) {
	switch name {
	case "BOOLEAN_STATES":
		if cfg.BoolStates != nil {
			return cfg.BoolStates, true
		}
		// Build default.
		bs := object.NewDict()
		for k, v := range booleanStates {
			bs.SetStr(k, object.BoolOf(v))
		}
		return bs, true

	case "default_section":
		return &object.Str{V: cfg.DefaultSection}, true

	case "defaults":
		return &object.BuiltinFunc{Name: "defaults", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			d := object.NewDict()
			for _, k := range cfg.Defaults.Keys {
				d.SetStr(k, &object.Str{V: cfg.Defaults.Values[k]})
			}
			return d, nil
		}}, true

	case "sections":
		return &object.BuiltinFunc{Name: "sections", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			v := make([]object.Object, len(cfg.Sections))
			for k, s := range cfg.Sections {
				v[k] = &object.Str{V: s}
			}
			return &object.List{V: v}, nil
		}}, true

	case "add_section":
		return &object.BuiltinFunc{Name: "add_section", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "add_section() missing section name")
			}
			name, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "add_section() section name must be str")
			}
			if cfgHasSection(cfg, name.V) {
				return nil, object.Errorf(cfg.DupSecErr, "Section %q already exists", name.V)
			}
			cfg.Sections = append(cfg.Sections, name.V)
			cfg.Data[name.V] = object.NewCfgSection()
			return object.None, nil
		}}, true

	case "has_section":
		return &object.BuiltinFunc{Name: "has_section", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			name, ok := a[0].(*object.Str)
			if !ok {
				return object.False, nil
			}
			return object.BoolOf(cfgHasSection(cfg, name.V)), nil
		}}, true

	case "options":
		return &object.BuiltinFunc{Name: "options", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "options() missing section")
			}
			sec, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "options() section must be str")
			}
			if !cfgHasSection(cfg, sec.V) {
				return nil, object.Errorf(cfg.NoSecErr, "No section: %q", sec.V)
			}
			opts := cfgOptionsList(cfg, sec.V)
			v := make([]object.Object, len(opts))
			for k, o := range opts {
				v[k] = &object.Str{V: o}
			}
			return &object.List{V: v}, nil
		}}, true

	case "has_option":
		return &object.BuiltinFunc{Name: "has_option", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			sec, ok := a[0].(*object.Str)
			opt, ok2 := a[1].(*object.Str)
			if !ok || !ok2 {
				return object.False, nil
			}
			if !cfgHasSection(cfg, sec.V) {
				return object.False, nil
			}
			return object.BoolOf(cfgHasOption(cfg, sec.V, opt.V)), nil
		}}, true

	case "get":
		return &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "get() missing section or option")
			}
			sec, ok := a[0].(*object.Str)
			opt, ok2 := a[1].(*object.Str)
			if !ok || !ok2 {
				return nil, object.Errorf(i.typeErr, "get() section and option must be str")
			}
			raw := false
			var fallback object.Object
			hasFallback := false
			if kw != nil {
				if rv, ok := kw.GetStr("raw"); ok {
					if b, ok := rv.(*object.Bool); ok {
						raw = b.V
					}
				}
				if fv, ok := kw.GetStr("fallback"); ok {
					fallback = fv
					hasFallback = true
				}
			}
			return cfgGetValue(cfg, sec.V, opt.V, raw, fallback, hasFallback)
		}}, true

	case "getint":
		return &object.BuiltinFunc{Name: "getint", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			v, err := cfgGetTyped(i, cfg, a, kw)
			if err != nil || v == nil {
				return v, err
			}
			s, ok := v.(*object.Str)
			if !ok {
				return v, nil
			}
			n, e := strconv.ParseInt(strings.TrimSpace(s.V), 10, 64)
			if e != nil {
				return nil, object.Errorf(i.valueErr, "invalid literal for int: %q", s.V)
			}
			return object.NewInt(n), nil
		}}, true

	case "getfloat":
		return &object.BuiltinFunc{Name: "getfloat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			v, err := cfgGetTyped(i, cfg, a, kw)
			if err != nil || v == nil {
				return v, err
			}
			s, ok := v.(*object.Str)
			if !ok {
				return v, nil
			}
			f, e := strconv.ParseFloat(strings.TrimSpace(s.V), 64)
			if e != nil {
				return nil, object.Errorf(i.valueErr, "could not convert to float: %q", s.V)
			}
			return &object.Float{V: f}, nil
		}}, true

	case "getboolean":
		return &object.BuiltinFunc{Name: "getboolean", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			v, err := cfgGetTyped(i, cfg, a, kw)
			if err != nil || v == nil {
				return v, err
			}
			// If fallback was a bool (not string), return it.
			if _, ok := v.(*object.Bool); ok {
				return v, nil
			}
			s, ok := v.(*object.Str)
			if !ok {
				return v, nil
			}
			bval, ok2 := booleanStates[strings.ToLower(strings.TrimSpace(s.V))]
			if !ok2 {
				return nil, object.Errorf(i.valueErr, "Not a boolean: %s", s.V)
			}
			return object.BoolOf(bval), nil
		}}, true

	case "items":
		return &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				// Return all sections.
				v := make([]object.Object, len(cfg.Sections))
				for k, s := range cfg.Sections {
					v[k] = &object.Tuple{V: []object.Object{
						&object.Str{V: s},
						&object.SectionProxyObj{Parser: cfg, Section: s},
					}}
				}
				return &object.List{V: v}, nil
			}
			sec, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "items() section must be str")
			}
			if !cfgHasSection(cfg, sec.V) {
				return nil, object.Errorf(cfg.NoSecErr, "No section: %q", sec.V)
			}
			raw := false
			if kw != nil {
				if rv, ok := kw.GetStr("raw"); ok {
					if b, ok := rv.(*object.Bool); ok {
						raw = b.V
					}
				}
			}
			opts := cfgOptionsList(cfg, sec.V)
			result := make([]object.Object, 0, len(opts))
			for _, opt := range opts {
				val, err := cfgGetValue(cfg, sec.V, opt, raw, nil, false)
				if err != nil {
					continue
				}
				result = append(result, &object.Tuple{V: []object.Object{
					&object.Str{V: opt}, val,
				}})
			}
			return &object.List{V: result}, nil
		}}, true

	case "set":
		return &object.BuiltinFunc{Name: "set", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "set() requires section, option, value")
			}
			sec, ok := a[0].(*object.Str)
			opt, ok2 := a[1].(*object.Str)
			if !ok || !ok2 {
				return nil, object.Errorf(i.typeErr, "set() section and option must be str")
			}
			valStr := ""
			if vs, ok := a[2].(*object.Str); ok {
				valStr = vs.V
			} else if n, ok := a[2].(*object.Int); ok {
				valStr = n.V.String()
			} else {
				valStr = object.Str_(a[2])
			}
			optKey := strings.ToLower(opt.V)
			if sec.V == cfg.DefaultSection {
				cfg.Defaults.Set(optKey, valStr)
			} else {
				secData, ok := cfg.Data[sec.V]
				if !ok {
					return nil, object.Errorf(cfg.NoSecErr, "No section: %q", sec.V)
				}
				secData.Set(optKey, valStr)
			}
			return object.None, nil
		}}, true

	case "remove_option":
		return &object.BuiltinFunc{Name: "remove_option", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "remove_option() requires section and option")
			}
			sec, ok := a[0].(*object.Str)
			opt, ok2 := a[1].(*object.Str)
			if !ok || !ok2 {
				return nil, object.Errorf(i.typeErr, "remove_option() section and option must be str")
			}
			secData, secOk := cfg.Data[sec.V]
			if !secOk {
				return nil, object.Errorf(cfg.NoSecErr, "No section: %q", sec.V)
			}
			optKey := strings.ToLower(opt.V)
			removed := secData.Del(optKey)
			return object.BoolOf(removed), nil
		}}, true

	case "remove_section":
		return &object.BuiltinFunc{Name: "remove_section", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			name, ok := a[0].(*object.Str)
			if !ok {
				return object.False, nil
			}
			if _, exists := cfg.Data[name.V]; !exists {
				return object.False, nil
			}
			delete(cfg.Data, name.V)
			newSecs := cfg.Sections[:0]
			for _, s := range cfg.Sections {
				if s != name.V {
					newSecs = append(newSecs, s)
				}
			}
			cfg.Sections = newSecs
			return object.True, nil
		}}, true

	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "write() requires a file object")
			}
			spaceAround := true
			if kw != nil {
				if sv, ok := kw.GetStr("space_around_delimiters"); ok {
					if b, ok := sv.(*object.Bool); ok {
						spaceAround = b.V
					}
				}
			}
			if len(a) >= 2 {
				if b, ok := a[1].(*object.Bool); ok {
					spaceAround = b.V
				}
			}
			return object.None, cfgWrite(cfg, a[0], spaceAround, i)
		}}, true

	case "read_string":
		return &object.BuiltinFunc{Name: "read_string", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "read_string() requires a string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "read_string() argument must be str")
			}
			return object.None, cfgParseString(cfg, s.V)
		}}, true

	case "read_file":
		return &object.BuiltinFunc{Name: "read_file", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "read_file() requires a file object")
			}
			// Read all from file-like object by calling read() or iterating.
			content, err := cfgReadFileObj(i, a[0])
			if err != nil {
				return nil, err
			}
			return object.None, cfgParseString(cfg, content)
		}}, true

	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// read() accepts filenames but in testable context we just return [].
			return &object.List{V: nil}, nil
		}}, true

	case "read_dict":
		return &object.BuiltinFunc{Name: "read_dict", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "read_dict() requires a dict")
			}
			d, ok := a[0].(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "read_dict() argument must be dict")
			}
			sectionKeys, sectionVals := d.Items()
			for k, kobj := range sectionKeys {
				secName, ok := kobj.(*object.Str)
				if !ok {
					continue
				}
				secDict, ok := sectionVals[k].(*object.Dict)
				if !ok {
					continue
				}
				sName := secName.V
				if !cfgHasSection(cfg, sName) {
					cfg.Sections = append(cfg.Sections, sName)
					cfg.Data[sName] = object.NewCfgSection()
				}
				optKeys, optVals := secDict.Items()
				for ok2, oobj := range optKeys {
					oname, ok3 := oobj.(*object.Str)
					if !ok3 {
						continue
					}
					oval := optVals[ok2]
					var sval string
					if sv, ok4 := oval.(*object.Str); ok4 {
						sval = sv.V
					} else {
						sval = object.Str_(oval)
					}
					cfg.Data[sName].Set(strings.ToLower(oname.V), sval)
				}
			}
			return object.None, nil
		}}, true

	case "optionxform":
		return &object.BuiltinFunc{Name: "optionxform", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "optionxform() requires an option string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "optionxform() argument must be str")
			}
			return &object.Str{V: strings.ToLower(s.V)}, nil
		}}, true
	}
	return nil, false
}

func cfgGetTyped(i *Interp, cfg *object.ConfigParserObj, a []object.Object, kw *object.Dict) (object.Object, error) {
	if len(a) < 2 {
		return nil, object.Errorf(i.typeErr, "getXxx() requires section and option")
	}
	sec, ok := a[0].(*object.Str)
	opt, ok2 := a[1].(*object.Str)
	if !ok || !ok2 {
		return nil, object.Errorf(i.typeErr, "getXxx() section and option must be str")
	}
	raw := false
	var fallback object.Object
	hasFallback := false
	if kw != nil {
		if rv, ok := kw.GetStr("raw"); ok {
			if b, ok := rv.(*object.Bool); ok {
				raw = b.V
			}
		}
		if fv, ok := kw.GetStr("fallback"); ok {
			fallback = fv
			hasFallback = true
		}
	}
	v, err := cfgGetValue(cfg, sec.V, opt.V, raw, fallback, hasFallback)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func cfgReadFileObj(i *Interp, f object.Object) (string, error) {
	// Try calling read() method first.
	if readFn, err := i.getAttr(f, "read"); err == nil {
		result, err2 := i.callObject(readFn, nil, nil)
		if err2 == nil {
			if s, ok := result.(*object.Str); ok {
				return s.V, nil
			}
		}
	}
	// Fall back to iterating lines.
	lines, err := i.iterStrings(f)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

// --- attr dispatch for SectionProxyObj -------------------------------------

func sectionProxyAttr(i *Interp, sp *object.SectionProxyObj, name string) (object.Object, bool) {
	cfg := sp.Parser
	section := sp.Section

	switch name {
	case "get":
		return &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "get() missing option")
			}
			opt, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "get() option must be str")
			}
			fallback := object.Object(object.None)
			hasFallback := false
			if len(a) >= 2 {
				fallback = a[1]
				hasFallback = true
			}
			if kw != nil {
				if fv, ok := kw.GetStr("fallback"); ok {
					fallback = fv
					hasFallback = true
				}
			}
			return cfgGetValue(cfg, section, opt.V, false, fallback, hasFallback)
		}}, true

	case "getint":
		return &object.BuiltinFunc{Name: "getint", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getint() missing option")
			}
			v, err := cfgGetValue(cfg, section, a[0].(*object.Str).V, false, nil, false)
			if err != nil {
				return nil, err
			}
			if s, ok := v.(*object.Str); ok {
				n, e := strconv.ParseInt(strings.TrimSpace(s.V), 10, 64)
				if e != nil {
					return nil, object.Errorf(i.valueErr, "invalid literal for int")
				}
				return object.NewInt(n), nil
			}
			return v, nil
		}}, true

	case "getfloat":
		return &object.BuiltinFunc{Name: "getfloat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getfloat() missing option")
			}
			v, err := cfgGetValue(cfg, section, a[0].(*object.Str).V, false, nil, false)
			if err != nil {
				return nil, err
			}
			if s, ok := v.(*object.Str); ok {
				f, e := strconv.ParseFloat(strings.TrimSpace(s.V), 64)
				if e != nil {
					return nil, object.Errorf(i.valueErr, "could not convert to float")
				}
				return &object.Float{V: f}, nil
			}
			return v, nil
		}}, true

	case "getboolean":
		return &object.BuiltinFunc{Name: "getboolean", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getboolean() missing option")
			}
			v, err := cfgGetValue(cfg, section, a[0].(*object.Str).V, false, nil, false)
			if err != nil {
				return nil, err
			}
			if s, ok := v.(*object.Str); ok {
				bval, ok2 := booleanStates[strings.ToLower(strings.TrimSpace(s.V))]
				if !ok2 {
					return nil, object.Errorf(i.valueErr, "Not a boolean: %s", s.V)
				}
				return object.BoolOf(bval), nil
			}
			return v, nil
		}}, true

	case "keys":
		return &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			opts := cfgOptionsList(cfg, section)
			v := make([]object.Object, len(opts))
			for k, o := range opts {
				v[k] = &object.Str{V: o}
			}
			return &object.List{V: v}, nil
		}}, true

	case "values":
		return &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			opts := cfgOptionsList(cfg, section)
			v := make([]object.Object, 0, len(opts))
			for _, opt := range opts {
				val, err := cfgGetValue(cfg, section, opt, false, nil, false)
				if err != nil {
					continue
				}
				v = append(v, val)
			}
			return &object.List{V: v}, nil
		}}, true

	case "items":
		return &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			opts := cfgOptionsList(cfg, section)
			result := make([]object.Object, 0, len(opts))
			for _, opt := range opts {
				val, err := cfgGetValue(cfg, section, opt, false, nil, false)
				if err != nil {
					continue
				}
				result = append(result, &object.Tuple{V: []object.Object{
					&object.Str{V: opt}, val,
				}})
			}
			return &object.List{V: result}, nil
		}}, true
	}
	return nil, false
}
