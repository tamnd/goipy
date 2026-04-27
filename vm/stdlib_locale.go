package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildLocale() *object.Module {
	m := &object.Module{Name: "locale", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────
	m.Dict.SetStr("LC_ALL", object.NewInt(0))
	m.Dict.SetStr("LC_COLLATE", object.NewInt(1))
	m.Dict.SetStr("LC_CTYPE", object.NewInt(2))
	m.Dict.SetStr("LC_MONETARY", object.NewInt(3))
	m.Dict.SetStr("LC_NUMERIC", object.NewInt(4))
	m.Dict.SetStr("LC_TIME", object.NewInt(5))
	m.Dict.SetStr("LC_MESSAGES", object.NewInt(6))
	m.Dict.SetStr("CHAR_MAX", object.NewInt(127))

	// ── Error exception class ─────────────────────────────────────────────────
	localeErr := &object.Class{
		Name:  "Error",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("Error", localeErr)

	// ── Module-level locale state ─────────────────────────────────────────────
	localeState := [7]string{"C", "C", "C", "C", "C", "C", "C"}

	type localeInfo struct {
		decimalPoint   string
		thousandsSep   string
		grouping       []int
		currencySymbol string
		fracDigits     int
	}
	knownLocales := map[string]localeInfo{
		"C": {
			decimalPoint: ".", thousandsSep: "",
			grouping: nil, currencySymbol: "", fracDigits: 127,
		},
		"POSIX": {
			decimalPoint: ".", thousandsSep: "",
			grouping: nil, currencySymbol: "", fracDigits: 127,
		},
		"en_US.UTF-8": {
			decimalPoint: ".", thousandsSep: ",",
			grouping: []int{3, 0}, currencySymbol: "$", fracDigits: 2,
		},
		"en_US.ISO8859-1": {
			decimalPoint: ".", thousandsSep: ",",
			grouping: []int{3, 0}, currencySymbol: "$", fracDigits: 2,
		},
	}
	isKnown := func(loc string) bool {
		_, ok := knownLocales[loc]
		return ok
	}
	currentNumericInfo := func() localeInfo {
		loc := localeState[4] // LC_NUMERIC
		if info, ok := knownLocales[loc]; ok {
			return info
		}
		return knownLocales["C"]
	}
	currentMonetaryInfo := func() localeInfo {
		loc := localeState[3] // LC_MONETARY
		if info, ok := knownLocales[loc]; ok {
			return info
		}
		return knownLocales["C"]
	}

	// ── applyGrouping: insert thousands separator from right ──────────────────
	applyGroupingFn := func(intStr string, sep string, grp []int) string {
		if sep == "" || len(grp) == 0 {
			return intStr
		}
		runes := []rune(intStr)
		if len(runes) == 0 {
			return intStr
		}
		var out []rune
		pos := 0
		grpIdx := 0
		grpSize := grp[0]
		for j := len(runes) - 1; j >= 0; j-- {
			if pos > 0 && grpSize > 0 && pos%grpSize == 0 {
				out = append([]rune(sep), out...)
				if grpIdx+1 < len(grp) && grp[grpIdx+1] != 0 {
					grpIdx++
					grpSize = grp[grpIdx]
				}
			}
			out = append([]rune{runes[j]}, out...)
			pos++
		}
		return string(out)
	}

	// ── setlocale ─────────────────────────────────────────────────────────────
	m.Dict.SetStr("setlocale", &object.BuiltinFunc{
		Name: "setlocale",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "setlocale() requires at least 1 argument")
			}
			catInt, ok := a[0].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "setlocale() category must be int")
			}
			cat := int(catInt.V.Int64())
			if cat < 0 || cat >= 7 {
				cat = 0
			}

			// Determine locale argument
			var locArg object.Object = object.None
			if len(a) >= 2 {
				locArg = a[1]
			} else if kw != nil {
				if v, ok2 := kw.GetStr("locale"); ok2 {
					locArg = v
				}
			}

			if _, isNone := locArg.(*object.NoneType); isNone {
				// Query mode
				return &object.Str{V: localeState[cat]}, nil
			}

			locStr, ok2 := locArg.(*object.Str)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "setlocale() locale must be str or None")
			}
			loc := locStr.V
			if !isKnown(loc) {
				return nil, object.Errorf(localeErr, "unsupported locale setting")
			}
			if cat == 0 { // LC_ALL: set all
				for j := 0; j < 7; j++ {
					localeState[j] = loc
				}
			} else {
				localeState[cat] = loc
			}
			return &object.Str{V: loc}, nil
		},
	})

	// ── getlocale ─────────────────────────────────────────────────────────────
	m.Dict.SetStr("getlocale", &object.BuiltinFunc{
		Name: "getlocale",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			cat := 2 // LC_CTYPE
			if len(a) >= 1 {
				if ci, ok := a[0].(*object.Int); ok {
					cat = int(ci.V.Int64())
				}
			}
			if cat < 0 || cat >= 7 {
				cat = 2
			}
			loc := localeState[cat]
			if loc == "C" || loc == "POSIX" {
				return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
			}
			dotIdx := strings.Index(loc, ".")
			if dotIdx < 0 {
				return &object.Tuple{V: []object.Object{&object.Str{V: loc}, object.None}}, nil
			}
			lang := loc[:dotIdx]
			enc := loc[dotIdx+1:]
			return &object.Tuple{V: []object.Object{&object.Str{V: lang}, &object.Str{V: enc}}}, nil
		},
	})

	// ── getdefaultlocale (deprecated stub) ────────────────────────────────────
	m.Dict.SetStr("getdefaultlocale", &object.BuiltinFunc{
		Name: "getdefaultlocale",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		},
	})

	// ── getencoding ───────────────────────────────────────────────────────────
	m.Dict.SetStr("getencoding", &object.BuiltinFunc{
		Name: "getencoding",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Str{V: "UTF-8"}, nil
		},
	})

	// ── getpreferredencoding ──────────────────────────────────────────────────
	m.Dict.SetStr("getpreferredencoding", &object.BuiltinFunc{
		Name: "getpreferredencoding",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Str{V: "UTF-8"}, nil
		},
	})

	// ── localeconv ────────────────────────────────────────────────────────────
	m.Dict.SetStr("localeconv", &object.BuiltinFunc{
		Name: "localeconv",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			info := currentNumericInfo()
			d := object.NewDict()
			d.SetStr("decimal_point", &object.Str{V: info.decimalPoint})
			d.SetStr("thousands_sep", &object.Str{V: info.thousandsSep})
			var groupList []object.Object
			for _, g := range info.grouping {
				groupList = append(groupList, object.NewInt(int64(g)))
			}
			d.SetStr("grouping", &object.List{V: groupList})
			d.SetStr("int_curr_symbol", &object.Str{V: ""})
			d.SetStr("currency_symbol", &object.Str{V: info.currencySymbol})
			d.SetStr("mon_decimal_point", &object.Str{V: ""})
			d.SetStr("mon_thousands_sep", &object.Str{V: ""})
			d.SetStr("mon_grouping", &object.List{V: nil})
			d.SetStr("positive_sign", &object.Str{V: ""})
			d.SetStr("negative_sign", &object.Str{V: ""})
			d.SetStr("frac_digits", object.NewInt(int64(info.fracDigits)))
			d.SetStr("int_frac_digits", object.NewInt(int64(info.fracDigits)))
			d.SetStr("p_cs_precedes", object.NewInt(127))
			d.SetStr("p_sep_by_space", object.NewInt(127))
			d.SetStr("n_cs_precedes", object.NewInt(127))
			d.SetStr("n_sep_by_space", object.NewInt(127))
			d.SetStr("p_sign_posn", object.NewInt(127))
			d.SetStr("n_sign_posn", object.NewInt(127))
			return d, nil
		},
	})

	// ── atof ──────────────────────────────────────────────────────────────────
	m.Dict.SetStr("atof", &object.BuiltinFunc{
		Name: "atof",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "atof() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "atof() argument must be str")
			}
			info := currentNumericInfo()
			cleaned := strings.TrimSpace(s.V)
			if info.decimalPoint != "." && info.decimalPoint != "" {
				cleaned = strings.ReplaceAll(cleaned, info.decimalPoint, ".")
			}
			f, err := strconv.ParseFloat(cleaned, 64)
			if err != nil {
				return nil, object.Errorf(localeErr, "could not convert string to float: %s", s.V)
			}
			return &object.Float{V: f}, nil
		},
	})

	// ── atoi ──────────────────────────────────────────────────────────────────
	m.Dict.SetStr("atoi", &object.BuiltinFunc{
		Name: "atoi",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "atoi() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "atoi() argument must be str")
			}
			n, err := strconv.ParseInt(strings.TrimSpace(s.V), 10, 64)
			if err != nil {
				return nil, object.Errorf(localeErr, "could not convert string to int: %s", s.V)
			}
			return object.NewInt(n), nil
		},
	})

	// ── delocalize ────────────────────────────────────────────────────────────
	m.Dict.SetStr("delocalize", &object.BuiltinFunc{
		Name: "delocalize",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "delocalize() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "delocalize() argument must be str")
			}
			info := currentNumericInfo()
			result := s.V
			if info.thousandsSep != "" {
				result = strings.ReplaceAll(result, info.thousandsSep, "")
			}
			if info.decimalPoint != "." && info.decimalPoint != "" {
				result = strings.ReplaceAll(result, info.decimalPoint, ".")
			}
			return &object.Str{V: result}, nil
		},
	})

	// ── localize ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("localize", &object.BuiltinFunc{
		Name: "localize",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "localize() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "localize() argument must be str")
			}
			info := currentNumericInfo()
			result := s.V
			if info.decimalPoint != "." && info.decimalPoint != "" {
				result = strings.ReplaceAll(result, ".", info.decimalPoint)
			}
			return &object.Str{V: result}, nil
		},
	})

	// ── format_string ─────────────────────────────────────────────────────────
	m.Dict.SetStr("format_string", &object.BuiltinFunc{
		Name: "format_string",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "format_string() requires at least 2 arguments")
			}
			fmtStr, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "format_string() first argument must be str")
			}
			val := a[1]

			grouping := false
			if len(a) >= 3 {
				if gb, ok2 := a[2].(*object.Bool); ok2 {
					grouping = gb.V
				}
			}
			if kw != nil {
				if g, ok2 := kw.GetStr("grouping"); ok2 {
					if gb, ok3 := g.(*object.Bool); ok3 {
						grouping = gb.V
					}
				}
			}

			info := currentNumericInfo()
			formatted, err := localeSprintfFormat(fmtStr.V, val)
			if err != nil {
				return nil, err
			}

			if grouping && info.thousandsSep != "" && len(info.grouping) > 0 {
				formatted = localeApplyGroupingToFormatted(formatted, info.thousandsSep, info.grouping, info.decimalPoint, applyGroupingFn)
			}
			if info.decimalPoint != "." && info.decimalPoint != "" {
				formatted = strings.ReplaceAll(formatted, ".", info.decimalPoint)
			}

			return &object.Str{V: formatted}, nil
		},
	})

	// ── currency ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("currency", &object.BuiltinFunc{
		Name: "currency",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "currency() requires at least 1 argument")
			}
			var val float64
			switch v := a[0].(type) {
			case *object.Int:
				val = float64(v.V.Int64())
			case *object.Float:
				val = v.V
			default:
				return nil, object.Errorf(i.typeErr, "currency() argument must be number")
			}

			symbol := true
			groupingC := false
			if len(a) >= 2 {
				if sb, ok := a[1].(*object.Bool); ok {
					symbol = sb.V
				}
			}
			if len(a) >= 3 {
				if gb, ok := a[2].(*object.Bool); ok {
					groupingC = gb.V
				}
			}
			if kw != nil {
				if s, ok := kw.GetStr("symbol"); ok {
					if sb, ok2 := s.(*object.Bool); ok2 {
						symbol = sb.V
					}
				}
				if g, ok := kw.GetStr("grouping"); ok {
					if gb, ok2 := g.(*object.Bool); ok2 {
						groupingC = gb.V
					}
				}
			}

			monInfo := currentMonetaryInfo()
			fracDigits := monInfo.fracDigits
			if fracDigits == 127 || fracDigits < 0 {
				fracDigits = 2
			}

			absVal := math.Abs(val)
			intPart := int64(absVal)
			fracPart := absVal - float64(intPart)

			intStr := strconv.FormatInt(intPart, 10)
			if groupingC && len(monInfo.grouping) > 0 && monInfo.thousandsSep != "" {
				intStr = applyGroupingFn(intStr, monInfo.thousandsSep, monInfo.grouping)
			}

			var result string
			if fracDigits > 0 {
				fracStr := fmt.Sprintf("%.*f", fracDigits, fracPart)
				result = intStr + "." + fracStr[2:] // "0.XX" → "XX"
			} else {
				result = intStr
			}

			if val < 0 {
				result = "-" + result
			}
			if symbol {
				result = monInfo.currencySymbol + result
			}

			return &object.Str{V: result}, nil
		},
	})

	// ── strcoll ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("strcoll", &object.BuiltinFunc{
		Name: "strcoll",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "strcoll() requires 2 arguments")
			}
			s1, ok1 := a[0].(*object.Str)
			s2, ok2 := a[1].(*object.Str)
			if !ok1 || !ok2 {
				return nil, object.Errorf(i.typeErr, "strcoll() arguments must be str")
			}
			if s1.V < s2.V {
				return object.NewInt(-1), nil
			}
			if s1.V > s2.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		},
	})

	// ── strxfrm ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("strxfrm", &object.BuiltinFunc{
		Name: "strxfrm",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "strxfrm() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "strxfrm() argument must be str")
			}
			return &object.Str{V: s.V}, nil
		},
	})

	// ── normalize ─────────────────────────────────────────────────────────────
	normalizeTable := map[string]string{
		"c": "C", "C": "C", "POSIX": "C",
		"en":            "en_EN.ISO8859-1",
		"en_US":         "en_US.ISO8859-1",
		"en_US.UTF-8":   "en_US.UTF-8",
		"en_US.UTF8":    "en_US.UTF-8",
		"en_US.utf8":    "en_US.UTF-8",
		"en_US.utf-8":   "en_US.UTF-8",
		"en_GB":         "en_GB.ISO8859-1",
		"en_GB.UTF-8":   "en_GB.UTF-8",
		"de_DE":         "de_DE.ISO8859-1",
		"de_DE.UTF-8":   "de_DE.UTF-8",
		"fr_FR":         "fr_FR.ISO8859-1",
		"fr_FR.UTF-8":   "fr_FR.UTF-8",
		"ja_JP":         "ja_JP.eucJP",
		"ja_JP.UTF-8":   "ja_JP.UTF-8",
		"zh_CN":         "zh_CN.eucCN",
		"zh_CN.UTF-8":   "zh_CN.UTF-8",
		"pt_BR":         "pt_BR.ISO8859-1",
		"pt_BR.UTF-8":   "pt_BR.UTF-8",
		"es_ES":         "es_ES.ISO8859-1",
		"es_ES.UTF-8":   "es_ES.UTF-8",
	}
	m.Dict.SetStr("normalize", &object.BuiltinFunc{
		Name: "normalize",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "normalize() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "normalize() argument must be str")
			}
			if v, found := normalizeTable[s.V]; found {
				return &object.Str{V: v}, nil
			}
			return &object.Str{V: s.V}, nil
		},
	})

	// ── nl_langinfo stub ──────────────────────────────────────────────────────
	m.Dict.SetStr("nl_langinfo", &object.BuiltinFunc{
		Name: "nl_langinfo",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})

	// ── bindtextdomain stub ───────────────────────────────────────────────────
	m.Dict.SetStr("bindtextdomain", &object.BuiltinFunc{
		Name: "bindtextdomain",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					return &object.Str{V: s.V}, nil
				}
			}
			return &object.Str{V: ""}, nil
		},
	})

	// ── textdomain stub ───────────────────────────────────────────────────────
	currentDomain := "messages"
	m.Dict.SetStr("textdomain", &object.BuiltinFunc{
		Name: "textdomain",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) >= 1 && a[0] != object.None {
				if s, ok := a[0].(*object.Str); ok {
					currentDomain = s.V
				}
			}
			return &object.Str{V: currentDomain}, nil
		},
	})

	return m
}

// localeSprintfFormat applies Python %-formatting to val using fmtStr.
func localeSprintfFormat(fmtStr string, val object.Object) (string, error) {
	specs := localeCollectFmtSpecs(fmtStr)
	if len(specs) == 0 {
		return fmtStr, nil
	}

	var args []object.Object
	if tup, ok := val.(*object.Tuple); ok && len(specs) > 1 {
		args = tup.V
	} else {
		args = []object.Object{val}
	}

	var sb strings.Builder
	argIdx := 0
	s := fmtStr
	for _, spec := range specs {
		idx := strings.Index(s, spec)
		if idx < 0 {
			break
		}
		sb.WriteString(s[:idx])
		s = s[idx+len(spec):]

		if argIdx >= len(args) {
			return "", fmt.Errorf("not enough arguments for format string")
		}
		formatted, err := localeApplyFmtSpec(spec, args[argIdx])
		if err != nil {
			return "", err
		}
		sb.WriteString(formatted)
		argIdx++
	}
	sb.WriteString(s)
	return sb.String(), nil
}

func localeCollectFmtSpecs(s string) []string {
	var specs []string
	i := 0
	for i < len(s) {
		if s[i] != '%' {
			i++
			continue
		}
		j := i + 1
		if j >= len(s) {
			break
		}
		if s[j] == '%' {
			i = j + 1
			continue
		}
		for j < len(s) && strings.ContainsRune("-+ #0", rune(s[j])) {
			j++
		}
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j < len(s) && s[j] == '.' {
			j++
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
		}
		if j < len(s) {
			j++
			specs = append(specs, s[i:j])
			i = j
		} else {
			break
		}
	}
	return specs
}

func localeApplyFmtSpec(spec string, arg object.Object) (string, error) {
	if len(spec) < 2 {
		return spec, nil
	}
	conv := spec[len(spec)-1]
	inner := spec[1 : len(spec)-1]
	switch conv {
	case 'd', 'i', 'u':
		var n int64
		switch v := arg.(type) {
		case *object.Int:
			n = v.V.Int64()
		case *object.Float:
			n = int64(v.V)
		case *object.Bool:
			if v.V {
				n = 1
			}
		}
		return fmt.Sprintf("%"+inner+"d", n), nil
	case 'f', 'F':
		var f float64
		switch v := arg.(type) {
		case *object.Float:
			f = v.V
		case *object.Int:
			f = float64(v.V.Int64())
		}
		return fmt.Sprintf("%"+inner+"f", f), nil
	case 'e', 'E', 'g', 'G':
		var f float64
		switch v := arg.(type) {
		case *object.Float:
			f = v.V
		case *object.Int:
			f = float64(v.V.Int64())
		}
		return fmt.Sprintf("%"+inner+string(conv), f), nil
	case 'o', 'x', 'X':
		var n int64
		if v, ok := arg.(*object.Int); ok {
			n = v.V.Int64()
		}
		return fmt.Sprintf("%"+inner+string(conv), n), nil
	case 's':
		str := localeObjToStr(arg)
		if inner == "" {
			return str, nil
		}
		return fmt.Sprintf("%"+inner+"s", str), nil
	}
	return spec, nil
}

func localeObjToStr(o object.Object) string {
	switch v := o.(type) {
	case *object.Str:
		return v.V
	case *object.Int:
		return v.V.String()
	case *object.Float:
		return strconv.FormatFloat(v.V, 'g', -1, 64)
	case *object.Bool:
		if v.V {
			return "True"
		}
		return "False"
	case *object.NoneType:
		return "None"
	}
	return fmt.Sprintf("%v", o)
}

func localeApplyGroupingToFormatted(s, sep string, grp []int, decPoint string,
	applyGrouping func(string, string, []int) string) string {
	dotIdx := strings.LastIndex(s, decPoint)
	var intPart, fracPart string
	if dotIdx >= 0 {
		intPart = s[:dotIdx]
		fracPart = s[dotIdx:]
	} else {
		intPart = s
		fracPart = ""
	}
	neg := ""
	if len(intPart) > 0 && intPart[0] == '-' {
		neg = "-"
		intPart = intPart[1:]
	}
	return neg + applyGrouping(intPart, sep, grp) + fracPart
}
