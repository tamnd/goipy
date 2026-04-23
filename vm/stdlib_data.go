package vm

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"regexp/syntax"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// --- json ---

// buildJSON exposes dumps/loads. Encoding walks Python objects manually so
// that: (a) dict insertion order is preserved by default, (b) Python-only
// types (Tuple, Set, namedtuple instances) round-trip as JSON arrays,
// (c) indent and separators match CPython formatting exactly.
func (i *Interp) buildJSON() *object.Module {
	m := &object.Module{Name: "json", Dict: object.NewDict()}

	m.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "dumps() missing required argument")
		}
		opts := jsonDumpOpts{itemSep: ", ", kvSep: ": "}
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					if n, ok := toInt64(v); ok {
						opts.indent = strings.Repeat(" ", int(n))
						opts.pretty = true
						opts.itemSep = ","
					}
				}
			}
			if v, ok := kw.GetStr("separators"); ok {
				if t, ok := v.(*object.Tuple); ok && len(t.V) == 2 {
					if s, ok := t.V[0].(*object.Str); ok {
						opts.itemSep = s.V
					}
					if s, ok := t.V[1].(*object.Str); ok {
						opts.kvSep = s.V
					}
				}
			}
			if v, ok := kw.GetStr("sort_keys"); ok {
				opts.sortKeys = object.Truthy(v)
			}
		}
		var b strings.Builder
		if err := jsonEncode(&b, a[0], &opts, 0); err != nil {
			return nil, err
		}
		return &object.Str{V: b.String()}, nil
	}})

	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "loads() missing required argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "loads() argument must be str")
		}
		var raw any
		dec := json.NewDecoder(strings.NewReader(s.V))
		dec.UseNumber()
		if err := dec.Decode(&raw); err != nil {
			return nil, object.Errorf(i.valueErr, "json: %s", err.Error())
		}
		return jsonToPy(raw), nil
	}})

	return m
}

type jsonDumpOpts struct {
	indent   string // per-level indent (empty when indent=0 — still pretty-prints).
	pretty   bool   // true if caller passed indent (even 0) — toggles newlines.
	itemSep  string
	kvSep    string
	sortKeys bool
}

func jsonEncode(b *strings.Builder, v object.Object, opts *jsonDumpOpts, depth int) error {
	switch x := v.(type) {
	case nil, *object.NoneType:
		b.WriteString("null")
	case *object.Bool:
		if x.V {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case *object.Int:
		b.WriteString(x.V.String())
	case *object.Float:
		return jsonEncodeFloat(b, x.V)
	case *object.Str:
		return jsonEncodeString(b, x.V)
	case *object.List:
		return jsonEncodeArray(b, x.V, opts, depth)
	case *object.Tuple:
		return jsonEncodeArray(b, x.V, opts, depth)
	case *object.Dict:
		return jsonEncodeDict(b, x, opts, depth)
	case *object.OrderedDict:
		return jsonEncodeDict(b, x.D, opts, depth)
	default:
		return fmt.Errorf("Object of type %s is not JSON serializable", object.TypeName(v))
	}
	return nil
}

func jsonEncodeFloat(b *strings.Builder, f float64) error {
	if math.IsNaN(f) {
		b.WriteString("NaN")
		return nil
	}
	if math.IsInf(f, 1) {
		b.WriteString("Infinity")
		return nil
	}
	if math.IsInf(f, -1) {
		b.WriteString("-Infinity")
		return nil
	}
	// Match Python's json float formatting: shortest repr that round-trips,
	// with a trailing ".0" when the value is integral.
	if f == math.Trunc(f) && !math.IsInf(f, 0) && math.Abs(f) < 1e16 {
		b.WriteString(strconv.FormatFloat(f, 'f', 1, 64))
		return nil
	}
	b.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
	return nil
}

func jsonEncodeString(b *strings.Builder, s string) error {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else if r < 0x80 {
				b.WriteRune(r)
			} else {
				// CPython with ensure_ascii=True (the default) encodes any
				// non-ASCII rune as \uXXXX; surrogate-pair for non-BMP.
				if r > 0xFFFF {
					r -= 0x10000
					fmt.Fprintf(b, `\u%04x\u%04x`, 0xD800+(r>>10), 0xDC00+(r&0x3FF))
				} else {
					fmt.Fprintf(b, `\u%04x`, r)
				}
			}
		}
	}
	b.WriteByte('"')
	return nil
}

func jsonEncodeArray(b *strings.Builder, items []object.Object, opts *jsonDumpOpts, depth int) error {
	if len(items) == 0 {
		b.WriteString("[]")
		return nil
	}
	b.WriteByte('[')
	for k, item := range items {
		if k > 0 {
			b.WriteString(opts.itemSep)
		}
		writeJSONNewline(b, opts, depth+1)
		if err := jsonEncode(b, item, opts, depth+1); err != nil {
			return err
		}
	}
	writeJSONNewline(b, opts, depth)
	b.WriteByte(']')
	return nil
}

func jsonEncodeDict(b *strings.Builder, d *object.Dict, opts *jsonDumpOpts, depth int) error {
	keys, vals := d.Items()
	if len(keys) == 0 {
		b.WriteString("{}")
		return nil
	}
	order := make([]int, len(keys))
	for k := range order {
		order[k] = k
	}
	if opts.sortKeys {
		sort.SliceStable(order, func(a, c int) bool {
			as, _ := keys[order[a]].(*object.Str)
			cs, _ := keys[order[c]].(*object.Str)
			av := ""
			cv := ""
			if as != nil {
				av = as.V
			}
			if cs != nil {
				cv = cs.V
			}
			return av < cv
		})
	}
	b.WriteByte('{')
	for k, idx := range order {
		if k > 0 {
			b.WriteString(opts.itemSep)
		}
		writeJSONNewline(b, opts, depth+1)
		key, err := jsonKey(keys[idx])
		if err != nil {
			return err
		}
		if err := jsonEncodeString(b, key); err != nil {
			return err
		}
		b.WriteString(opts.kvSep)
		if err := jsonEncode(b, vals[idx], opts, depth+1); err != nil {
			return err
		}
	}
	writeJSONNewline(b, opts, depth)
	b.WriteByte('}')
	return nil
}

func writeJSONNewline(b *strings.Builder, opts *jsonDumpOpts, depth int) {
	if !opts.pretty {
		return
	}
	b.WriteByte('\n')
	for k := 0; k < depth; k++ {
		b.WriteString(opts.indent)
	}
}

func jsonKey(k object.Object) (string, error) {
	switch x := k.(type) {
	case *object.Str:
		return x.V, nil
	case *object.Int:
		return x.V.String(), nil
	case *object.Bool:
		if x.V {
			return "true", nil
		}
		return "false", nil
	case *object.NoneType, nil:
		return "null", nil
	}
	return "", fmt.Errorf("keys must be str, int, float, bool or None, not %s", object.TypeName(k))
}

func jsonToPy(v any) object.Object {
	switch x := v.(type) {
	case nil:
		return object.None
	case bool:
		return object.BoolOf(x)
	case string:
		return &object.Str{V: x}
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return object.NewInt(n)
		}
		if bi, ok := new(big.Int).SetString(string(x), 10); ok {
			return object.IntFromBig(bi)
		}
		f, _ := x.Float64()
		return &object.Float{V: f}
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e16 {
			return object.NewInt(int64(x))
		}
		return &object.Float{V: x}
	case []any:
		out := make([]object.Object, len(x))
		for k, v := range x {
			out[k] = jsonToPy(v)
		}
		return &object.List{V: out}
	case map[string]any:
		d := object.NewDict()
		for k, v := range x {
			d.SetStr(k, jsonToPy(v))
		}
		return d
	}
	return object.None
}

// --- re ---

// buildRe wraps Go's regexp. Note: Go uses RE2 which does not support
// backreferences (`\1` in the pattern). We translate the replacement
// syntax (`\1`, `\g<name>`) so callers can use Python-style replacements,
// but pattern backrefs are not supported and will fail to compile.
func (i *Interp) buildRe() *object.Module {
	m := &object.Module{Name: "re", Dict: object.NewDict()}

	// re.error / re.PatternError exception class.
	reErrClass := &object.Class{
		Name:   "error",
		Bases:  []*object.Class{i.exception},
		Dict:   object.NewDict(),
	}
	i.reErr = reErrClass
	m.Dict.SetStr("error", reErrClass)
	m.Dict.SetStr("PatternError", reErrClass)

	// Flag constants (bit values match CPython).
	m.Dict.SetStr("IGNORECASE", object.NewInt(2))
	m.Dict.SetStr("I", object.NewInt(2))
	m.Dict.SetStr("MULTILINE", object.NewInt(8))
	m.Dict.SetStr("M", object.NewInt(8))
	m.Dict.SetStr("DOTALL", object.NewInt(16))
	m.Dict.SetStr("S", object.NewInt(16))
	m.Dict.SetStr("VERBOSE", object.NewInt(64))
	m.Dict.SetStr("X", object.NewInt(64))
	m.Dict.SetStr("ASCII", object.NewInt(256))
	m.Dict.SetStr("A", object.NewInt(256))
	m.Dict.SetStr("UNICODE", object.NewInt(32))
	m.Dict.SetStr("U", object.NewInt(32))
	m.Dict.SetStr("NOFLAG", object.NewInt(0))

	compileArg := func(a []object.Object) (*object.Pattern, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pattern required")
		}
		if p, ok := a[0].(*object.Pattern); ok {
			return p, nil
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pattern must be str")
		}
		flags := int64(0)
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				flags = n
			}
		}
		return i.compileRe(s.V, flags)
	}

	m.Dict.SetStr("compile", &object.BuiltinFunc{Name: "compile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := compileArg(a)
		if err != nil {
			return nil, err
		}
		return p, nil
	}})

	// module-level helpers bypass an explicit compile. flagsAt is the
	// positional index at which `flags` may appear in the module-level
	// signature; fwdArgs is the number of args after the pattern that the
	// underlying Pattern method expects (flags are stripped since the
	// compiled pattern already carries them). extraKw lists kwarg names
	// the method accepts at positions 1.., so we can splice them into
	// the positional args handed to the Pattern method.
	shortcut := func(name string, fwdArgs, flagsAt int, extraKw []string, fn func(*object.Pattern, []object.Object) (object.Object, error)) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "%s requires pattern", name)
			}
			flags := int64(0)
			if flagsAt >= 0 && flagsAt < len(a) {
				if n, ok := toInt64(a[flagsAt]); ok {
					flags = n
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("flags"); ok {
					if n, ok := toInt64(v); ok {
						flags = n
					}
				}
			}
			var p *object.Pattern
			if pat, ok := a[0].(*object.Pattern); ok {
				p = pat
			} else {
				s, ok := a[0].(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "pattern must be str")
				}
				var err error
				p, err = i.compileRe(s.V, flags)
				if err != nil {
					return nil, err
				}
			}
			// Drop the flags argument so downstream Pattern methods don't
			// interpret it as pos/maxsplit/count.
			rest := a[1:]
			if fwdArgs >= 0 && len(rest) > fwdArgs {
				rest = rest[:fwdArgs]
			}
			// Splice kwargs (maxsplit/count) into the expected positions. An
			// empty slot in extraKw is a placeholder for a positional arg
			// that has no kwarg alias (e.g. the repl arg of sub()).
			if kw != nil {
				for idx, kwName := range extraKw {
					if kwName == "" {
						continue
					}
					if v, ok := kw.GetStr(kwName); ok {
						pos := idx + 1 // position in rest (pos 0 is the string or first positional).
						for len(rest) <= pos {
							rest = append(rest, object.NewInt(0))
						}
						rest[pos] = v
					}
				}
			}
			return fn(p, rest)
		}})
	}
	// Signatures (module-level):
	//   match/search/fullmatch(pattern, string, flags=0) → Pattern method takes (string).
	//   findall/finditer(pattern, string, flags=0)       → Pattern method takes (string).
	//   split(pattern, string, maxsplit=0, flags=0)      → Pattern method takes (string, maxsplit).
	//   sub/subn(pattern, repl, string, count=0, flags=0)→ Pattern method takes (repl, string, count).
	shortcut("match", 1, 2, nil, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternMatch(i, p, a, "match") })
	shortcut("search", 1, 2, nil, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternMatch(i, p, a, "search") })
	shortcut("fullmatch", 1, 2, nil, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternMatch(i, p, a, "fullmatch") })
	shortcut("findall", 1, 2, nil, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternFindall(i, p, a) })
	shortcut("finditer", 1, 2, nil, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternFinditer(i, p, a) })
	shortcut("split", 2, 3, []string{"maxsplit"}, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternSplit(i, p, a) })
	shortcut("sub", 3, 4, []string{"", "count"}, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternSub(i, p, a, false) })
	shortcut("subn", 3, 4, []string{"", "count"}, func(p *object.Pattern, a []object.Object) (object.Object, error) { return patternSub(i, p, a, true) })

	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "escape argument must be str")
		}
		return &object.Str{V: regexp.QuoteMeta(s.V)}, nil
	}})

	m.Dict.SetStr("purge", &object.BuiltinFunc{Name: "purge", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return m
}

// pythonPatternToRE2 translates Python-specific regex syntax to Go RE2 syntax.
func pythonPatternToRE2(pattern string, verbose bool) string {
	var b strings.Builder
	inClass := false // inside [...]
	for k := 0; k < len(pattern); k++ {
		c := pattern[k]
		if c == '\\' && k+1 < len(pattern) {
			next := pattern[k+1]
			switch next {
			case 'Z':
				b.WriteString(`\z`)
				k++
				continue
			}
			b.WriteByte(c)
			b.WriteByte(next)
			k++
			continue
		}
		if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		}
		if verbose && !inClass {
			if c == '#' {
				// skip to end of line
				for k < len(pattern) && pattern[k] != '\n' {
					k++
				}
				continue
			}
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v' {
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (i *Interp) compileRe(pattern string, flags int64) (*object.Pattern, error) {
	var prefix strings.Builder
	if flags != 0 {
		prefix.WriteString("(?")
		if flags&2 != 0 {
			prefix.WriteByte('i')
		}
		if flags&8 != 0 {
			prefix.WriteByte('m')
		}
		if flags&16 != 0 {
			prefix.WriteByte('s')
		}
			prefix.WriteByte(')')
	}
	verbose := flags&64 != 0
	translated := pythonPatternToRE2(pattern, verbose)
	re, err := regexp.Compile(prefix.String() + translated)
	if err != nil {
		errCls := i.reErr
		if errCls == nil {
			errCls = i.valueErr
		}
		return nil, object.Errorf(errCls, "%s", err.Error())
	}
	return &object.Pattern{Pattern: pattern, Regexp: re, Flags: flags}, nil
}

func patternMatch(i *Interp, p *object.Pattern, a []object.Object, kind string) (object.Object, error) {
	if len(a) < 1 {
		return nil, object.Errorf(i.typeErr, "%s() missing string", kind)
	}
	s, ok := a[0].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "argument must be str")
	}
	pos, endpos := 0, len(s.V)
	if len(a) >= 2 {
		if n, ok := toInt64(a[1]); ok {
			pos = clampPos(int(n), len(s.V))
		}
	}
	if len(a) >= 3 {
		if n, ok := toInt64(a[2]); ok {
			endpos = clampPos(int(n), len(s.V))
		}
	}
	sub := s.V[pos:endpos]
	var locs []int
	switch kind {
	case "match":
		locs = p.Regexp.FindStringSubmatchIndex(sub)
		if locs == nil || locs[0] != 0 {
			return object.None, nil
		}
	case "fullmatch":
		locs = p.Regexp.FindStringSubmatchIndex(sub)
		if locs == nil || locs[0] != 0 || locs[1] != len(sub) {
			return object.None, nil
		}
	case "search":
		locs = p.Regexp.FindStringSubmatchIndex(sub)
		if locs == nil {
			return object.None, nil
		}
	}
	// Rebase indices onto the full string.
	offsets := make([]int, len(locs))
	for k, v := range locs {
		if v < 0 {
			offsets[k] = -1
		} else {
			offsets[k] = v + pos
		}
	}
	return &object.Match{Pattern: p, String: s.V, Offsets: offsets, Pos: pos, Endpos: endpos}, nil
}

func clampPos(n, length int) int {
	if n < 0 {
		n += length
	}
	if n < 0 {
		return 0
	}
	if n > length {
		return length
	}
	return n
}

func patternFindall(i *Interp, p *object.Pattern, a []object.Object) (object.Object, error) {
	s, ok := a[0].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "argument must be str")
	}
	n := numCaptureGroups(p.Regexp)
	all := p.Regexp.FindAllStringSubmatchIndex(s.V, -1)
	out := make([]object.Object, 0, len(all))
	for _, locs := range all {
		switch {
		case n == 0:
			out = append(out, &object.Str{V: s.V[locs[0]:locs[1]]})
		case n == 1:
			out = append(out, strFromLocs(s.V, locs, 1))
		default:
			parts := make([]object.Object, n)
			for g := 0; g < n; g++ {
				parts[g] = strFromLocs(s.V, locs, g+1)
			}
			out = append(out, &object.Tuple{V: parts})
		}
	}
	return &object.List{V: out}, nil
}

func patternFinditer(i *Interp, p *object.Pattern, a []object.Object) (object.Object, error) {
	s, ok := a[0].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "argument must be str")
	}
	all := p.Regexp.FindAllStringSubmatchIndex(s.V, -1)
	idx := 0
	return &object.Iter{Next: func() (object.Object, bool, error) {
		if idx >= len(all) {
			return nil, false, nil
		}
		locs := all[idx]
		idx++
		return &object.Match{Pattern: p, String: s.V, Offsets: locs, Pos: 0, Endpos: len(s.V)}, true, nil
	}}, nil
}

func patternSplit(i *Interp, p *object.Pattern, a []object.Object) (object.Object, error) {
	s, ok := a[0].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "argument must be str")
	}
	maxSplit := -1
	if len(a) >= 2 {
		if n, ok := toInt64(a[1]); ok {
			maxSplit = int(n)
		}
	}
	matches := p.Regexp.FindAllStringSubmatchIndex(s.V, maxSplit)
	out := []object.Object{}
	n := numCaptureGroups(p.Regexp)
	prev := 0
	for _, m := range matches {
		out = append(out, &object.Str{V: s.V[prev:m[0]]})
		// If the pattern has groups, Python includes each captured group
		// between pieces; we mirror that.
		for g := 0; g < n; g++ {
			out = append(out, strFromLocs(s.V, m, g+1))
		}
		prev = m[1]
	}
	out = append(out, &object.Str{V: s.V[prev:]})
	return &object.List{V: out}, nil
}

// patternSub handles both re.sub and re.subn; the only difference is the
// return shape.
func patternSub(i *Interp, p *object.Pattern, a []object.Object, returnCount bool) (object.Object, error) {
	if len(a) < 2 {
		return nil, object.Errorf(i.typeErr, "sub() requires repl and string")
	}
	repl := a[0]
	src, ok := a[1].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "sub() subject must be str")
	}
	count := -1
	if len(a) >= 3 {
		if n, ok := toInt64(a[2]); ok && n >= 0 {
			if n == 0 {
				count = -1
			} else {
				count = int(n)
			}
		}
	}
	matches := p.Regexp.FindAllStringSubmatchIndex(src.V, -1)
	if count >= 0 && count < len(matches) {
		matches = matches[:count]
	}
	var b strings.Builder
	prev := 0
	for _, m := range matches {
		b.WriteString(src.V[prev:m[0]])
		var out string
		if replStr, ok := repl.(*object.Str); ok {
			out = expandReReplacement(replStr.V, src.V, m, p)
		} else {
			match := &object.Match{Pattern: p, String: src.V, Offsets: m, Pos: 0, Endpos: len(src.V)}
			r, err := i.callObject(repl, []object.Object{match}, nil)
			if err != nil {
				return nil, err
			}
			s, ok := r.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "sub replacement must return str")
			}
			out = s.V
		}
		b.WriteString(out)
		prev = m[1]
	}
	b.WriteString(src.V[prev:])
	if returnCount {
		return &object.Tuple{V: []object.Object{&object.Str{V: b.String()}, object.NewInt(int64(len(matches)))}}, nil
	}
	return &object.Str{V: b.String()}, nil
}

// expandReReplacement translates Python replacement tokens (\1, \g<name>,
// \\) into literal text using the captures from m. If p is non-nil, named
// group references (\g<name>) resolve through its regexp.
func expandReReplacement(repl, src string, m []int, p *object.Pattern) string {
	var b strings.Builder
	for k := 0; k < len(repl); k++ {
		c := repl[k]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if k+1 >= len(repl) {
			b.WriteByte('\\')
			break
		}
		next := repl[k+1]
		switch next {
		case '\\':
			b.WriteByte('\\')
			k++
		case 'n':
			b.WriteByte('\n')
			k++
		case 't':
			b.WriteByte('\t')
			k++
		case 'r':
			b.WriteByte('\r')
			k++
		case 'g':
			if k+2 < len(repl) && repl[k+2] == '<' {
				end := strings.IndexByte(repl[k+3:], '>')
				if end == -1 {
					b.WriteByte('\\')
					continue
				}
				name := repl[k+3 : k+3+end]
				if n, err := strconv.Atoi(name); err == nil {
					b.WriteString(captureString(src, m, n))
				} else if p != nil {
					for idx, nm := range p.Regexp.SubexpNames() {
						if nm == name {
							b.WriteString(captureString(src, m, idx))
							break
						}
					}
				}
				k += 3 + end
			}
		default:
			if next >= '0' && next <= '9' {
				// \N or \NN
				end := k + 2
				if end < len(repl) && repl[end] >= '0' && repl[end] <= '9' {
					end++
				}
				n, _ := strconv.Atoi(repl[k+1 : end])
				b.WriteString(captureString(src, m, n))
				k = end - 1
			} else {
				b.WriteByte('\\')
				b.WriteByte(next)
				k++
			}
		}
	}
	return b.String()
}

func captureString(src string, m []int, group int) string {
	if group*2+1 >= len(m) {
		return ""
	}
	s, e := m[group*2], m[group*2+1]
	if s < 0 {
		return ""
	}
	return src[s:e]
}

func strFromLocs(src string, m []int, group int) object.Object {
	s, e := m[group*2], m[group*2+1]
	if s < 0 {
		return &object.Str{V: ""}
	}
	return &object.Str{V: src[s:e]}
}

// numCaptureGroups returns the number of capturing groups in a compiled
// regex. Go's regexp.NumSubexp already reports this.
func numCaptureGroups(re *regexp.Regexp) int {
	return re.NumSubexp()
}

// Ensures syntax package is referenced (some helpers use it for group map
// extraction in the future).
var _ = syntax.ClassNL

// patternAttr exposes compiled-regex methods (match/search/... just like the
// module-level shortcuts) plus introspection attributes (pattern, flags,
// groups, groupindex).
func patternAttr(i *Interp, p *object.Pattern, name string) (object.Object, bool) {
	switch name {
	case "pattern":
		return &object.Str{V: p.Pattern}, true
	case "flags":
		return object.NewInt(p.Flags), true
	case "groups":
		return object.NewInt(int64(p.Regexp.NumSubexp())), true
	case "groupindex":
		d := object.NewDict()
		for k, n := range p.Regexp.SubexpNames() {
			if n != "" {
				d.SetStr(n, object.NewInt(int64(k)))
			}
		}
		return d, true
	case "match":
		return &object.BuiltinFunc{Name: "match", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternMatch(i, p, a, "match")
		}}, true
	case "search":
		return &object.BuiltinFunc{Name: "search", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternMatch(i, p, a, "search")
		}}, true
	case "fullmatch":
		return &object.BuiltinFunc{Name: "fullmatch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternMatch(i, p, a, "fullmatch")
		}}, true
	case "findall":
		return &object.BuiltinFunc{Name: "findall", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternFindall(i, p, a)
		}}, true
	case "finditer":
		return &object.BuiltinFunc{Name: "finditer", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternFinditer(i, p, a)
		}}, true
	case "split":
		return &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternSplit(i, p, a)
		}}, true
	case "sub":
		return &object.BuiltinFunc{Name: "sub", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternSub(i, p, a, false)
		}}, true
	case "subn":
		return &object.BuiltinFunc{Name: "subn", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return patternSub(i, p, a, true)
		}}, true
	}
	return nil, false
}

// matchAttr exposes Match instance methods and attributes.
func matchAttr(i *Interp, mt *object.Match, name string) (object.Object, bool) {
	switch name {
	case "string":
		return &object.Str{V: mt.String}, true
	case "re":
		return mt.Pattern, true
	case "pos":
		return object.NewInt(int64(mt.Pos)), true
	case "endpos":
		return object.NewInt(int64(mt.Endpos)), true
	case "lastindex":
		idx := 0
		for g := 1; g*2+1 < len(mt.Offsets); g++ {
			if mt.Offsets[g*2] >= 0 {
				idx = g
			}
		}
		if idx == 0 {
			return object.None, true
		}
		return object.NewInt(int64(idx)), true
	case "lastgroup":
		idx := 0
		for g := 1; g*2+1 < len(mt.Offsets); g++ {
			if mt.Offsets[g*2] >= 0 {
				idx = g
			}
		}
		if idx == 0 {
			return object.None, true
		}
		for g, nm := range mt.Pattern.Regexp.SubexpNames() {
			if g == idx && nm != "" {
				return &object.Str{V: nm}, true
			}
		}
		return object.None, true
	case "group":
		return &object.BuiltinFunc{Name: "group", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return matchGroup(i, mt, a)
		}}, true
	case "expand":
		return &object.BuiltinFunc{Name: "expand", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "expand() takes one argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "expand() argument must be str")
			}
			return &object.Str{V: expandReReplacement(s.V, mt.String, mt.Offsets, mt.Pattern)}, nil
		}}, true
	case "groups":
		return &object.BuiltinFunc{Name: "groups", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var dflt object.Object = object.None
			if len(a) >= 1 {
				dflt = a[0]
			}
			if kw != nil {
				if v, ok := kw.GetStr("default"); ok {
					dflt = v
				}
			}
			n := mt.Pattern.Regexp.NumSubexp()
			out := make([]object.Object, n)
			for g := 0; g < n; g++ {
				if v := matchGroupStr(mt, g+1); v != nil {
					out[g] = v
				} else {
					out[g] = dflt
				}
			}
			return &object.Tuple{V: out}, nil
		}}, true
	case "groupdict":
		return &object.BuiltinFunc{Name: "groupdict", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var dflt object.Object = object.None
			if len(a) >= 1 {
				dflt = a[0]
			}
			d := object.NewDict()
			for g, nm := range mt.Pattern.Regexp.SubexpNames() {
				if nm == "" {
					continue
				}
				if v := matchGroupStr(mt, g); v != nil {
					d.SetStr(nm, v)
				} else {
					d.SetStr(nm, dflt)
				}
			}
			return d, nil
		}}, true
	case "start":
		return &object.BuiltinFunc{Name: "start", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			g, err := groupIndex(i, mt, a)
			if err != nil {
				return nil, err
			}
			if g*2 >= len(mt.Offsets) {
				return object.NewInt(-1), nil
			}
			return object.NewInt(int64(mt.Offsets[g*2])), nil
		}}, true
	case "end":
		return &object.BuiltinFunc{Name: "end", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			g, err := groupIndex(i, mt, a)
			if err != nil {
				return nil, err
			}
			if g*2+1 >= len(mt.Offsets) {
				return object.NewInt(-1), nil
			}
			return object.NewInt(int64(mt.Offsets[g*2+1])), nil
		}}, true
	case "span":
		return &object.BuiltinFunc{Name: "span", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			g, err := groupIndex(i, mt, a)
			if err != nil {
				return nil, err
			}
			if g*2+1 >= len(mt.Offsets) {
				return &object.Tuple{V: []object.Object{object.NewInt(-1), object.NewInt(-1)}}, nil
			}
			return &object.Tuple{V: []object.Object{
				object.NewInt(int64(mt.Offsets[g*2])),
				object.NewInt(int64(mt.Offsets[g*2+1])),
			}}, nil
		}}, true
	}
	return nil, false
}

// matchGroupStr returns the Str for a given group number, or nil if the
// group did not participate in the match.
func matchGroupStr(mt *object.Match, g int) object.Object {
	if g*2+1 >= len(mt.Offsets) {
		return nil
	}
	s, e := mt.Offsets[g*2], mt.Offsets[g*2+1]
	if s < 0 {
		return nil
	}
	return &object.Str{V: mt.String[s:e]}
}

// matchGroup handles the variadic .group() call: no args → group 0, one arg
// → that group (or a single default), multiple args → a tuple of each group.
func matchGroup(i *Interp, mt *object.Match, a []object.Object) (object.Object, error) {
	if len(a) == 0 {
		return matchGroupValue(i, mt, 0)
	}
	if len(a) == 1 {
		return matchGroupValue(i, mt, resolveGroup(mt, a[0]))
	}
	out := make([]object.Object, len(a))
	for k, g := range a {
		v, err := matchGroupValue(i, mt, resolveGroup(mt, g))
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return &object.Tuple{V: out}, nil
}

func matchGroupValue(i *Interp, mt *object.Match, g int) (object.Object, error) {
	if g < 0 || g*2+1 >= len(mt.Offsets) {
		return nil, object.Errorf(i.indexErr, "no such group")
	}
	s, e := mt.Offsets[g*2], mt.Offsets[g*2+1]
	if s < 0 {
		return object.None, nil
	}
	return &object.Str{V: mt.String[s:e]}, nil
}

// resolveGroup maps either an int index or a named-group string to a numeric
// group index. Returns -1 for unknown names.
func resolveGroup(mt *object.Match, o object.Object) int {
	if n, ok := toInt64(o); ok {
		return int(n)
	}
	if s, ok := o.(*object.Str); ok {
		for g, nm := range mt.Pattern.Regexp.SubexpNames() {
			if nm == s.V {
				return g
			}
		}
	}
	return -1
}

func groupIndex(i *Interp, mt *object.Match, a []object.Object) (int, error) {
	if len(a) == 0 {
		return 0, nil
	}
	g := resolveGroup(mt, a[0])
	if g < 0 {
		return 0, object.Errorf(i.indexErr, "no such group")
	}
	return g, nil
}

// --- copy ---

func (i *Interp) buildCopy() *object.Module {
	m := &object.Module{Name: "copy", Dict: object.NewDict()}

	// copy.error / copy.Error — exception class
	copyErr := &object.Class{Name: "Error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("error", copyErr)
	m.Dict.SetStr("Error", copyErr)

	m.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "copy() takes 1 argument")
		}
		return i.shallowCopy(a[0])
	}})

	m.Dict.SetStr("deepcopy", &object.BuiltinFunc{Name: "deepcopy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "deepcopy() takes 1 argument")
		}
		// optional second arg: memo dict (we use it as our seen map)
		seen := map[any]object.Object{}
		return i.deepCopy(a[0], seen)
	}})

	// copy.replace (Python 3.13+): create a modified copy via __replace__
	m.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "replace() requires at least 1 argument")
		}
		obj := a[0]
		// Try __replace__ protocol
		if inst, ok := obj.(*object.Instance); ok {
			if fn, found := classLookup(inst.Class, "__replace__"); found {
				return i.callObject(fn, []object.Object{inst}, kw)
			}
		}
		return nil, object.Errorf(i.typeErr, "replace() argument must support __replace__ protocol")
	}})

	return m
}

// shallowCopy implements copy.copy with __copy__ protocol support.
func (i *Interp) shallowCopy(o object.Object) (object.Object, error) {
	// __copy__ protocol
	if inst, ok := o.(*object.Instance); ok {
		if fn, found := classLookup(inst.Class, "__copy__"); found {
			return i.callObject(fn, []object.Object{inst}, nil)
		}
	}
	return shallowCopyPlain(o), nil
}

// shallowCopyPlain is the pure-Go shallow copy without protocol dispatch.
func shallowCopyPlain(o object.Object) object.Object {
	switch v := o.(type) {
	case *object.List:
		out := make([]object.Object, len(v.V))
		copy(out, v.V)
		return &object.List{V: out}
	case *object.Dict:
		nd := object.NewDict()
		keys, vals := v.Items()
		for k, key := range keys {
			nd.Set(key, vals[k])
		}
		return nd
	case *object.Set:
		ns := object.NewSet()
		for _, it := range v.Items() {
			ns.Add(it)
		}
		return ns
	case *object.Frozenset:
		nf := object.NewFrozenset()
		for _, it := range v.Items() {
			nf.Add(it)
		}
		return nf
	case *object.Tuple:
		out := make([]object.Object, len(v.V))
		copy(out, v.V)
		return &object.Tuple{V: out}
	case *object.Bytearray:
		out := make([]byte, len(v.V))
		copy(out, v.V)
		return &object.Bytearray{V: out}
	case *object.Instance:
		nd := object.NewDict()
		keys, vals := v.Dict.Items()
		for k, key := range keys {
			nd.Set(key, vals[k])
		}
		return &object.Instance{Class: v.Class, Dict: nd}
	}
	return o
}

// deepCopy implements copy.deepcopy with __deepcopy__ protocol and cycle detection.
func (i *Interp) deepCopy(o object.Object, seen map[any]object.Object) (object.Object, error) {
	// __deepcopy__ protocol
	if inst, ok := o.(*object.Instance); ok {
		if c, ok := seen[inst]; ok {
			return c, nil
		}
		if fn, found := classLookup(inst.Class, "__deepcopy__"); found {
			// Pass memo as an empty dict (goipy doesn't need real memo for protocol)
			memo := object.NewDict()
			return i.callObject(fn, []object.Object{inst, memo}, nil)
		}
	}
	return deepCopyPlain(o, seen), nil
}

func deepCopyPlain(o object.Object, seen map[any]object.Object) object.Object {
	switch v := o.(type) {
	case *object.List:
		if c, ok := seen[v]; ok {
			return c
		}
		out := &object.List{V: make([]object.Object, len(v.V))}
		seen[v] = out
		for k, x := range v.V {
			out.V[k] = deepCopyPlain(x, seen)
		}
		return out
	case *object.Dict:
		if c, ok := seen[v]; ok {
			return c
		}
		nd := object.NewDict()
		seen[v] = nd
		keys, vals := v.Items()
		for k, key := range keys {
			nd.Set(deepCopyPlain(key, seen), deepCopyPlain(vals[k], seen))
		}
		return nd
	case *object.Set:
		if c, ok := seen[v]; ok {
			return c
		}
		ns := object.NewSet()
		seen[v] = ns
		for _, it := range v.Items() {
			ns.Add(deepCopyPlain(it, seen))
		}
		return ns
	case *object.Frozenset:
		nf := object.NewFrozenset()
		for _, it := range v.Items() {
			nf.Add(deepCopyPlain(it, seen))
		}
		return nf
	case *object.Tuple:
		out := &object.Tuple{V: make([]object.Object, len(v.V))}
		for k, x := range v.V {
			out.V[k] = deepCopyPlain(x, seen)
		}
		return out
	case *object.Bytearray:
		out := make([]byte, len(v.V))
		copy(out, v.V)
		return &object.Bytearray{V: out}
	case *object.Instance:
		if c, ok := seen[v]; ok {
			return c
		}
		nd := object.NewDict()
		ni := &object.Instance{Class: v.Class, Dict: nd}
		seen[v] = ni
		keys, vals := v.Dict.Items()
		for k, key := range keys {
			nd.Set(key, deepCopyPlain(vals[k], seen))
		}
		return ni
	}
	return o
}

