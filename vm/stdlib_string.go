package vm

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// buildStringMod returns the full `string` module: constants, Formatter,
// Template, and capwords.
func (i *Interp) buildStringMod() *object.Module {
	m := &object.Module{Name: "string", Dict: object.NewDict()}

	// --- constants ---
	m.Dict.SetStr("ascii_lowercase", &object.Str{V: "abcdefghijklmnopqrstuvwxyz"})
	m.Dict.SetStr("ascii_uppercase", &object.Str{V: "ABCDEFGHIJKLMNOPQRSTUVWXYZ"})
	m.Dict.SetStr("ascii_letters", &object.Str{V: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"})
	m.Dict.SetStr("digits", &object.Str{V: "0123456789"})
	m.Dict.SetStr("hexdigits", &object.Str{V: "0123456789abcdefABCDEF"})
	m.Dict.SetStr("octdigits", &object.Str{V: "01234567"})
	m.Dict.SetStr("punctuation", &object.Str{V: "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"})
	m.Dict.SetStr("whitespace", &object.Str{V: " \t\n\r\v\f"})
	m.Dict.SetStr("printable", &object.Str{V: "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ \t\n\r\v\f"})

	// --- capwords ---
	m.Dict.SetStr("capwords", &object.BuiltinFunc{Name: "capwords", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "capwords() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "capwords() argument must be str")
		}
		sep := ""
		if len(a) >= 2 {
			if sv, ok2 := a[1].(*object.Str); ok2 {
				sep = sv.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("sep"); ok2 {
				if sv, ok3 := v.(*object.Str); ok3 {
					sep = sv.V
				}
			}
		}
		var words []string
		if sep == "" {
			words = strings.Fields(s.V)
			for k, w := range words {
				words[k] = capitalize(w)
			}
			return &object.Str{V: strings.Join(words, " ")}, nil
		}
		words = strings.Split(s.V, sep)
		for k, w := range words {
			words[k] = capitalize(w)
		}
		return &object.Str{V: strings.Join(words, sep)}, nil
	}})

	// --- Formatter class ---
	m.Dict.SetStr("Formatter", i.buildFormatterClass())

	// --- Template class ---
	m.Dict.SetStr("Template", i.buildTemplateClass())

	return m
}

// capitalize mimics Python's str.capitalize(): first rune to upper, rest to lower.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for k := 1; k < len(runes); k++ {
		runes[k] = unicode.ToLower(runes[k])
	}
	return string(runes)
}

// buildFormatterClass returns a BuiltinFunc that acts as the Formatter class
// constructor (calling it returns a Formatter instance).
func (i *Interp) buildFormatterClass() *object.BuiltinFunc {
	constructor := &object.BuiltinFunc{Name: "Formatter", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.newFormatterInstance(), nil
	}}
	return constructor
}

func (i *Interp) newFormatterInstance() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}

	inst.Dict.SetStr("format", &object.BuiltinFunc{Name: "format", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "format() missing format_string argument")
		}
		fs, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "format() format_string must be str")
		}
		result, err := strFormatFull(i, fs.V, a[1:], kw)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: result}, nil
	}})

	inst.Dict.SetStr("vformat", &object.BuiltinFunc{Name: "vformat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "vformat() takes 3 arguments")
		}
		fs, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "vformat() format_string must be str")
		}
		var args []object.Object
		if tup, ok2 := a[1].(*object.Tuple); ok2 {
			args = tup.V
		} else if lst, ok2 := a[1].(*object.List); ok2 {
			args = lst.V
		}
		var kw *object.Dict
		if d, ok2 := a[2].(*object.Dict); ok2 {
			kw = d
		}
		result, err := strFormatFull(i, fs.V, args, kw)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: result}, nil
	}})

	inst.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "parse() missing format_string argument")
		}
		fs, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "parse() argument must be str")
		}
		return parseFormatString(fs.V), nil
	}})

	inst.Dict.SetStr("get_value", &object.BuiltinFunc{Name: "get_value", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "get_value() takes 3 arguments")
		}
		key := a[0]
		var args []object.Object
		if tup, ok := a[1].(*object.Tuple); ok {
			args = tup.V
		} else if lst, ok := a[1].(*object.List); ok {
			args = lst.V
		}
		kw, _ := a[2].(*object.Dict)
		if n, ok := toInt64(key); ok {
			if int(n) < len(args) {
				return args[n], nil
			}
			return nil, object.Errorf(i.indexErr, "index out of range")
		}
		if s, ok := key.(*object.Str); ok && kw != nil {
			if v, ok2 := kw.GetStr(s.V); ok2 {
				return v, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "key not found")
	}})

	inst.Dict.SetStr("get_field", &object.BuiltinFunc{Name: "get_field", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "get_field() takes 3 arguments")
		}
		fn, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "get_field() field_name must be str")
		}
		var args []object.Object
		if tup, ok2 := a[1].(*object.Tuple); ok2 {
			args = tup.V
		} else if lst, ok2 := a[1].(*object.List); ok2 {
			args = lst.V
		}
		kw, _ := a[2].(*object.Dict)
		obj, usedKey, err := resolveFieldName(i, fn.V, args, kw)
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: []object.Object{obj, &object.Str{V: usedKey}}}, nil
	}})

	inst.Dict.SetStr("format_field", &object.BuiltinFunc{Name: "format_field", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "format_field() takes 2 arguments")
		}
		spec := ""
		if sv, ok := a[1].(*object.Str); ok {
			spec = sv.V
		}
		s, err := applyFormatSpec(i, a[0], spec)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: s}, nil
	}})

	inst.Dict.SetStr("convert_field", &object.BuiltinFunc{Name: "convert_field", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "convert_field() takes 2 arguments")
		}
		conv := ""
		if sv, ok := a[1].(*object.Str); ok {
			conv = sv.V
		}
		return applyConversion(a[0], conv), nil
	}})

	inst.Dict.SetStr("check_unused_args", &object.BuiltinFunc{Name: "check_unused_args", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return inst
}

// buildTemplateClass returns a BuiltinFunc acting as the Template class.
func (i *Interp) buildTemplateClass() *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "Template", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		tmpl := ""
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				tmpl = s.V
			}
		}
		return i.newTemplateInstance(tmpl), nil
	}}
}

func (i *Interp) newTemplateInstance(tmpl string) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("template", &object.Str{V: tmpl})

	inst.Dict.SetStr("substitute", &object.BuiltinFunc{Name: "substitute", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mapping := mergeTemplateMapping(a, kw)
		result, err := templateSubstitute(i, tmpl, mapping, false)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: result}, nil
	}})

	inst.Dict.SetStr("safe_substitute", &object.BuiltinFunc{Name: "safe_substitute", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mapping := mergeTemplateMapping(a, kw)
		result, err := templateSubstitute(i, tmpl, mapping, true)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: result}, nil
	}})

	inst.Dict.SetStr("is_valid", &object.BuiltinFunc{Name: "is_valid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(templateIsValid(tmpl)), nil
	}})

	inst.Dict.SetStr("get_identifiers", &object.BuiltinFunc{Name: "get_identifiers", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		ids := templateGetIdentifiers(tmpl)
		out := make([]object.Object, len(ids))
		for k, id := range ids {
			out[k] = &object.Str{V: id}
		}
		return &object.List{V: out}, nil
	}})

	return inst
}

// mergeTemplateMapping combines positional mapping argument with keyword args.
func mergeTemplateMapping(a []object.Object, kw *object.Dict) map[string]string {
	m := map[string]string{}
	if len(a) > 0 {
		if d, ok := a[0].(*object.Dict); ok {
			ks, vs := d.Items()
			for k, key := range ks {
				if s, ok2 := key.(*object.Str); ok2 {
					m[s.V] = object.Str_(vs[k])
				}
			}
		}
	}
	if kw != nil {
		ks, vs := kw.Items()
		for k, key := range ks {
			if s, ok := key.(*object.Str); ok {
				m[s.V] = object.Str_(vs[k])
			}
		}
	}
	return m
}

// templateSubstitute performs $-substitution on the template string.
func templateSubstitute(i *Interp, tmpl string, mapping map[string]string, safe bool) (string, error) {
	var b strings.Builder
	j := 0
	for j < len(tmpl) {
		if tmpl[j] != '$' {
			b.WriteByte(tmpl[j])
			j++
			continue
		}
		j++ // consume '$'
		if j >= len(tmpl) {
			if safe {
				b.WriteByte('$')
			} else {
				return "", object.Errorf(i.valueErr, "Invalid placeholder in string: line 1, col %d", j)
			}
			continue
		}
		if tmpl[j] == '$' {
			b.WriteByte('$')
			j++
			continue
		}
		if tmpl[j] == '{' {
			j++ // consume '{'
			end := strings.IndexByte(tmpl[j:], '}')
			if end < 0 {
				if safe {
					b.WriteString("${")
					continue
				}
				return "", object.Errorf(i.valueErr, "Invalid placeholder in string")
			}
			name := tmpl[j : j+end]
			j += end + 1
			if !isTemplateIdent(name) {
				if safe {
					b.WriteString("${" + name + "}")
					continue
				}
				return "", object.Errorf(i.valueErr, "Invalid placeholder in string: ${%s}", name)
			}
			if v, ok := mapping[name]; ok {
				b.WriteString(v)
			} else if safe {
				b.WriteString("${" + name + "}")
			} else {
				return "", object.Errorf(i.keyErr, "'%s'", name)
			}
			continue
		}
		// bare $identifier
		end := j
		for end < len(tmpl) && isIdentChar(tmpl[end], end == j) {
			end++
		}
		if end == j {
			if safe {
				b.WriteByte('$')
				continue
			}
			return "", object.Errorf(i.valueErr, "Invalid placeholder in string: line 1, col %d", j)
		}
		name := tmpl[j:end]
		j = end
		if v, ok := mapping[name]; ok {
			b.WriteString(v)
		} else if safe {
			b.WriteString("$" + name)
		} else {
			return "", object.Errorf(i.keyErr, "'%s'", name)
		}
	}
	return b.String(), nil
}

func isTemplateIdent(s string) bool {
	if s == "" {
		return false
	}
	for k, c := range s {
		if k == 0 {
			if !unicode.IsLetter(c) && c != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
				return false
			}
		}
	}
	return true
}

func isIdentChar(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	if !first && c >= '0' && c <= '9' {
		return true
	}
	return false
}

func templateIsValid(tmpl string) bool {
	j := 0
	for j < len(tmpl) {
		if tmpl[j] != '$' {
			j++
			continue
		}
		j++
		if j >= len(tmpl) {
			return false
		}
		if tmpl[j] == '$' {
			j++
			continue
		}
		if tmpl[j] == '{' {
			j++
			end := strings.IndexByte(tmpl[j:], '}')
			if end < 0 {
				return false
			}
			name := tmpl[j : j+end]
			if !isTemplateIdent(name) {
				return false
			}
			j += end + 1
			continue
		}
		end := j
		for end < len(tmpl) && isIdentChar(tmpl[end], end == j) {
			end++
		}
		if end == j {
			return false
		}
		j = end
	}
	return true
}

func templateGetIdentifiers(tmpl string) []string {
	var ids []string
	seen := map[string]bool{}
	j := 0
	for j < len(tmpl) {
		if tmpl[j] != '$' {
			j++
			continue
		}
		j++
		if j >= len(tmpl) {
			break
		}
		if tmpl[j] == '$' {
			j++
			continue
		}
		if tmpl[j] == '{' {
			j++
			end := strings.IndexByte(tmpl[j:], '}')
			if end < 0 {
				break
			}
			name := tmpl[j : j+end]
			j += end + 1
			if isTemplateIdent(name) && !seen[name] {
				seen[name] = true
				ids = append(ids, name)
			}
			continue
		}
		end := j
		for end < len(tmpl) && isIdentChar(tmpl[end], end == j) {
			end++
		}
		if end > j {
			name := tmpl[j:end]
			if !seen[name] {
				seen[name] = true
				ids = append(ids, name)
			}
			j = end
		} else {
			j++
		}
	}
	return ids
}

// parseFormatString returns a List of (literal, field_name, format_spec, conversion) tuples.
func parseFormatString(tmpl string) *object.List {
	var out []object.Object
	j := 0
	for j <= len(tmpl) {
		// collect literal text up to next '{' or end
		litStart := j
		for j < len(tmpl) && tmpl[j] != '{' && tmpl[j] != '}' {
			j++
		}
		lit := tmpl[litStart:j]
		if j >= len(tmpl) {
			if lit != "" || len(out) == 0 {
				out = append(out, &object.Tuple{V: []object.Object{
					&object.Str{V: lit},
					object.None,
					object.None,
					object.None,
				}})
			}
			break
		}
		if tmpl[j] == '}' && j+1 < len(tmpl) && tmpl[j+1] == '}' {
			lit += "}"
			j += 2
			continue
		}
		if tmpl[j] == '{' && j+1 < len(tmpl) && tmpl[j+1] == '{' {
			lit += "{"
			j += 2
			continue
		}
		if tmpl[j] == '{' {
			j++ // consume '{'
			// find matching '}'
			depth := 1
			end := j
			for end < len(tmpl) && depth > 0 {
				if tmpl[end] == '{' {
					depth++
				} else if tmpl[end] == '}' {
					depth--
				}
				if depth > 0 {
					end++
				}
			}
			field := tmpl[j:end]
			j = end + 1 // consume '}'

			fieldName := field
			formatSpec := ""
			var conversion object.Object = object.None

			// split on '!' for conversion (but not inside nested {})
			if bang := strings.IndexByte(field, '!'); bang >= 0 {
				rest := field[bang+1:]
				// conversion is a single char
				if len(rest) >= 1 {
					conv := string(rest[0])
					fieldName = field[:bang]
					conversion = &object.Str{V: conv}
					// format spec follows the conversion char
					if len(rest) > 1 && rest[1] == ':' {
						formatSpec = rest[2:]
						// unused but split correctly
					}
				}
			}
			// split on ':' for format spec (may come before or after !)
			if bang := strings.IndexByte(fieldName, '!'); bang < 0 {
				if colon := strings.IndexByte(fieldName, ':'); colon >= 0 {
					formatSpec = fieldName[colon+1:]
					fieldName = fieldName[:colon]
				}
			}

			var fnObj object.Object
			if fieldName == "" {
				fnObj = &object.Str{V: ""}
			} else {
				fnObj = &object.Str{V: fieldName}
			}

			out = append(out, &object.Tuple{V: []object.Object{
				&object.Str{V: lit},
				fnObj,
				&object.Str{V: formatSpec},
				conversion,
			}})
		} else {
			j++
		}
	}
	return &object.List{V: out}
}

// ─── Full format implementation ───────────────────────────────────────────────

// strFormatFull replaces the minimal strFormat: it parses format specs,
// conversions, and attribute/item navigation.
func strFormatFull(i *Interp, tmpl string, args []object.Object, kwargs *object.Dict) (string, error) {
	var b strings.Builder
	autoIdx := 0
	j := 0
	for j < len(tmpl) {
		if tmpl[j] == '{' {
			if j+1 < len(tmpl) && tmpl[j+1] == '{' {
				b.WriteByte('{')
				j += 2
				continue
			}
			// find the closing '}'
			depth := 1
			end := j + 1
			for end < len(tmpl) && depth > 0 {
				if tmpl[end] == '{' {
					depth++
				} else if tmpl[end] == '}' {
					depth--
				}
				if depth > 0 {
					end++
				}
			}
			field := tmpl[j+1 : end]
			j = end + 1

			// parse: field_name[!conv][:spec]
			fieldName, conv, spec := splitField(field)

			// resolve the value
			var val object.Object
			if fieldName == "" {
				if autoIdx < len(args) {
					val = args[autoIdx]
					autoIdx++
				} else {
					return "", object.Errorf(i.indexErr, "tuple index out of range")
				}
			} else {
				var err error
				val, _, err = resolveFieldName(i, fieldName, args, kwargs)
				if err != nil {
					return "", err
				}
			}

			// apply conversion
			val = applyConversion(val, conv)

			// apply format spec (may itself contain nested fields)
			resolvedSpec, err := strFormatFull(i, spec, args, kwargs)
			if err != nil {
				return "", err
			}
			formatted, err := applyFormatSpec(i, val, resolvedSpec)
			if err != nil {
				return "", err
			}
			b.WriteString(formatted)
		} else if tmpl[j] == '}' && j+1 < len(tmpl) && tmpl[j+1] == '}' {
			b.WriteByte('}')
			j += 2
		} else {
			b.WriteByte(tmpl[j])
			j++
		}
	}
	return b.String(), nil
}

// splitField parses "field_name[!conv][:spec]" into components.
func splitField(field string) (name, conv, spec string) {
	// find '!' and ':' positions (ignoring nested braces)
	bangPos := -1
	colonPos := -1
	depth := 0
	for k := 0; k < len(field); k++ {
		switch field[k] {
		case '{':
			depth++
		case '}':
			depth--
		case '!':
			if depth == 0 && bangPos < 0 && colonPos < 0 {
				bangPos = k
			}
		case ':':
			if depth == 0 && colonPos < 0 {
				colonPos = k
			}
		}
	}
	name = field
	if bangPos >= 0 && (colonPos < 0 || bangPos < colonPos) {
		name = field[:bangPos]
		rest := field[bangPos+1:]
		if len(rest) >= 1 {
			conv = string(rest[0])
			rest = rest[1:]
		}
		if len(rest) > 0 && rest[0] == ':' {
			spec = rest[1:]
		}
	} else if colonPos >= 0 {
		name = field[:colonPos]
		spec = field[colonPos+1:]
	}
	return
}

// resolveFieldName navigates "argname.attr[key].attr2" expressions.
func resolveFieldName(i *Interp, fieldName string, args []object.Object, kwargs *object.Dict) (object.Object, string, error) {
	if fieldName == "" {
		return object.None, "", nil
	}
	// find the first name component (before . or [)
	first := fieldName
	rest := ""
	for k := 0; k < len(fieldName); k++ {
		if fieldName[k] == '.' || fieldName[k] == '[' {
			first = fieldName[:k]
			rest = fieldName[k:]
			break
		}
	}

	var val object.Object
	usedKey := first
	if n, ok := parseInt(first); ok {
		if int(n) < len(args) {
			val = args[n]
		} else {
			return nil, "", object.Errorf(i.indexErr, "tuple index out of range")
		}
	} else if kwargs != nil {
		if v, ok := kwargs.GetStr(first); ok {
			val = v
		} else {
			return nil, "", object.Errorf(i.keyErr, "'%s'", first)
		}
	} else {
		return nil, "", object.Errorf(i.keyErr, "'%s'", first)
	}

	// navigate rest: ".attr" or "[key]"
	j := 0
	for j < len(rest) {
		if rest[j] == '.' {
			j++
			end := j
			for end < len(rest) && rest[end] != '.' && rest[end] != '[' {
				end++
			}
			attr := rest[j:end]
			j = end
			var err error
			val, err = i.getAttr(val, attr)
			if err != nil {
				return nil, "", err
			}
		} else if rest[j] == '[' {
			j++
			end := strings.IndexByte(rest[j:], ']')
			if end < 0 {
				break
			}
			key := rest[j : j+end]
			j += end + 1
			var err error
			if n, ok := parseInt(key); ok {
				val, err = i.getitem(val, object.NewInt(n))
			} else {
				val, err = i.getitem(val, &object.Str{V: key})
			}
			if err != nil {
				return nil, "", err
			}
		} else {
			break
		}
	}
	return val, usedKey, nil
}

// applyConversion applies !s, !r, !a.
func applyConversion(val object.Object, conv string) object.Object {
	switch conv {
	case "r":
		return &object.Str{V: object.Repr(val)}
	case "a":
		return &object.Str{V: asciiReprStr(object.Repr(val))}
	case "s":
		return &object.Str{V: object.Str_(val)}
	}
	return val
}

// asciiReprStr escapes non-ASCII chars in a string (for !a conversion).
func asciiReprStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r > 127 {
			b.WriteString(fmt.Sprintf("\\u%04x", r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ─── Format spec mini-language ────────────────────────────────────────────────

type fmtSpec struct {
	fill     rune
	align    byte // '<', '>', '^', '=', 0=unset
	sign     byte // '+', '-', ' ', 0=default
	altForm  bool
	zeroPad  bool
	width    int
	hasWidth bool
	grouping byte // ',', '_', 0=none
	prec     int
	hasPrec  bool
	typChar  byte // 's','d','b','o','x','X','e','E','f','F','g','G','%','c','n', 0=default
}

func parseSpec(spec string) fmtSpec {
	fs := fmtSpec{fill: ' '}
	j := 0
	// fill+align: if spec[1] is align char, spec[0] is fill
	if len(spec) >= 2 && isAlignChar(spec[1]) {
		fs.fill = rune(spec[0])
		fs.align = spec[1]
		j = 2
	} else if len(spec) >= 1 && isAlignChar(spec[0]) {
		fs.align = spec[0]
		j = 1
	}
	// sign
	if j < len(spec) && (spec[j] == '+' || spec[j] == '-' || spec[j] == ' ') {
		fs.sign = spec[j]
		j++
	}
	// 'z' (coerce -0 float — skip)
	if j < len(spec) && spec[j] == 'z' {
		j++
	}
	// '#'
	if j < len(spec) && spec[j] == '#' {
		fs.altForm = true
		j++
	}
	// '0'
	if j < len(spec) && spec[j] == '0' {
		fs.zeroPad = true
		j++
	}
	// width
	wStart := j
	for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
		j++
	}
	if j > wStart {
		fs.width, _ = strconv.Atoi(spec[wStart:j])
		fs.hasWidth = true
	}
	// grouping
	if j < len(spec) && (spec[j] == ',' || spec[j] == '_') {
		fs.grouping = spec[j]
		j++
	}
	// precision
	if j < len(spec) && spec[j] == '.' {
		j++
		pStart := j
		for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
			j++
		}
		if j > pStart {
			fs.prec, _ = strconv.Atoi(spec[pStart:j])
		}
		fs.hasPrec = true
	}
	// type char
	if j < len(spec) {
		fs.typChar = spec[j]
	}
	return fs
}

func isAlignChar(c byte) bool {
	return c == '<' || c == '>' || c == '^' || c == '='
}

// rebuildSpec converts a fmtSpec struct back to a format spec string so
// we can re-invoke applyFormatSpec after adjusting the type char.
func rebuildSpec(fs fmtSpec) string {
	var b strings.Builder
	if fs.fill != ' ' || fs.align != 0 {
		if fs.align != 0 {
			b.WriteRune(fs.fill)
			b.WriteByte(fs.align)
		}
	} else if fs.align != 0 {
		b.WriteByte(fs.align)
	}
	if fs.sign != 0 {
		b.WriteByte(fs.sign)
	}
	if fs.altForm {
		b.WriteByte('#')
	}
	if fs.zeroPad {
		b.WriteByte('0')
	}
	if fs.hasWidth {
		b.WriteString(strconv.Itoa(fs.width))
	}
	if fs.grouping != 0 {
		b.WriteByte(fs.grouping)
	}
	if fs.hasPrec {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(fs.prec))
	}
	if fs.typChar != 0 {
		b.WriteByte(fs.typChar)
	}
	return b.String()
}

// applyFormatSpec formats val according to the Python format spec mini-language.
func applyFormatSpec(i *Interp, val object.Object, spec string) (string, error) {
	if spec == "" {
		return object.Str_(val), nil
	}
	fs := parseSpec(spec)
	var body string
	var prefix string
	var negative bool
	var err error

	switch fs.typChar {
	case 0:
		// default type: use 'd' for integers, 'g' for floats, 's' for others
		if _, _, ok := valToInt(val); ok {
			if _, isFloat := val.(*object.Float); !isFloat {
				// integer path: apply grouping if requested
				fs2 := fs
				fs2.typChar = 'd'
				return applyFormatSpec(i, val, rebuildSpec(fs2))
			}
		}
		if _, ok := valToFloat(val); ok {
			fs2 := fs
			fs2.typChar = 'g'
			if !fs2.hasPrec {
				fs2.prec = 6
				fs2.hasPrec = true
			}
			return applyFormatSpec(i, val, rebuildSpec(fs2))
		}
		// string path
		body = object.Str_(val)
		if fs.hasPrec {
			runes := []rune(body)
			if fs.prec < len(runes) {
				body = string(runes[:fs.prec])
			}
		}
		return applyWidthAlign(body, "", false, fs), nil

	case 's':
		// string formatting
		body = object.Str_(val)
		if fs.hasPrec {
			runes := []rune(body)
			if fs.prec < len(runes) {
				body = string(runes[:fs.prec])
			}
		}
		return applyWidthAlign(body, "", false, fs), nil

	case 'c':
		switch v := val.(type) {
		case *object.Int:
			n, ok := toInt64(v)
			if !ok {
				return "", object.Errorf(i.valueErr, "%%c format: a real number is required")
			}
			body = string(rune(n))
		case *object.Str:
			body = v.V
		default:
			body = object.Str_(val)
		}
		return applyWidthAlign(body, "", false, fs), nil

	case 'd', 'n':
		n, neg, ok := valToInt(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "d format: integer required")
		}
		negative = neg
		body = n.Text(10)
		if fs.grouping == ',' {
			body = insertGroupSep(body, 3, ',')
		} else if fs.grouping == '_' {
			body = insertGroupSep(body, 3, '_')
		}

	case 'b':
		n, neg, ok := valToInt(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "b format: integer required")
		}
		negative = neg
		body = n.Text(2)
		if fs.grouping == '_' {
			body = insertGroupSep(body, 4, '_')
		}
		if fs.altForm {
			prefix = "0b"
		}

	case 'o':
		n, neg, ok := valToInt(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "o format: integer required")
		}
		negative = neg
		body = n.Text(8)
		if fs.grouping == '_' {
			body = insertGroupSep(body, 4, '_')
		}
		if fs.altForm {
			prefix = "0o"
		}

	case 'x':
		n, neg, ok := valToInt(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "x format: integer required")
		}
		negative = neg
		body = n.Text(16)
		if fs.grouping == '_' {
			body = insertGroupSep(body, 4, '_')
		}
		if fs.altForm {
			prefix = "0x"
		}

	case 'X':
		n, neg, ok := valToInt(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "X format: integer required")
		}
		negative = neg
		body = strings.ToUpper(n.Text(16))
		if fs.grouping == '_' {
			body = insertGroupSep(body, 4, '_')
		}
		if fs.altForm {
			prefix = "0X"
		}

	case 'e', 'E':
		f, ok := valToFloat(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "e format: float required")
		}
		negative = math.Signbit(f)
		if negative {
			f = -f
		}
		prec := 6
		if fs.hasPrec {
			prec = fs.prec
		}
		body = strconv.FormatFloat(f, 'e', prec, 64)
		// Python uses e+XX not e+0XX for 2-digit exponents
		body = normalizeExp(body, fs.typChar == 'E')
		if fs.grouping == '_' {
			body = groupFloat_(body)
		}

	case 'f', 'F':
		f, ok := valToFloat(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "f format: float required")
		}
		negative = math.Signbit(f)
		if negative {
			f = -f
		}
		prec := 6
		if fs.hasPrec {
			prec = fs.prec
		}
		body = strconv.FormatFloat(f, 'f', prec, 64)
		if fs.typChar == 'F' {
			body = strings.ToUpper(body)
		}
		if fs.grouping == ',' {
			body = groupFloatComma(body)
		} else if fs.grouping == '_' {
			body = groupFloat_(body)
		}

	case 'g', 'G':
		f, ok := valToFloat(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "g format: float required")
		}
		negative = math.Signbit(f)
		if negative {
			f = -f
		}
		prec := 6
		if fs.hasPrec {
			prec = fs.prec
		}
		if prec == 0 {
			prec = 1
		}
		body = strconv.FormatFloat(f, 'g', prec, 64)
		if fs.typChar == 'G' {
			body = strings.ToUpper(body)
		}
		if fs.altForm && !strings.Contains(body, ".") {
			body += "."
		}
		if fs.grouping == '_' {
			body = groupFloat_(body)
		}

	case '%':
		f, ok := valToFloat(val)
		if !ok {
			return "", object.Errorf(i.valueErr, "percent format: float required")
		}
		negative = math.Signbit(f)
		if negative {
			f = -f
		}
		prec := 6
		if fs.hasPrec {
			prec = fs.prec
		}
		body = strconv.FormatFloat(f*100, 'f', prec, 64) + "%"

	default:
		body = object.Str_(val)
	}

	if err != nil {
		return "", err
	}
	return applyWidthAlign(body, prefix, negative, fs), nil
}

// applyWidthAlign pads/aligns `body` according to the format spec.
func applyWidthAlign(body, prefix string, negative bool, fs fmtSpec) string {
	// build sign string
	var signStr string
	switch fs.sign {
	case '+':
		if negative {
			signStr = "-"
		} else {
			signStr = "+"
		}
	case ' ':
		if negative {
			signStr = "-"
		} else {
			signStr = " "
		}
	default:
		if negative {
			signStr = "-"
		}
	}

	full := signStr + prefix + body

	if !fs.hasWidth || len([]rune(full)) >= fs.width {
		return full
	}

	pad := fs.width - len([]rune(full))
	fill := string(fs.fill)
	if fill == "" {
		fill = " "
	}

	align := fs.align

	// zero-pad: check before defaulting alignment
	if fs.zeroPad && align == 0 {
		// zero-padding implies '=' alignment with '0' fill
		return signStr + prefix + strings.Repeat("0", pad) + body
	}
	if fs.zeroPad && align == '=' {
		return signStr + prefix + strings.Repeat("0", pad) + body
	}

	if align == 0 {
		// default: numbers right-align, strings left-align
		if fs.typChar != 0 && fs.typChar != 's' {
			align = '>'
		} else if negative || fs.sign != 0 {
			align = '>'
		} else {
			align = '<'
		}
	}

	switch align {
	case '<':
		return full + strings.Repeat(fill, pad)
	case '>':
		return strings.Repeat(fill, pad) + full
	case '^':
		left := pad / 2
		right := pad - left
		return strings.Repeat(fill, left) + full + strings.Repeat(fill, right)
	case '=':
		return signStr + prefix + strings.Repeat(fill, pad) + body
	default:
		return strings.Repeat(fill, pad) + full
	}
}

// valToInt extracts a big.Int from an int-like object.
func valToInt(val object.Object) (n *big.Int, negative bool, ok bool) {
	switch v := val.(type) {
	case *object.Int:
		n = new(big.Int).Set(&v.V)
		negative = n.Sign() < 0
		if negative {
			n.Neg(n)
		}
		return n, negative, true
	case *object.Bool:
		if v.V {
			return big.NewInt(1), false, true
		}
		return big.NewInt(0), false, true
	case *object.Float:
		i64 := int64(v.V)
		n = big.NewInt(i64)
		negative = i64 < 0
		if negative {
			n.Neg(n)
		}
		return n, negative, true
	}
	return nil, false, false
}

// valToFloat extracts a float64 from a numeric object.
func valToFloat(val object.Object) (float64, bool) {
	switch v := val.(type) {
	case *object.Float:
		return v.V, true
	case *object.Int:
		f, _ := new(big.Float).SetInt(&v.V).Float64()
		return f, true
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// insertGroupSep inserts `sep` every `n` digits from the right.
func insertGroupSep(s string, n int, sep rune) string {
	if len(s) <= n {
		return s
	}
	var b strings.Builder
	start := len(s) % n
	if start > 0 {
		b.WriteString(s[:start])
	}
	for i := start; i < len(s); i += n {
		if i > 0 {
			b.WriteRune(sep)
		}
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
	}
	return b.String()
}

// groupFloatComma inserts commas in the integer part of a float string.
func groupFloatComma(s string) string {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return insertGroupSep(s, 3, ',')
	}
	return insertGroupSep(s[:dot], 3, ',') + s[dot:]
}

// groupFloat_ inserts underscores in the integer part of a float string.
func groupFloat_(s string) string {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return insertGroupSep(s, 3, '_')
	}
	return insertGroupSep(s[:dot], 3, '_') + s[dot:]
}

// normalizeExp fixes Go's 3-digit exponent e+006 → e+06 to match Python.
func normalizeExp(s string, upper bool) string {
	eChar := byte('e')
	if upper {
		eChar = 'E'
		s = strings.Replace(s, "e", "E", 1)
	}
	eIdx := strings.IndexByte(s, eChar)
	if eIdx < 0 {
		return s
	}
	expPart := s[eIdx+1:]
	mantissa := s[:eIdx+1]
	sign := ""
	if len(expPart) > 0 && (expPart[0] == '+' || expPart[0] == '-') {
		sign = string(expPart[0])
		expPart = expPart[1:]
	}
	// strip leading zeros to match Python's e+06 style (at least 2 digits)
	for len(expPart) > 2 && expPart[0] == '0' {
		expPart = expPart[1:]
	}
	return mantissa + sign + expPart
}
