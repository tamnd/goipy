package vm

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// --- calendar module -------------------------------------------------------

var calMonthName = []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
var calMonthAbbr = []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
var calDayName = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
var calDayAbbr = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func (i *Interp) buildCalendar() *object.Module {
	m := &object.Module{Name: "calendar", Dict: object.NewDict()}

	// Constants for weekday names.
	m.Dict.SetStr("MONDAY", object.IntFromBig(big.NewInt(0)))
	m.Dict.SetStr("TUESDAY", object.IntFromBig(big.NewInt(1)))
	m.Dict.SetStr("WEDNESDAY", object.IntFromBig(big.NewInt(2)))
	m.Dict.SetStr("THURSDAY", object.IntFromBig(big.NewInt(3)))
	m.Dict.SetStr("FRIDAY", object.IntFromBig(big.NewInt(4)))
	m.Dict.SetStr("SATURDAY", object.IntFromBig(big.NewInt(5)))
	m.Dict.SetStr("SUNDAY", object.IntFromBig(big.NewInt(6)))

	m.Dict.SetStr("month_name", strList(calMonthName))
	m.Dict.SetStr("month_abbr", strList(calMonthAbbr))
	m.Dict.SetStr("day_name", strList(calDayName))
	m.Dict.SetStr("day_abbr", strList(calDayAbbr))

	m.Dict.SetStr("isleap", &object.BuiltinFunc{Name: "isleap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		y, err := intArg(i, a, "isleap")
		if err != nil {
			return nil, err
		}
		return object.BoolOf(isLeap(y)), nil
	}})

	m.Dict.SetStr("leapdays", &object.BuiltinFunc{Name: "leapdays", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "leapdays() requires y1 and y2")
		}
		y1v, ok := toInt64(a[0])
		y2v, ok2 := toInt64(a[1])
		if !ok || !ok2 {
			return nil, object.Errorf(i.typeErr, "leapdays() requires int args")
		}
		y1, y2 := int(y1v), int(y2v)
		y1 -= 1
		y2 -= 1
		d := func(y int) int {
			return y/4 - y/100 + y/400
		}
		return object.IntFromBig(big.NewInt(int64(d(y2) - d(y1)))), nil
	}})

	m.Dict.SetStr("weekday", &object.BuiltinFunc{Name: "weekday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "weekday() requires year, month, day")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		d, _ := toInt64(a[2])
		t := time.Date(int(y), time.Month(mo), int(d), 0, 0, 0, 0, time.UTC)
		// Go: Sunday=0..Saturday=6. Python: Monday=0..Sunday=6.
		w := (int(t.Weekday()) + 6) % 7
		return object.IntFromBig(big.NewInt(int64(w))), nil
	}})

	m.Dict.SetStr("monthrange", &object.BuiltinFunc{Name: "monthrange", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthrange() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.Errorf(i.valueErr, "bad month number %d", mo)
		}
		first := time.Date(int(y), time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		w := (int(first.Weekday()) + 6) % 7
		days := daysInMonth(int(y), int(mo))
		return &object.Tuple{V: []object.Object{
			object.IntFromBig(big.NewInt(int64(w))),
			object.IntFromBig(big.NewInt(int64(days))),
		}}, nil
	}})

	m.Dict.SetStr("monthcalendar", &object.BuiltinFunc{Name: "monthcalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthcalendar() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.Errorf(i.valueErr, "bad month number %d", mo)
		}
		first := time.Date(int(y), time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		firstWd := (int(first.Weekday()) + 6) % 7
		days := daysInMonth(int(y), int(mo))
		weeks := [][]int{}
		row := make([]int, 7)
		for i := 0; i < firstWd; i++ {
			row[i] = 0
		}
		day := 1
		col := firstWd
		for day <= days {
			row[col] = day
			day++
			col++
			if col == 7 {
				weeks = append(weeks, row)
				row = make([]int, 7)
				col = 0
			}
		}
		if col != 0 {
			weeks = append(weeks, row)
		}
		out := make([]object.Object, len(weeks))
		for i, w := range weeks {
			lst := make([]object.Object, 7)
			for j, d := range w {
				lst[j] = object.IntFromBig(big.NewInt(int64(d)))
			}
			out[i] = &object.List{V: lst}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("timegm", &object.BuiltinFunc{Name: "timegm", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "timegm() requires a tuple")
		}
		var parts []object.Object
		switch v := a[0].(type) {
		case *object.Tuple:
			parts = v.V
		case *object.List:
			parts = v.V
		default:
			return nil, object.Errorf(i.typeErr, "timegm() requires a tuple")
		}
		if len(parts) < 6 {
			return nil, object.Errorf(i.typeErr, "timegm() needs 6+ fields")
		}
		y, _ := toInt64(parts[0])
		mo, _ := toInt64(parts[1])
		d, _ := toInt64(parts[2])
		h, _ := toInt64(parts[3])
		mi, _ := toInt64(parts[4])
		se, _ := toInt64(parts[5])
		t := time.Date(int(y), time.Month(mo), int(d), int(h), int(mi), int(se), 0, time.UTC)
		return object.IntFromBig(big.NewInt(t.Unix())), nil
	}})

	i.extendCalendar(m)
	return m
}

func isLeap(y int64) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

func daysInMonth(y, m int) int {
	days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if m == 2 && isLeap(int64(y)) {
		return 29
	}
	return days[m-1]
}

func strList(ss []string) *object.List {
	v := make([]object.Object, len(ss))
	for i, s := range ss {
		v[i] = &object.Str{V: s}
	}
	return &object.List{V: v}
}

func intArg(i *Interp, a []object.Object, fn string) (int64, error) {
	if len(a) == 0 {
		return 0, object.Errorf(i.typeErr, "%s() missing argument", fn)
	}
	v, ok := toInt64(a[0])
	if !ok {
		return 0, object.Errorf(i.typeErr, "%s() argument must be int", fn)
	}
	return v, nil
}

// --- pprint module ---------------------------------------------------------

func (i *Interp) buildPprint() *object.Module {
	m := &object.Module{Name: "pprint", Dict: object.NewDict()}

	parseOpts := func(defaultSortDicts bool, kw *object.Dict) pformatOpts {
		opts := pformatOpts{indent: 1, width: 80, depth: -1, sortDicts: defaultSortDicts}
		if kw == nil {
			return opts
		}
		if v, ok := kw.GetStr("indent"); ok {
			if iv, ok := toInt64(v); ok {
				opts.indent = int(iv)
			}
		}
		if v, ok := kw.GetStr("width"); ok {
			if iv, ok := toInt64(v); ok {
				opts.width = int(iv)
			}
		}
		if v, ok := kw.GetStr("depth"); ok {
			if iv, ok := toInt64(v); ok {
				opts.depth = int(iv)
			}
		}
		if v, ok := kw.GetStr("compact"); ok {
			if b, ok := v.(*object.Bool); ok {
				opts.compact = b.V
			}
		}
		if v, ok := kw.GetStr("sort_dicts"); ok {
			if b, ok := v.(*object.Bool); ok {
				opts.sortDicts = b.V
			}
		}
		return opts
	}

	// pformat — sort_dicts=True by default
	m.Dict.SetStr("pformat", &object.BuiltinFunc{Name: "pformat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pformat() missing argument")
		}
		opts := parseOpts(true, kw)
		return &object.Str{V: pformat(a[0], opts)}, nil
	}})

	// pprint — sort_dicts=True by default
	m.Dict.SetStr("pprint", &object.BuiltinFunc{Name: "pprint", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pprint() missing argument")
		}
		opts := parseOpts(true, kw)
		// stream kwarg (ignored: always write to stdout)
		fmt.Fprintln(i.Stdout, pformat(a[0], opts))
		return object.None, nil
	}})

	// pp — sort_dicts=False by default (Python 3.8+)
	m.Dict.SetStr("pp", &object.BuiltinFunc{Name: "pp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pp() missing argument")
		}
		opts := parseOpts(false, kw)
		fmt.Fprintln(i.Stdout, pformat(a[0], opts))
		return object.None, nil
	}})

	m.Dict.SetStr("isreadable", &object.BuiltinFunc{Name: "isreadable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "isreadable() missing argument")
		}
		return object.BoolOf(ppIsReadable(a[0], map[any]bool{})), nil
	}})

	m.Dict.SetStr("isrecursive", &object.BuiltinFunc{Name: "isrecursive", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "isrecursive() missing argument")
		}
		return object.BoolOf(ppIsRecursive(a[0], map[any]bool{})), nil
	}})

	m.Dict.SetStr("saferepr", &object.BuiltinFunc{Name: "saferepr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "saferepr() missing argument")
		}
		return &object.Str{V: ppSafeRepr(a[0], map[any]bool{})}, nil
	}})

	// PrettyPrinter class
	ppClass := &object.Class{Name: "PrettyPrinter", Dict: object.NewDict()}

	ppClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		opts := pformatOpts{indent: 1, width: 80, depth: -1, sortDicts: true}
		// positional args: indent, width, depth, stream
		if len(a) >= 2 {
			if iv, ok := toInt64(a[1]); ok {
				opts.indent = int(iv)
			}
		}
		if len(a) >= 3 {
			if iv, ok := toInt64(a[2]); ok {
				opts.width = int(iv)
			}
		}
		if len(a) >= 4 {
			if iv, ok := toInt64(a[3]); ok {
				opts.depth = int(iv)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if iv, ok2 := toInt64(v); ok2 {
					opts.indent = int(iv)
				}
			}
			if v, ok := kw.GetStr("width"); ok {
				if iv, ok2 := toInt64(v); ok2 {
					opts.width = int(iv)
				}
			}
			if v, ok := kw.GetStr("depth"); ok {
				if iv, ok2 := toInt64(v); ok2 {
					opts.depth = int(iv)
				}
			}
			if v, ok := kw.GetStr("compact"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					opts.compact = b.V
				}
			}
			if v, ok := kw.GetStr("sort_dicts"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					opts.sortDicts = b.V
				}
			}
		}
		// store opts fields in inst dict
		inst.Dict.SetStr("_indent", object.NewInt(int64(opts.indent)))
		inst.Dict.SetStr("_width", object.NewInt(int64(opts.width)))
		inst.Dict.SetStr("_depth", object.NewInt(int64(opts.depth)))
		inst.Dict.SetStr("_compact", object.BoolOf(opts.compact))
		inst.Dict.SetStr("_sort_dicts", object.BoolOf(opts.sortDicts))
		return object.None, nil
	}})

	ppGetOpts := func(inst *object.Instance) pformatOpts {
		opts := pformatOpts{indent: 1, width: 80, depth: -1, sortDicts: true}
		if v, ok := inst.Dict.GetStr("_indent"); ok {
			if iv, ok2 := toInt64(v); ok2 {
				opts.indent = int(iv)
			}
		}
		if v, ok := inst.Dict.GetStr("_width"); ok {
			if iv, ok2 := toInt64(v); ok2 {
				opts.width = int(iv)
			}
		}
		if v, ok := inst.Dict.GetStr("_depth"); ok {
			if iv, ok2 := toInt64(v); ok2 {
				opts.depth = int(iv)
			}
		}
		if v, ok := inst.Dict.GetStr("_compact"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				opts.compact = b.V
			}
		}
		if v, ok := inst.Dict.GetStr("_sort_dicts"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				opts.sortDicts = b.V
			}
		}
		return opts
	}

	ppClass.Dict.SetStr("pformat", &object.BuiltinFunc{Name: "pformat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "pformat() missing argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be PrettyPrinter")
		}
		opts := ppGetOpts(inst)
		return &object.Str{V: pformat(a[1], opts)}, nil
	}})

	ppClass.Dict.SetStr("pprint", &object.BuiltinFunc{Name: "pprint", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "pprint() missing argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be PrettyPrinter")
		}
		opts := ppGetOpts(inst)
		fmt.Fprintln(i.Stdout, pformat(a[1], opts))
		return object.None, nil
	}})

	ppClass.Dict.SetStr("isreadable", &object.BuiltinFunc{Name: "isreadable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		return object.BoolOf(ppIsReadable(a[1], map[any]bool{})), nil
	}})

	ppClass.Dict.SetStr("isrecursive", &object.BuiltinFunc{Name: "isrecursive", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		return object.BoolOf(ppIsRecursive(a[1], map[any]bool{})), nil
	}})

	// format(obj, context, maxlevels, level) → (repr, readable, recursive)
	ppClass.Dict.SetStr("format", &object.BuiltinFunc{Name: "format", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "format() requires object argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be PrettyPrinter")
		}
		obj := a[1]
		opts := ppGetOpts(inst)
		repr := pformat(obj, opts)
		readable := ppIsReadable(obj, map[any]bool{})
		recursive := ppIsRecursive(obj, map[any]bool{})
		return &object.Tuple{V: []object.Object{
			&object.Str{V: repr},
			object.BoolOf(readable),
			object.BoolOf(recursive),
		}}, nil
	}})

	m.Dict.SetStr("PrettyPrinter", ppClass)

	return m
}

type pformatOpts struct {
	indent    int
	width     int
	depth     int
	compact   bool
	sortDicts bool
}

func pformat(o object.Object, opts pformatOpts) string {
	return pformatAt(o, opts, 0, 0)
}

func pformatAt(o object.Object, opts pformatOpts, col, depth int) string {
	if opts.depth >= 0 && depth >= opts.depth {
		switch o.(type) {
		case *object.List:
			return "[...]"
		case *object.Tuple:
			return "(...)"
		case *object.Set, *object.Frozenset:
			return "{...}"
		case *object.Dict:
			return "{...}"
		default:
			return "..."
		}
	}
	switch v := o.(type) {
	case *object.List:
		return formatSeq(v.V, "[", "]", opts, col, depth)
	case *object.Tuple:
		if len(v.V) == 1 {
			s := pformatAt(v.V[0], opts, col+1, depth+1)
			return "(" + s + ",)"
		}
		return formatSeq(v.V, "(", ")", opts, col, depth)
	case *object.Set:
		items := append([]object.Object(nil), v.Items()...)
		if len(items) == 0 {
			return "set()"
		}
		sortItemsByRepr(items)
		return formatSeq(items, "{", "}", opts, col, depth)
	case *object.Frozenset:
		items := append([]object.Object(nil), v.Items()...)
		if len(items) == 0 {
			return "frozenset()"
		}
		sortItemsByRepr(items)
		return "frozenset(" + formatSeq(items, "{", "}", opts, col+len("frozenset("), depth) + ")"
	case *object.Dict:
		return formatDict(v, opts, col, depth)
	}
	return object.Repr(o)
}

func formatSeq(items []object.Object, open, close string, opts pformatOpts, col, depth int) string {
	if len(items) == 0 {
		return open + close
	}
	itemCol := col + opts.indent
	// Build repr of each item at next depth.
	parts := make([]string, len(items))
	for idx, it := range items {
		parts[idx] = pformatAt(it, opts, itemCol, depth+1)
	}
	// Try single-line.
	oneLine := open + strings.Join(parts, ", ") + close
	if col+len(oneLine) <= opts.width {
		return oneLine
	}
	// Multi-line: firstPad aligns first item after open bracket; contPad for continuation lines.
	firstPad := strings.Repeat(" ", max(0, opts.indent-len(open)))
	contPad := strings.Repeat(" ", itemCol)
	if opts.compact {
		// Pack multiple items per line up to width.
		var b strings.Builder
		b.WriteString(open)
		b.WriteString(firstPad)
		lineCol := itemCol
		for idx, p := range parts {
			if idx == 0 {
				b.WriteString(p)
				lineCol += len(p)
			} else {
				needed := 2 + len(p)
				if lineCol+needed <= opts.width {
					b.WriteString(", ")
					b.WriteString(p)
					lineCol += 2 + len(p)
				} else {
					b.WriteString(",\n")
					b.WriteString(contPad)
					b.WriteString(p)
					lineCol = itemCol + len(p)
				}
			}
		}
		b.WriteString(close)
		return b.String()
	}
	// compact=False: one item per line.
	var b strings.Builder
	b.WriteString(open)
	b.WriteString(firstPad)
	for idx, p := range parts {
		if idx > 0 {
			b.WriteString(",\n")
			b.WriteString(contPad)
		}
		b.WriteString(p)
	}
	b.WriteString(close)
	return b.String()
}

func formatDict(d *object.Dict, opts pformatOpts, col, depth int) string {
	keys, vals := d.Items()
	if len(keys) == 0 {
		return "{}"
	}
	type kv struct{ k, v object.Object }
	pairs := make([]kv, len(keys))
	for j := range keys {
		pairs[j] = kv{keys[j], vals[j]}
	}
	if opts.sortDicts {
		sort.SliceStable(pairs, func(a, b int) bool {
			return dictKeyLess(pairs[a].k, pairs[b].k)
		})
	}
	// Build kRepr and vRepr using pformatAt so depth truncation applies.
	type fmtPair struct{ kRepr, vRepr string }
	fpairs := make([]fmtPair, len(pairs))
	for idx, p := range pairs {
		kRepr := object.Repr(p.k)
		vRepr := pformatAt(p.v, opts, col+1+len(kRepr)+2, depth+1)
		fpairs[idx] = fmtPair{kRepr, vRepr}
	}
	// Try single-line.
	parts := make([]string, len(fpairs))
	for idx, fp := range fpairs {
		parts[idx] = fp.kRepr + ": " + fp.vRepr
	}
	oneLine := "{" + strings.Join(parts, ", ") + "}"
	if col+len(oneLine) <= opts.width {
		return oneLine
	}
	pad := strings.Repeat(" ", col+1)
	var b strings.Builder
	b.WriteString("{")
	for idx, fp := range fpairs {
		if idx > 0 {
			b.WriteString(",\n")
			b.WriteString(pad)
		}
		b.WriteString(fp.kRepr)
		b.WriteString(": ")
		b.WriteString(fp.vRepr)
	}
	b.WriteString("}")
	return b.String()
}

func sortItemsByRepr(items []object.Object) {
	sort.SliceStable(items, func(i, j int) bool {
		return object.Repr(items[i]) < object.Repr(items[j])
	})
}

func dictKeyLess(a, b object.Object) bool {
	return object.Repr(a) < object.Repr(b)
}

// ppIsReadable returns true if o can be reconstructed via eval().
func ppIsReadable(o object.Object, seen map[any]bool) bool {
	switch v := o.(type) {
	case *object.List:
		if seen[v] {
			return false
		}
		seen[v] = true
		defer delete(seen, v)
		for _, x := range v.V {
			if !ppIsReadable(x, seen) {
				return false
			}
		}
		return true
	case *object.Tuple:
		for _, x := range v.V {
			if !ppIsReadable(x, seen) {
				return false
			}
		}
		return true
	case *object.Dict:
		if seen[v] {
			return false
		}
		seen[v] = true
		defer delete(seen, v)
		ks, vs := v.Items()
		for j := range ks {
			if !ppIsReadable(ks[j], seen) || !ppIsReadable(vs[j], seen) {
				return false
			}
		}
		return true
	case *object.Set, *object.Frozenset, *object.NoneType, *object.Bool,
		*object.Int, *object.Float, *object.Complex, *object.Str,
		*object.Bytes, *object.Range:
		return true
	}
	return false
}

// ppIsRecursive detects cyclic references via DFS.
func ppIsRecursive(o object.Object, inProgress map[any]bool) bool {
	switch v := o.(type) {
	case *object.List:
		if inProgress[v] {
			return true
		}
		inProgress[v] = true
		defer delete(inProgress, v)
		for _, x := range v.V {
			if ppIsRecursive(x, inProgress) {
				return true
			}
		}
		return false
	case *object.Tuple:
		for _, x := range v.V {
			if ppIsRecursive(x, inProgress) {
				return true
			}
		}
		return false
	case *object.Dict:
		if inProgress[v] {
			return true
		}
		inProgress[v] = true
		defer delete(inProgress, v)
		ks, vs := v.Items()
		for j := range ks {
			if ppIsRecursive(ks[j], inProgress) || ppIsRecursive(vs[j], inProgress) {
				return true
			}
		}
		return false
	case *object.Set:
		for _, x := range v.Items() {
			if ppIsRecursive(x, inProgress) {
				return true
			}
		}
		return false
	case *object.Frozenset:
		for _, x := range v.Items() {
			if ppIsRecursive(x, inProgress) {
				return true
			}
		}
		return false
	}
	return false
}

// ppSafeRepr produces a safe repr that marks recursive objects.
func ppSafeRepr(o object.Object, seen map[any]bool) string {
	switch v := o.(type) {
	case *object.List:
		if seen[v] {
			return fmt.Sprintf("<Recursion on list with id=%d>", ptrID(v))
		}
		seen[v] = true
		defer delete(seen, v)
		parts := make([]string, len(v.V))
		for idx, x := range v.V {
			parts[idx] = ppSafeRepr(x, seen)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *object.Tuple:
		if len(v.V) == 0 {
			return "()"
		}
		parts := make([]string, len(v.V))
		for idx, x := range v.V {
			parts[idx] = ppSafeRepr(x, seen)
		}
		if len(parts) == 1 {
			return "(" + parts[0] + ",)"
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case *object.Dict:
		if seen[v] {
			return fmt.Sprintf("<Recursion on dict with id=%d>", ptrID(v))
		}
		seen[v] = true
		defer delete(seen, v)
		ks, vs := v.Items()
		parts := make([]string, len(ks))
		for j := range ks {
			parts[j] = ppSafeRepr(ks[j], seen) + ": " + ppSafeRepr(vs[j], seen)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return object.Repr(o)
}

func ptrID(v any) uintptr {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.UnsafePointer {
		return rv.Pointer()
	}
	return 0
}

// objectID returns a stable uintptr identity for any object.Object pointer.
func objectID(o object.Object) uintptr {
	return reflect.ValueOf(o).Pointer()
}

// --- reprlib module ---------------------------------------------------------

func (i *Interp) buildReprlib() *object.Module {
	m := &object.Module{Name: "reprlib", Dict: object.NewDict()}

	// Track objects currently being repr'd for recursive_repr decorator.
	reprInProgress := map[uintptr]bool{}

	// ---- helper: get int attribute from instance dict (with default) ----
	getIntAttr := func(inst *object.Instance, name string, def int) int {
		if v, ok := inst.Dict.GetStr(name); ok {
			if n, ok := toInt64(v); ok {
				return int(n)
			}
		}
		return def
	}
	getStrAttr := func(inst *object.Instance, name string, def string) string {
		if v, ok := inst.Dict.GetStr(name); ok {
			if s, ok := v.(*object.Str); ok {
				return s.V
			}
		}
		return def
	}

	// ---- reprLibStr: string with Python-style truncation ----
	reprLibStr := func(s string, maxstring int) string {
		if len(s) <= maxstring {
			return object.Repr(&object.Str{V: s})
		}
		ii := (maxstring - 3) / 2
		j := maxstring - 3 - ii
		if ii < 0 {
			ii = 0
		}
		if j < 0 {
			j = 0
		}
		var tail string
		if j > 0 {
			tail = s[len(s)-j:]
		}
		combined := object.Repr(&object.Str{V: s[:ii] + tail})
		if len(combined) < ii+j {
			return combined
		}
		return combined[:ii] + "..." + combined[len(combined)-j:]
	}

	// ---- reprLibInt: long int truncation ----
	reprLibInt := func(s string, maxlong int) string {
		if len(s) <= maxlong {
			return s
		}
		half := (maxlong - 3) / 2
		rest := maxlong - 3 - half
		return s[:half] + "..." + s[len(s)-rest:]
	}

	// ---- reprLibObj: recursive repr dispatcher ----
	var reprLibObj func(inst *object.Instance, o object.Object, level int) string
	reprLibObj = func(inst *object.Instance, o object.Object, level int) string {
		fv := getStrAttr(inst, "fillvalue", "...")

		seqRepr := func(items []object.Object, open, close string, maxN int) string {
			if len(items) == 0 {
				return open + close
			}
			if level <= 0 {
				return open + fv + close
			}
			var parts []string
			for idx, it := range items {
				if idx >= maxN {
					parts = append(parts, fv)
					break
				}
				parts = append(parts, reprLibObj(inst, it, level-1))
			}
			return open + strings.Join(parts, ", ") + close
		}

		switch v := o.(type) {
		case *object.List:
			maxlist := getIntAttr(inst, "maxlist", 6)
			return seqRepr(v.V, "[", "]", maxlist)
		case *object.Tuple:
			maxtuple := getIntAttr(inst, "maxtuple", 6)
			if len(v.V) == 0 {
				return "()"
			}
			if level <= 0 {
				return "(" + fv + ")"
			}
			if len(v.V) == 1 {
				return "(" + reprLibObj(inst, v.V[0], level-1) + ",)"
			}
			var parts []string
			for idx, it := range v.V {
				if idx >= maxtuple {
					parts = append(parts, fv)
					break
				}
				parts = append(parts, reprLibObj(inst, it, level-1))
			}
			return "(" + strings.Join(parts, ", ") + ")"
		case *object.Dict:
			maxdict := getIntAttr(inst, "maxdict", 4)
			keys, vals := v.Items()
			if len(keys) == 0 {
				return "{}"
			}
			if level <= 0 {
				return "{" + fv + "}"
			}
			var parts []string
			for idx := range keys {
				if idx >= maxdict {
					parts = append(parts, fv)
					break
				}
				kRepr := reprLibObj(inst, keys[idx], level-1)
				vRepr := reprLibObj(inst, vals[idx], level-1)
				parts = append(parts, kRepr+": "+vRepr)
			}
			return "{" + strings.Join(parts, ", ") + "}"
		case *object.Set:
			maxset := getIntAttr(inst, "maxset", 6)
			items := v.Items()
			return seqRepr(items, "{", "}", maxset)
		case *object.Frozenset:
			maxfrozenset := getIntAttr(inst, "maxfrozenset", 6)
			items := v.Items()
			if len(items) == 0 {
				return "frozenset()"
			}
			inner := seqRepr(items, "{", "}", maxfrozenset)
			return "frozenset(" + inner + ")"
		case *object.Str:
			maxstring := getIntAttr(inst, "maxstring", 30)
			return reprLibStr(v.V, maxstring)
		case *object.Int:
			maxlong := getIntAttr(inst, "maxlong", 40)
			return reprLibInt(v.V.String(), maxlong)
		case *object.Deque:
			maxdeque := getIntAttr(inst, "maxdeque", 6)
			return seqRepr(v.V, "[", "]", maxdeque)
		case *object.PyArray:
			maxarray := getIntAttr(inst, "maxarray", 5)
			return seqRepr(v.V, "[", "]", maxarray)
		default:
			maxother := getIntAttr(inst, "maxother", 30)
			s := object.Repr(o)
			if len(s) > maxother {
				half := (maxother - 3) / 2
				rest := maxother - 3 - half
				return s[:half] + "..." + s[len(s)-rest:]
			}
			return s
		}
	}

	// ---- Repr class ----
	reprClass := &object.Class{Name: "Repr", Dict: object.NewDict()}

	reprClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "Repr.__init__ requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "Repr.__init__: self must be Instance")
		}
		// Set defaults.
		inst.Dict.SetStr("maxlevel", object.NewInt(6))
		inst.Dict.SetStr("maxdict", object.NewInt(4))
		inst.Dict.SetStr("maxlist", object.NewInt(6))
		inst.Dict.SetStr("maxtuple", object.NewInt(6))
		inst.Dict.SetStr("maxset", object.NewInt(6))
		inst.Dict.SetStr("maxfrozenset", object.NewInt(6))
		inst.Dict.SetStr("maxdeque", object.NewInt(6))
		inst.Dict.SetStr("maxarray", object.NewInt(5))
		inst.Dict.SetStr("maxstring", object.NewInt(30))
		inst.Dict.SetStr("maxlong", object.NewInt(40))
		inst.Dict.SetStr("maxother", object.NewInt(30))
		inst.Dict.SetStr("fillvalue", &object.Str{V: "..."})
		inst.Dict.SetStr("indent", object.None)
		// Apply keyword arguments.
		if kw != nil {
			for _, name := range []string{"maxlevel", "maxdict", "maxlist", "maxtuple", "maxset",
				"maxfrozenset", "maxdeque", "maxarray", "maxstring", "maxlong", "maxother"} {
				if v, ok := kw.GetStr(name); ok {
					inst.Dict.SetStr(name, v)
				}
			}
			if v, ok := kw.GetStr("fillvalue"); ok {
				inst.Dict.SetStr("fillvalue", v)
			}
			if v, ok := kw.GetStr("indent"); ok {
				inst.Dict.SetStr("indent", v)
			}
		}
		return object.None, nil
	}})

	reprClass.Dict.SetStr("repr1", &object.BuiltinFunc{Name: "repr1", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "repr1() requires self, obj, level")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "repr1: self must be Instance")
		}
		obj := a[1]
		level, _ := toInt64(a[2])
		return &object.Str{V: reprLibObj(inst, obj, int(level))}, nil
	}})

	reprClass.Dict.SetStr("repr", &object.BuiltinFunc{Name: "repr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "repr() requires self, obj")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "repr: self must be Instance")
		}
		maxlevel := getIntAttr(inst, "maxlevel", 6)
		return &object.Str{V: reprLibObj(inst, a[1], maxlevel)}, nil
	}})

	m.Dict.SetStr("Repr", reprClass)

	// ---- Module-level aRepr instance ----
	aRepr := &object.Instance{Class: reprClass, Dict: object.NewDict()}
	// Initialize default attributes on aRepr.
	aRepr.Dict.SetStr("maxlevel", object.NewInt(6))
	aRepr.Dict.SetStr("maxdict", object.NewInt(4))
	aRepr.Dict.SetStr("maxlist", object.NewInt(6))
	aRepr.Dict.SetStr("maxtuple", object.NewInt(6))
	aRepr.Dict.SetStr("maxset", object.NewInt(6))
	aRepr.Dict.SetStr("maxfrozenset", object.NewInt(6))
	aRepr.Dict.SetStr("maxdeque", object.NewInt(6))
	aRepr.Dict.SetStr("maxarray", object.NewInt(5))
	aRepr.Dict.SetStr("maxstring", object.NewInt(30))
	aRepr.Dict.SetStr("maxlong", object.NewInt(40))
	aRepr.Dict.SetStr("maxother", object.NewInt(30))
	aRepr.Dict.SetStr("fillvalue", &object.Str{V: "..."})
	aRepr.Dict.SetStr("indent", object.None)
	m.Dict.SetStr("aRepr", aRepr)

	// ---- Module-level repr() function ----
	m.Dict.SetStr("repr", &object.BuiltinFunc{Name: "repr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "repr() missing argument")
		}
		maxlevel := getIntAttr(aRepr, "maxlevel", 6)
		return &object.Str{V: reprLibObj(aRepr, a[0], maxlevel)}, nil
	}})

	// ---- recursive_repr decorator ----
	m.Dict.SetStr("recursive_repr", &object.BuiltinFunc{Name: "recursive_repr", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// fillvalue argument (positional or keyword).
		fillvalue := "..."
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				fillvalue = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("fillvalue"); ok {
				if s, ok := v.(*object.Str); ok {
					fillvalue = s.V
				}
			}
		}
		fv := fillvalue

		// Return the decorator.
		return &object.BuiltinFunc{Name: "recursive_repr_decorator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "recursive_repr decorator requires a function")
			}
			fn := a[0]
			// Return the wrapper function.
			wrapName := "wrapper"
			if pyFn, ok := fn.(*object.Function); ok {
				wrapName = pyFn.Name
			}
			return &object.BuiltinFunc{Name: wrapName, Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return nil, object.Errorf(i.typeErr, "recursive_repr wrapper requires self")
				}
				selfID := objectID(a[0])
				if reprInProgress[selfID] {
					return &object.Str{V: fv}, nil
				}
				reprInProgress[selfID] = true
				defer delete(reprInProgress, selfID)
				return i.callObject(fn, a, kw)
			}}, nil
		}}, nil
	}})

	return m
}

// --- html module -----------------------------------------------------------

func (i *Interp) buildHtml() *object.Module {
	m := &object.Module{Name: "html", Dict: object.NewDict()}

	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "escape")
		if err != nil {
			return nil, err
		}
		quote := true
		if len(a) >= 2 {
			if b, ok := a[1].(*object.Bool); ok {
				quote = b.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("quote"); ok {
				if b, ok := v.(*object.Bool); ok {
					quote = b.V
				}
			}
		}
		return &object.Str{V: htmlEscape(s, quote)}, nil
	}})

	m.Dict.SetStr("unescape", &object.BuiltinFunc{Name: "unescape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "unescape")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: htmlUnescape(s)}, nil
	}})

	return m
}

func htmlEscape(s string, quote bool) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	if quote {
		s = strings.ReplaceAll(s, `"`, "&quot;")
		s = strings.ReplaceAll(s, "'", "&#x27;")
	}
	return s
}

// htmlEntities maps the common named entities we support. This is a subset of
// HTML5's entity list — enough for real-world text without pulling in a full
// 2000+ entry table.
var htmlEntities = map[string]string{
	"amp": "&", "lt": "<", "gt": ">", "quot": "\"", "apos": "'",
	"nbsp": "\u00a0", "copy": "\u00a9", "reg": "\u00ae", "trade": "\u2122",
	"hellip": "\u2026", "mdash": "\u2014", "ndash": "\u2013",
	"lsquo": "\u2018", "rsquo": "\u2019", "ldquo": "\u201c", "rdquo": "\u201d",
	"laquo": "\u00ab", "raquo": "\u00bb",
	"deg": "\u00b0", "plusmn": "\u00b1", "times": "\u00d7", "divide": "\u00f7",
	"euro": "\u20ac", "pound": "\u00a3", "yen": "\u00a5", "cent": "\u00a2",
	"sect": "\u00a7", "para": "\u00b6", "middot": "\u00b7",
	"iexcl": "\u00a1", "iquest": "\u00bf",
}

func htmlUnescape(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// Find trailing semicolon within a reasonable window.
		end := -1
		for j := i + 1; j < len(s) && j-i < 40; j++ {
			if s[j] == ';' {
				end = j
				break
			}
			if s[j] == '&' || s[j] == ' ' || s[j] == '\t' || s[j] == '\n' {
				break
			}
		}
		if end < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		body := s[i+1 : end]
		if len(body) >= 2 && body[0] == '#' {
			var n int64
			var err error
			if body[1] == 'x' || body[1] == 'X' {
				_, err = fmt.Sscanf(body[2:], "%x", &n)
			} else {
				_, err = fmt.Sscanf(body[1:], "%d", &n)
			}
			if err == nil && n >= 0 && n <= 0x10FFFF {
				b.WriteRune(rune(n))
				i = end + 1
				continue
			}
		} else if v, ok := htmlEntities[body]; ok {
			b.WriteString(v)
			i = end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
