package vm

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// instanceArgs strips a leading *object.Instance self argument from a bound
// method call, returning just the user-visible arguments.
func instanceArgs(a []object.Object) []object.Object {
	if len(a) > 0 {
		if _, ok := a[0].(*object.Instance); ok {
			return a[1:]
		}
	}
	return a
}

// dialectKwargOrPos returns the dialect argument from positional args (index
// pos) or from kw["dialect"], preferring the positional form.
func dialectKwargOrPos(a []object.Object, kw *object.Dict, pos int) object.Object {
	if len(a) > pos {
		return a[pos]
	}
	if kw != nil {
		if v, ok := kw.GetStr("dialect"); ok {
			return v
		}
	}
	return nil
}

// --- dialect registry -------------------------------------------------------

var (
	csvDialectMu       sync.RWMutex
	csvDialectRegistry map[string]*object.CSVDialect
	csvFieldSizeLimit  = 131072
)

func init() {
	csvDialectRegistry = map[string]*object.CSVDialect{
		"excel":     excelDialect(),
		"excel-tab": excelTabDialect(),
		"unix":      unixDialect(),
	}
}

func excelDialect() *object.CSVDialect {
	return &object.CSVDialect{
		Delimiter:      ',',
		Quotechar:      '"',
		Doublequote:    true,
		Lineterminator: "\r\n",
		Quoting:        0,
	}
}

func excelTabDialect() *object.CSVDialect {
	d := excelDialect()
	d.Delimiter = '\t'
	return d
}

func unixDialect() *object.CSVDialect {
	return &object.CSVDialect{
		Delimiter:      ',',
		Quotechar:      '"',
		Doublequote:    true,
		Lineterminator: "\n",
		Quoting:        1, // QUOTE_ALL
	}
}

// resolveDialect builds a CSVDialect from an optional dialect name/object and
// keyword overrides. dialectArg may be nil, a string name, or a CSVDialectObj.
func resolveDialect(base *object.CSVDialect, dialectArg object.Object, kw *object.Dict) *object.CSVDialect {
	d := *base
	if dialectArg != nil {
		switch v := dialectArg.(type) {
		case *object.Str:
			csvDialectMu.RLock()
			if reg, ok := csvDialectRegistry[v.V]; ok {
				d = *reg
			}
			csvDialectMu.RUnlock()
		case *object.CSVDialectObj:
			d = *v.D
		}
	}
	if kw == nil {
		return &d
	}
	if v, ok := kw.GetStr("delimiter"); ok {
		if s, ok := v.(*object.Str); ok && len(s.V) > 0 {
			d.Delimiter = s.V[0]
		}
	}
	if v, ok := kw.GetStr("quotechar"); ok {
		switch s := v.(type) {
		case *object.Str:
			if len(s.V) > 0 {
				d.Quotechar = s.V[0]
			}
		case *object.NoneType:
			d.Quotechar = 0
		}
	}
	if v, ok := kw.GetStr("lineterminator"); ok {
		if s, ok := v.(*object.Str); ok {
			d.Lineterminator = s.V
		}
	}
	if v, ok := kw.GetStr("quoting"); ok {
		if n, ok := toInt64(v); ok {
			d.Quoting = int(n)
		}
	}
	if v, ok := kw.GetStr("doublequote"); ok {
		if b, ok := v.(*object.Bool); ok {
			d.Doublequote = b.V
		}
	}
	if v, ok := kw.GetStr("skipinitialspace"); ok {
		if b, ok := v.(*object.Bool); ok {
			d.SkipInitial = b.V
		}
	}
	if v, ok := kw.GetStr("escapechar"); ok {
		switch s := v.(type) {
		case *object.Str:
			if len(s.V) > 0 {
				d.Escapechar = s.V[0]
			}
		case *object.NoneType:
			d.Escapechar = 0
		}
	}
	if v, ok := kw.GetStr("strict"); ok {
		if b, ok := v.(*object.Bool); ok {
			d.Strict = b.V
		}
	}
	return &d
}

// --- csv module builder -----------------------------------------------------

func (i *Interp) buildCsv() *object.Module {
	m := &object.Module{Name: "csv", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("QUOTE_MINIMAL", object.NewInt(0))
	m.Dict.SetStr("QUOTE_ALL", object.NewInt(1))
	m.Dict.SetStr("QUOTE_NONNUMERIC", object.NewInt(2))
	m.Dict.SetStr("QUOTE_NONE", object.NewInt(3))
	m.Dict.SetStr("QUOTE_STRINGS", object.NewInt(4))
	m.Dict.SetStr("QUOTE_NOTNULL", object.NewInt(5))

	// csv.Error exception class.
	csvErr := &object.Class{Name: "Error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("Error", csvErr)

	// Dialect base class (stub for isinstance checks).
	csvDialectClass := &object.Class{Name: "Dialect", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("Dialect", csvDialectClass)

	// Built-in dialect objects.
	m.Dict.SetStr("excel", &object.CSVDialectObj{D: excelDialect()})
	m.Dict.SetStr("excel_tab", &object.CSVDialectObj{D: excelTabDialect()})
	m.Dict.SetStr("unix_dialect", &object.CSVDialectObj{D: unixDialect()})

	// reader()
	m.Dict.SetStr("reader", &object.BuiltinFunc{Name: "reader", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "reader() missing source")
		}
		lines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		dialectArg := dialectKwargOrPos(a, kw, 1)
		d := resolveDialect(excelDialect(), dialectArg, kw)
		r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
		r.Comma = rune(d.Delimiter)
		r.LazyQuotes = true
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = d.SkipInitial
		rows, err := r.ReadAll()
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return &object.CSVReader{Rows: rows, Dialect: d}, nil
	}})

	// writer()
	m.Dict.SetStr("writer", &object.BuiltinFunc{Name: "writer", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "writer() missing target")
		}
		dialectArg := dialectKwargOrPos(a, kw, 1)
		d := resolveDialect(excelDialect(), dialectArg, kw)
		return &object.CSVWriter{Target: a[0], Dialect: d}, nil
	}})

	// DictReader()
	m.Dict.SetStr("DictReader", &object.BuiltinFunc{Name: "DictReader", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "DictReader() missing source")
		}
		lines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		d := resolveDialect(excelDialect(), nil, kw)
		r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
		r.Comma = rune(d.Delimiter)
		r.LazyQuotes = true
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = d.SkipInitial
		rows, err := r.ReadAll()
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}

		// Optional positional fieldnames arg (index 1).
		var fieldnames []string
		if len(a) >= 2 {
			if l, ok := a[1].(*object.List); ok {
				for _, v := range l.V {
					if s, ok := v.(*object.Str); ok {
						fieldnames = append(fieldnames, s.V)
					}
				}
			}
		}
		// Keyword fieldnames overrides positional.
		if kw != nil {
			if v, ok := kw.GetStr("fieldnames"); ok {
				if l, ok := v.(*object.List); ok {
					fieldnames = nil
					for _, v2 := range l.V {
						if s, ok := v2.(*object.Str); ok {
							fieldnames = append(fieldnames, s.V)
						}
					}
				}
			}
		}

		// If fieldnames given, all rows are data. Otherwise first row is header.
		if fieldnames != nil {
			// All rows are data.
		} else if len(rows) > 0 {
			fieldnames = rows[0]
			rows = rows[1:]
		}

		restkey := object.Object(object.None)
		restval := object.Object(object.None)
		if kw != nil {
			if v, ok := kw.GetStr("restkey"); ok {
				restkey = v
			}
			if v, ok := kw.GetStr("restval"); ok {
				restval = v
			}
		}

		return &object.CSVDictReader{
			Rows:       rows,
			Fieldnames: fieldnames,
			Dialect:    d,
			Restkey:    restkey,
			Restval:    restval,
		}, nil
	}})

	// DictWriter()
	m.Dict.SetStr("DictWriter", &object.BuiltinFunc{Name: "DictWriter", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "DictWriter() missing target")
		}
		var fieldnames []string
		if len(a) >= 2 {
			if l, ok := a[1].(*object.List); ok {
				for _, v := range l.V {
					if s, ok := v.(*object.Str); ok {
						fieldnames = append(fieldnames, s.V)
					}
				}
			}
		}
		if fieldnames == nil && kw != nil {
			if fv, ok := kw.GetStr("fieldnames"); ok {
				if l, ok := fv.(*object.List); ok {
					for _, v := range l.V {
						if s, ok := v.(*object.Str); ok {
							fieldnames = append(fieldnames, s.V)
						}
					}
				}
			}
		}
		if fieldnames == nil {
			return nil, object.Errorf(i.typeErr, "DictWriter() missing fieldnames")
		}
		d := resolveDialect(excelDialect(), nil, kw)
		extrasaction := "raise"
		if kw != nil {
			if v, ok := kw.GetStr("extrasaction"); ok {
				if s, ok := v.(*object.Str); ok {
					extrasaction = s.V
				}
			}
		}
		w := &object.CSVWriter{Target: a[0], Dialect: d}
		return &object.CSVDictWriter{Writer: w, Fieldnames: fieldnames, Extrasaction: extrasaction}, nil
	}})

	// register_dialect()
	m.Dict.SetStr("register_dialect", &object.BuiltinFunc{Name: "register_dialect", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "register_dialect() missing name")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "register_dialect() name must be str")
		}
		var base object.Object
		if len(a) >= 2 {
			base = a[1]
		}
		d := resolveDialect(excelDialect(), base, kw)
		csvDialectMu.Lock()
		csvDialectRegistry[name.V] = d
		csvDialectMu.Unlock()
		return object.None, nil
	}})

	// unregister_dialect()
	m.Dict.SetStr("unregister_dialect", &object.BuiltinFunc{Name: "unregister_dialect", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "unregister_dialect() missing name")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "unregister_dialect() name must be str")
		}
		csvDialectMu.Lock()
		_, exists := csvDialectRegistry[name.V]
		if exists {
			delete(csvDialectRegistry, name.V)
		}
		csvDialectMu.Unlock()
		if !exists {
			return nil, object.Errorf(csvErr, "unknown dialect")
		}
		return object.None, nil
	}})

	// get_dialect()
	m.Dict.SetStr("get_dialect", &object.BuiltinFunc{Name: "get_dialect", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "get_dialect() missing name")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "get_dialect() name must be str")
		}
		csvDialectMu.RLock()
		d, exists := csvDialectRegistry[name.V]
		csvDialectMu.RUnlock()
		if !exists {
			return nil, object.Errorf(csvErr, "unknown dialect: %s", name.V)
		}
		cp := *d
		return &object.CSVDialectObj{D: &cp}, nil
	}})

	// list_dialects()
	m.Dict.SetStr("list_dialects", &object.BuiltinFunc{Name: "list_dialects", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		csvDialectMu.RLock()
		names := make([]object.Object, 0, len(csvDialectRegistry))
		for k := range csvDialectRegistry {
			names = append(names, &object.Str{V: k})
		}
		csvDialectMu.RUnlock()
		return &object.List{V: names}, nil
	}})

	// field_size_limit()
	m.Dict.SetStr("field_size_limit", &object.BuiltinFunc{Name: "field_size_limit", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		old := csvFieldSizeLimit
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				csvFieldSizeLimit = int(n)
			}
		}
		return object.NewInt(int64(old)), nil
	}})

	// Sniffer class.
	snifferClass := &object.Class{Name: "Sniffer", Dict: object.NewDict()}
	snifferClass.Dict.SetStr("sniff", &object.BuiltinFunc{Name: "sniff", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// a[0] is self (when called as bound method); a[1] is sample (or a[0] if unbound)
		args := instanceArgs(a)
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "sniff() missing sample")
		}
		sample, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "sniff() sample must be str")
		}
		delimiters := ",\t|;:"
		if len(args) >= 2 {
			if s, ok := args[1].(*object.Str); ok {
				delimiters = s.V
			}
		}
		delim := csvSniffDelimiter(sample.V, delimiters)
		d := excelDialect()
		d.Delimiter = delim
		return &object.CSVDialectObj{D: d}, nil
	}})
	snifferClass.Dict.SetStr("has_header", &object.BuiltinFunc{Name: "has_header", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		args := instanceArgs(a)
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "has_header() missing sample")
		}
		sample, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "has_header() sample must be str")
		}
		return object.BoolOf(csvHasHeader(sample.V)), nil
	}})
	m.Dict.SetStr("Sniffer", &object.BuiltinFunc{Name: "Sniffer", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return &object.Instance{Class: snifferClass, Dict: object.NewDict()}, nil
	}})

	return m
}

// --- iter support for CSVDictReader -----------------------------------------

func csvDictReaderNextRow(dr *object.CSVDictReader) (object.Object, bool, error) {
	if dr.Pos >= len(dr.Rows) {
		return nil, false, nil
	}
	row := dr.Rows[dr.Pos]
	dr.Pos++

	dd := object.NewDict()
	fieldnames := dr.Fieldnames
	for k, name := range fieldnames {
		if k < len(row) {
			dd.SetStr(name, &object.Str{V: row[k]})
		} else {
			// Missing field: use restval.
			dd.SetStr(name, dr.Restval)
		}
	}
	// Extra fields beyond fieldnames: collect under restkey.
	if len(row) > len(fieldnames) {
		extra := make([]object.Object, len(row)-len(fieldnames))
		for k, v := range row[len(fieldnames):] {
			extra[k] = &object.Str{V: v}
		}
		if s, ok := dr.Restkey.(*object.Str); ok {
			dd.SetStr(s.V, &object.List{V: extra})
		}
	}
	return dd, true, nil
}

// csvDictReaderAttr handles attribute access on a CSVDictReader.
func csvDictReaderAttr(i *Interp, dr *object.CSVDictReader, name string) (object.Object, bool) {
	switch name {
	case "fieldnames":
		if dr.Fieldnames == nil {
			return object.None, true
		}
		v := make([]object.Object, len(dr.Fieldnames))
		for k, n := range dr.Fieldnames {
			v[k] = &object.Str{V: n}
		}
		return &object.List{V: v}, true
	case "restkey":
		return dr.Restkey, true
	case "restval":
		return dr.Restval, true
	case "dialect":
		return &object.CSVDialectObj{D: dr.Dialect}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return dr, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			v, ok, err := csvDictReaderNextRow(dr)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, object.NewException(i.stopIter, "")
			}
			return v, nil
		}}, true
	}
	return nil, false
}

// --- attribute access on CSVDialectObj --------------------------------------

func csvDialectObjAttr(d *object.CSVDialect, name string) (object.Object, bool) {
	switch name {
	case "delimiter":
		return &object.Str{V: string(d.Delimiter)}, true
	case "quotechar":
		if d.Quotechar == 0 {
			return object.None, true
		}
		return &object.Str{V: string(d.Quotechar)}, true
	case "doublequote":
		return object.BoolOf(d.Doublequote), true
	case "skipinitialspace":
		return object.BoolOf(d.SkipInitial), true
	case "lineterminator":
		return &object.Str{V: d.Lineterminator}, true
	case "quoting":
		return object.NewInt(int64(d.Quoting)), true
	case "escapechar":
		if d.Escapechar == 0 {
			return object.None, true
		}
		return &object.Str{V: string(d.Escapechar)}, true
	case "strict":
		return object.BoolOf(d.Strict), true
	}
	return nil, false
}

// --- writer attr handlers ---------------------------------------------------

// csvReaderAttr dispatches attribute access on a *object.CSVReader.
func csvReaderAttr(i *Interp, r *object.CSVReader, name string) (object.Object, bool) {
	switch name {
	case "line_num":
		return object.NewInt(int64(r.LineNo)), true
	case "dialect":
		return &object.CSVDialectObj{D: r.Dialect}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return r, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if r.Pos >= len(r.Rows) {
				exc := object.NewException(i.stopIter, "")
				return nil, exc
			}
			row := r.Rows[r.Pos]
			r.Pos++
			r.LineNo++
			v := make([]object.Object, len(row))
			for k, s := range row {
				v[k] = &object.Str{V: s}
			}
			return &object.List{V: v}, nil
		}}, true
	}
	return nil, false
}

// csvWriterAttr dispatches attribute access on a *object.CSVWriter.
func csvWriterAttr(i *Interp, w *object.CSVWriter, name string) (object.Object, bool) {
	switch name {
	case "dialect":
		return &object.CSVDialectObj{D: w.Dialect}, true
	case "writerow":
		return &object.BuiltinFunc{Name: "writerow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerow() missing row")
			}
			cells, err := pyRowToCsvCells(i, a[0])
			if err != nil {
				return nil, err
			}
			line, err := csvFormatRow(cells, w.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + w.Dialect.Lineterminator
			if err := csvWriteLine(i, w.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerows":
		return &object.BuiltinFunc{Name: "writerows", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerows() missing rows")
			}
			it, err := i.getIter(a[0])
			if err != nil {
				return nil, err
			}
			for {
				v, ok, err := it.Next()
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				cells, err := pyRowToCsvCells(i, v)
				if err != nil {
					return nil, err
				}
				line, err := csvFormatRow(cells, w.Dialect)
				if err != nil {
					return nil, err
				}
				if err := csvWriteLine(i, w.Target, line+w.Dialect.Lineterminator); err != nil {
					return nil, err
				}
			}
			return object.None, nil
		}}, true
	}
	return nil, false
}

// csvDictWriterAttr dispatches attribute access on *object.CSVDictWriter.
func csvDictWriterAttr(i *Interp, dw *object.CSVDictWriter, name string) (object.Object, bool) {
	switch name {
	case "fieldnames":
		v := make([]object.Object, len(dw.Fieldnames))
		for k, n := range dw.Fieldnames {
			v[k] = &object.Str{V: n}
		}
		return &object.List{V: v}, true
	case "dialect":
		return &object.CSVDialectObj{D: dw.Writer.Dialect}, true
	case "writeheader":
		return &object.BuiltinFunc{Name: "writeheader", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cells := make([]csvCell, len(dw.Fieldnames))
			for k, n := range dw.Fieldnames {
				cells[k] = csvCell{val: n}
			}
			line, err := csvFormatRow(cells, dw.Writer.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + dw.Writer.Dialect.Lineterminator
			if err := csvWriteLine(i, dw.Writer.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerow":
		return &object.BuiltinFunc{Name: "writerow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerow() missing row")
			}
			d, ok := a[0].(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "writerow() row must be dict")
			}
			if dw.Extrasaction != "ignore" {
				fieldSet := make(map[string]bool, len(dw.Fieldnames))
				for _, f := range dw.Fieldnames {
					fieldSet[f] = true
				}
				keys, _ := d.Items()
				var extras []string
				for _, k := range keys {
					if ks, ok := k.(*object.Str); ok && !fieldSet[ks.V] {
						extras = append(extras, ks.V)
					}
				}
				if len(extras) > 0 {
					return nil, object.Errorf(i.valueErr, "dict contains fields not in fieldnames: '%s'", strings.Join(extras, "', '"))
				}
			}
			cells := dictRowToCsvCells(d, dw.Fieldnames)
			line, err := csvFormatRow(cells, dw.Writer.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + dw.Writer.Dialect.Lineterminator
			if err := csvWriteLine(i, dw.Writer.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerows":
		return &object.BuiltinFunc{Name: "writerows", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerows() missing rows")
			}
			it, err := i.getIter(a[0])
			if err != nil {
				return nil, err
			}
			for {
				v, ok, err := it.Next()
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				d, ok := v.(*object.Dict)
				if !ok {
					return nil, object.Errorf(i.typeErr, "writerows() row must be dict")
				}
				cells := dictRowToCsvCells(d, dw.Fieldnames)
				line, err := csvFormatRow(cells, dw.Writer.Dialect)
				if err != nil {
					return nil, err
				}
				if err := csvWriteLine(i, dw.Writer.Target, line+dw.Writer.Dialect.Lineterminator); err != nil {
					return nil, err
				}
			}
			return object.None, nil
		}}, true
	}
	return nil, false
}

// --- row formatting ---------------------------------------------------------

// csvCell pairs a formatted string value with type metadata for quoting.
type csvCell struct {
	val     string
	numeric bool // int or float value (not quoted in QUOTE_NONNUMERIC)
	null    bool // None value (not quoted in QUOTE_NOTNULL)
}

func pyToCsvCellTyped(v object.Object) csvCell {
	switch x := v.(type) {
	case *object.Str:
		return csvCell{val: x.V}
	case *object.Int:
		return csvCell{val: x.V.String(), numeric: true}
	case *object.Float:
		return csvCell{val: formatFloatSimple(x.V), numeric: true}
	case *object.Bool:
		if x.V {
			return csvCell{val: "True"}
		}
		return csvCell{val: "False"}
	case *object.NoneType:
		return csvCell{val: "", null: true}
	}
	return csvCell{val: object.Repr(v)}
}

func pyRowToCsvCells(i *Interp, o object.Object) ([]csvCell, error) {
	it, err := i.getIter(o)
	if err != nil {
		return nil, err
	}
	var out []csvCell
	for {
		v, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, pyToCsvCellTyped(v))
	}
	return out, nil
}

func dictRowToCsvCells(d *object.Dict, fieldnames []string) []csvCell {
	out := make([]csvCell, len(fieldnames))
	for k, name := range fieldnames {
		if v, ok := d.GetStr(name); ok {
			out[k] = pyToCsvCellTyped(v)
		}
	}
	return out
}

// quoteField wraps a string in quotechar, doubling or escaping embedded quotes.
func quoteField(s string, d *object.CSVDialect) string {
	q := string(d.Quotechar)
	if d.Doublequote {
		s = strings.ReplaceAll(s, q, q+q)
	} else if d.Escapechar != 0 {
		s = strings.ReplaceAll(s, q, string(d.Escapechar)+q)
	}
	return q + s + q
}

// escapeFieldNone escapes special chars for QUOTE_NONE mode.
func escapeFieldNone(s string, d *object.CSVDialect) string {
	if d.Escapechar == 0 {
		return s
	}
	esc := string(d.Escapechar)
	s = strings.ReplaceAll(s, esc, esc+esc)
	s = strings.ReplaceAll(s, string(d.Delimiter), esc+string(d.Delimiter))
	if d.Quotechar != 0 {
		s = strings.ReplaceAll(s, string(d.Quotechar), esc+string(d.Quotechar))
	}
	return s
}

func csvFormatRow(cells []csvCell, d *object.CSVDialect) (string, error) {
	switch d.Quoting {
	case 0: // QUOTE_MINIMAL - use Go's csv.Writer
		row := make([]string, len(cells))
		for k, c := range cells {
			row[k] = c.val
		}
		return csvFormatMinimal(row, d)

	case 1: // QUOTE_ALL - quote everything
		parts := make([]string, len(cells))
		for k, c := range cells {
			parts[k] = quoteField(c.val, d)
		}
		return strings.Join(parts, string(d.Delimiter)), nil

	case 2, 4: // QUOTE_NONNUMERIC / QUOTE_STRINGS - quote non-numeric
		parts := make([]string, len(cells))
		for k, c := range cells {
			if c.numeric {
				parts[k] = c.val
			} else {
				parts[k] = quoteField(c.val, d)
			}
		}
		return strings.Join(parts, string(d.Delimiter)), nil

	case 3: // QUOTE_NONE - no quotes, escape specials
		parts := make([]string, len(cells))
		for k, c := range cells {
			parts[k] = escapeFieldNone(c.val, d)
		}
		return strings.Join(parts, string(d.Delimiter)), nil

	case 5: // QUOTE_NOTNULL - quote all non-None; None → empty bare
		parts := make([]string, len(cells))
		for k, c := range cells {
			if c.null {
				parts[k] = ""
			} else {
				parts[k] = quoteField(c.val, d)
			}
		}
		return strings.Join(parts, string(d.Delimiter)), nil
	}

	// Fallback: QUOTE_MINIMAL.
	row := make([]string, len(cells))
	for k, c := range cells {
		row[k] = c.val
	}
	return csvFormatMinimal(row, d)
}

func csvFormatMinimal(row []string, d *object.CSVDialect) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Comma = rune(d.Delimiter)
	if err := w.Write(row); err != nil {
		return "", err
	}
	w.Flush()
	out := buf.String()
	// Go's encoding/csv appends \n; strip all trailing CRLF/LF.
	out = strings.TrimRight(out, "\r\n")
	return out, nil
}

func csvWriteLine(i *Interp, target object.Object, line string) error {
	wr, err := i.getAttr(target, "write")
	if err != nil {
		return err
	}
	_, err = i.callObject(wr, []object.Object{&object.Str{V: line}}, nil)
	return err
}

// iterStrings collects an iterable of str (e.g. a list of lines or a
// StringIO) into a flat []string.
func (i *Interp) iterStrings(o object.Object) ([]string, error) {
	if sio, ok := o.(*object.StringIO); ok {
		return strings.Split(string(sio.V), "\n"), nil
	}
	it, err := i.getIter(o)
	if err != nil {
		return nil, err
	}
	var out []string
	for {
		v, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		s, ok := v.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "csv iterator yielded a non-str")
		}
		out = append(out, s.V)
	}
	return out, nil
}

func formatFloatSimple(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// --- Sniffer helpers --------------------------------------------------------

func csvSniffDelimiter(sample string, delimiters string) byte {
	lines := strings.Split(strings.TrimRight(sample, "\n"), "\n")
	if len(lines) == 0 {
		return ','
	}
	bestDelim := byte(',')
	bestScore := -1
	for _, d := range []byte(delimiters) {
		counts := make([]int, len(lines))
		for k, line := range lines {
			counts[k] = strings.Count(line, string(d))
		}
		if len(counts) == 0 || counts[0] == 0 {
			continue
		}
		consistent := true
		for _, c := range counts[1:] {
			if c != counts[0] {
				consistent = false
				break
			}
		}
		score := counts[0]
		if consistent && score > bestScore {
			bestScore = score
			bestDelim = d
		}
	}
	return bestDelim
}

func csvHasHeader(sample string) bool {
	r := csv.NewReader(strings.NewReader(sample))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil || len(rows) < 2 {
		return false
	}
	numericInRow0 := 0
	for _, f := range rows[0] {
		if csvIsNumeric(f) {
			numericInRow0++
		}
	}
	totalNumericRest := 0
	for _, row := range rows[1:] {
		for _, f := range row {
			if csvIsNumeric(f) {
				totalNumericRest++
			}
		}
	}
	avgNumericRest := float64(totalNumericRest) / float64(len(rows)-1)
	return float64(numericInRow0) < avgNumericRest
}

func csvIsNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
