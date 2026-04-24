package vm

// Python sqlite3 module — thin wrapper over Go's database/sql package.
// No SQL engine is bundled. Callers must register a compatible driver
// (e.g. modernc.org/sqlite as "sqlite3") before running Python scripts
// that import sqlite3.
//
// To change the driver name used by sqlite3.connect():
//
//	vm.SQLite3DriverName = "sqlite"

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

// SQLite3DriverName is passed to sql.Open when Python calls sqlite3.connect().
var SQLite3DriverName = "sqlite3"

func (i *Interp) buildSqlite3() *object.Module {
	m := &object.Module{Name: "sqlite3", Dict: object.NewDict()}

	mkExc := func(name string, parent *object.Class) *object.Class {
		cls := &object.Class{Name: name, Bases: []*object.Class{parent}, Dict: object.NewDict()}
		m.Dict.SetStr(name, cls)
		return cls
	}

	_ = mkExc("Warning", i.exception)
	errCls := mkExc("Error", i.exception)
	dbErrCls := mkExc("DatabaseError", errCls)
	opErrCls := mkExc("OperationalError", dbErrCls)
	intErrCls := mkExc("IntegrityError", dbErrCls)
	progErrCls := mkExc("ProgrammingError", dbErrCls)
	_ = mkExc("DataError", dbErrCls)
	_ = mkExc("InternalError", dbErrCls)
	_ = mkExc("NotSupportedError", dbErrCls)

	m.Dict.SetStr("sqlite_version", &object.Str{V: "3.0.0"})
	m.Dict.SetStr("sqlite_version_info", &object.Tuple{V: []object.Object{
		object.IntFromInt64(3), object.IntFromInt64(0), object.IntFromInt64(0),
	}})
	m.Dict.SetStr("PARSE_DECLTYPES", object.IntFromInt64(1))
	m.Dict.SetStr("PARSE_COLNAMES", object.IntFromInt64(2))

	rowCls := i.buildSqlite3RowClass()
	m.Dict.SetStr("Row", rowCls)

	connect := &object.BuiltinFunc{Name: "connect", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(progErrCls, "connect() requires database name")
		}
		s, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "database must be str")
		}
		db, err := sql.Open(SQLite3DriverName, s.V)
		if err != nil {
			return nil, object.Errorf(opErrCls, "%v", err)
		}
		return i.makeSqlite3Conn(db, opErrCls, intErrCls, progErrCls, rowCls), nil
	}}
	m.Dict.SetStr("connect", connect)

	return m
}

// ── Row class ─────────────────────────────────────────────────────────────────

func (i *Interp) buildSqlite3RowClass() *object.Class {
	cls := &object.Class{Name: "Row", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	// Row instances store _cols (*List of *Str) and _row (*Tuple) in their dict.
	getitem := &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "Row.__getitem__ requires key")
		}
		inst, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "expected Row instance")
		}
		rowObj, _ := inst.Dict.GetStr("_row")
		row, _ := rowObj.(*object.Tuple)
		if row == nil {
			return object.None, nil
		}
		switch k := args[1].(type) {
		case *object.Int:
			idx := int(k.Int64())
			if idx < 0 {
				idx += len(row.V)
			}
			if idx < 0 || idx >= len(row.V) {
				return nil, object.Errorf(i.indexErr, "index out of range")
			}
			return row.V[idx], nil
		case *object.Str:
			colsObj, _ := inst.Dict.GetStr("_cols")
			cols, _ := colsObj.(*object.List)
			if cols != nil {
				for ci, c := range cols.V {
					if cs, ok2 := c.(*object.Str); ok2 && strings.EqualFold(cs.V, k.V) {
						if ci < len(row.V) {
							return row.V[ci], nil
						}
					}
				}
			}
			return nil, object.Errorf(i.keyErr, "%q", k.V)
		}
		return nil, object.Errorf(i.typeErr, "Row indices must be int or str")
	}}
	cls.Dict.SetStr("__getitem__", getitem)

	keys := &object.BuiltinFunc{Name: "keys", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return &object.List{}, nil
		}
		inst, ok := args[0].(*object.Instance)
		if !ok {
			return &object.List{}, nil
		}
		colsObj, _ := inst.Dict.GetStr("_cols")
		cols, _ := colsObj.(*object.List)
		if cols == nil {
			return &object.List{}, nil
		}
		cp := make([]object.Object, len(cols.V))
		copy(cp, cols.V)
		return &object.List{V: cp}, nil
	}}
	cls.Dict.SetStr("keys", keys)

	lenFn := &object.BuiltinFunc{Name: "__len__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return object.IntFromInt64(0), nil
		}
		inst, ok := args[0].(*object.Instance)
		if !ok {
			return object.IntFromInt64(0), nil
		}
		rowObj, _ := inst.Dict.GetStr("_row")
		if row, ok2 := rowObj.(*object.Tuple); ok2 {
			return object.IntFromInt64(int64(len(row.V))), nil
		}
		return object.IntFromInt64(0), nil
	}}
	cls.Dict.SetStr("__len__", lenFn)

	iterFn := &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}, nil
		}
		inst, ok := args[0].(*object.Instance)
		if !ok {
			return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}, nil
		}
		rowObj, _ := inst.Dict.GetStr("_row")
		row, _ := rowObj.(*object.Tuple)
		pos := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if row == nil || pos >= len(row.V) {
				return nil, false, nil
			}
			v := row.V[pos]
			pos++
			return v, true, nil
		}}, nil
	}}
	cls.Dict.SetStr("__iter__", iterFn)

	return cls
}

// ── Connection ────────────────────────────────────────────────────────────────

func (i *Interp) makeSqlite3Conn(db *sql.DB, opErr, intErr, progErr *object.Class, rowCls *object.Class) *object.Instance {
	cls := &object.Class{Name: "Connection", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	closed := false

	wrapErr := func(err error) error {
		if err == nil {
			return nil
		}
		msg := err.Error()
		if strings.Contains(msg, "UNIQUE") || strings.Contains(msg, "FOREIGN KEY") ||
			strings.Contains(msg, "NOT NULL") || strings.Contains(msg, "constraint") {
			return object.Errorf(intErr, "%v", err)
		}
		return object.Errorf(opErr, "%v", err)
	}

	checkOpen := func() error {
		if closed {
			return object.Errorf(progErr, "cannot operate on a closed database")
		}
		return nil
	}

	getRowFactory := func() object.Object {
		rf, _ := d.GetStr("row_factory")
		return rf
	}

	newCursor := func() *object.Instance {
		return i.makeSqlite3Cursor(db, opErr, progErr, rowCls, wrapErr, getRowFactory)
	}

	cursor := &object.BuiltinFunc{Name: "cursor", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		cur := newCursor()
		cur.Dict.SetStr("connection", inst)
		return cur, nil
	}}
	d.SetStr("cursor", cursor)

	execute := &object.BuiltinFunc{Name: "execute", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		cur := newCursor()
		cur.Dict.SetStr("connection", inst)
		execFn, _ := cur.Dict.GetStr("execute")
		if bf, ok := execFn.(*object.BuiltinFunc); ok {
			if _, err := bf.Call(nil, append([]object.Object{cur}, args...), nil); err != nil {
				return nil, err
			}
		}
		return cur, nil
	}}
	d.SetStr("execute", execute)

	executemany := &object.BuiltinFunc{Name: "executemany", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		cur := newCursor()
		cur.Dict.SetStr("connection", inst)
		execFn, _ := cur.Dict.GetStr("executemany")
		if bf, ok := execFn.(*object.BuiltinFunc); ok {
			if _, err := bf.Call(nil, append([]object.Object{cur}, args...), nil); err != nil {
				return nil, err
			}
		}
		return cur, nil
	}}
	d.SetStr("executemany", executemany)

	executescript := &object.BuiltinFunc{Name: "executescript", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(args) < 1 {
			return nil, object.Errorf(progErr, "executescript() requires script string")
		}
		s, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "executescript() requires str")
		}
		cur := newCursor()
		cur.Dict.SetStr("connection", inst)
		if err := sqlite3ExecScript(db, s.V, wrapErr); err != nil {
			return nil, err
		}
		return cur, nil
	}}
	d.SetStr("executescript", executescript)

	commit := &object.BuiltinFunc{Name: "commit", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		return object.None, nil
	}}
	d.SetStr("commit", commit)

	rollback := &object.BuiltinFunc{Name: "rollback", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		return object.None, nil
	}}
	d.SetStr("rollback", rollback)

	closeFn := &object.BuiltinFunc{Name: "close", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if !closed {
			closed = true
			db.Close()
		}
		return object.None, nil
	}}
	d.SetStr("close", closeFn)

	enter := &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}}
	d.SetStr("__enter__", enter)

	exit := &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}}
	d.SetStr("__exit__", exit)

	d.SetStr("row_factory", object.None)
	d.SetStr("isolation_level", object.None)
	d.SetStr("in_transaction", object.False)

	return inst
}

// ── Cursor ────────────────────────────────────────────────────────────────────

func (i *Interp) makeSqlite3Cursor(
	db *sql.DB,
	opErr, progErr *object.Class,
	rowCls *object.Class,
	wrapErr func(error) error,
	getRowFactory func() object.Object,
) *object.Instance {
	cls := &object.Class{Name: "Cursor", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	// Mutable cursor state
	var bufferedRows [][]object.Object
	var colNames []string
	pos := 0
	rowcount := int64(-1)
	lastrowid := int64(0)
	arraysize := 1

	syncDesc := func() {
		var descItems []object.Object
		for _, c := range colNames {
			descItems = append(descItems, &object.Tuple{V: []object.Object{
				&object.Str{V: c}, object.None, object.None, object.None,
				object.None, object.None, object.None,
			}})
		}
		if len(descItems) > 0 {
			d.SetStr("description", &object.Tuple{V: descItems})
		} else {
			d.SetStr("description", object.None)
		}
		d.SetStr("rowcount", object.IntFromInt64(rowcount))
		d.SetStr("lastrowid", object.IntFromInt64(lastrowid))
		d.SetStr("arraysize", object.IntFromInt64(int64(arraysize)))
	}
	syncDesc()
	d.SetStr("connection", object.None)

	makeRow := func(row []object.Object) object.Object {
		rf := getRowFactory()
		if rf != nil && rf != object.None {
			if rfCls, ok := rf.(*object.Class); ok && rfCls == rowCls {
				colObjs := make([]object.Object, len(colNames))
				for ci, c := range colNames {
					colObjs[ci] = &object.Str{V: c}
				}
				rd := object.NewDict()
				rd.SetStr("_cols", &object.List{V: colObjs})
				rd.SetStr("_row", &object.Tuple{V: row})
				return &object.Instance{Class: rowCls, Dict: rd}
			}
		}
		return &object.Tuple{V: row}
	}

	doExec := func(sqlStr string, params []any) error {
		trimmed := strings.TrimSpace(sqlStr)
		upper := strings.ToUpper(trimmed)
		isSelect := strings.HasPrefix(upper, "SELECT") ||
			strings.HasPrefix(upper, "WITH") ||
			strings.HasPrefix(upper, "PRAGMA") ||
			strings.HasPrefix(upper, "EXPLAIN")

		if isSelect {
			sqlRows, err := db.Query(sqlStr, params...)
			if err != nil {
				return wrapErr(err)
			}
			defer sqlRows.Close()
			cols, err := sqlRows.Columns()
			if err != nil {
				return wrapErr(err)
			}
			colNames = cols
			bufferedRows = nil
			pos = 0
			for sqlRows.Next() {
				raw := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for ci := range raw {
					ptrs[ci] = &raw[ci]
				}
				if err := sqlRows.Scan(ptrs...); err != nil {
					return wrapErr(err)
				}
				row := make([]object.Object, len(cols))
				for ci, v := range raw {
					row[ci] = sqlite3GoToObj(v)
				}
				bufferedRows = append(bufferedRows, row)
			}
			if err := sqlRows.Err(); err != nil {
				return wrapErr(err)
			}
			rowcount = -1
			lastrowid = 0
		} else {
			result, err := db.Exec(sqlStr, params...)
			if err != nil {
				return wrapErr(err)
			}
			affected, _ := result.RowsAffected()
			lid, _ := result.LastInsertId()
			rowcount = affected
			lastrowid = lid
			colNames = nil
			bufferedRows = nil
			pos = 0
		}
		syncDesc()
		return nil
	}

	execute := &object.BuiltinFunc{Name: "execute", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		// args[0]=self (when called via connection.execute dispatch), else args[0]=sql
		// Detect: if first arg is *Instance (self), skip it
		actualArgs := args
		if len(actualArgs) > 0 {
			if _, isSelf := actualArgs[0].(*object.Instance); isSelf {
				actualArgs = actualArgs[1:]
			}
		}
		if len(actualArgs) < 1 {
			return nil, object.Errorf(progErr, "execute() requires SQL string")
		}
		sqlStr, ok := actualArgs[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "SQL must be str")
		}
		var params []any
		if len(actualArgs) >= 2 {
			p, err := sqlite3ConvertParams(actualArgs[1])
			if err != nil {
				return nil, object.Errorf(progErr, "%v", err)
			}
			params = p
		}
		if err := doExec(sqlStr.V, params); err != nil {
			return nil, err
		}
		return inst, nil
	}}
	d.SetStr("execute", execute)

	executemany := &object.BuiltinFunc{Name: "executemany", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		actualArgs := args
		if len(actualArgs) > 0 {
			if _, isSelf := actualArgs[0].(*object.Instance); isSelf {
				actualArgs = actualArgs[1:]
			}
		}
		if len(actualArgs) < 2 {
			return nil, object.Errorf(progErr, "executemany() requires SQL and sequence")
		}
		sqlStr, ok := actualArgs[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "SQL must be str")
		}
		var rows []object.Object
		switch seq := actualArgs[1].(type) {
		case *object.List:
			rows = seq.V
		case *object.Tuple:
			rows = seq.V
		default:
			return nil, object.Errorf(i.typeErr, "executemany() second arg must be iterable")
		}
		var totalAffected int64
		for _, rowObj := range rows {
			p, err := sqlite3ConvertParams(rowObj)
			if err != nil {
				return nil, object.Errorf(progErr, "%v", err)
			}
			result, err := db.Exec(sqlStr.V, p...)
			if err != nil {
				return nil, wrapErr(err)
			}
			n, _ := result.RowsAffected()
			totalAffected += n
		}
		rowcount = totalAffected
		syncDesc()
		return inst, nil
	}}
	d.SetStr("executemany", executemany)

	fetchone := &object.BuiltinFunc{Name: "fetchone", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if pos >= len(bufferedRows) {
			return object.None, nil
		}
		row := bufferedRows[pos]
		pos++
		return makeRow(row), nil
	}}
	d.SetStr("fetchone", fetchone)

	fetchall := &object.BuiltinFunc{Name: "fetchall", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		var result []object.Object
		for pos < len(bufferedRows) {
			result = append(result, makeRow(bufferedRows[pos]))
			pos++
		}
		return &object.List{V: result}, nil
	}}
	d.SetStr("fetchall", fetchall)

	fetchmany := &object.BuiltinFunc{Name: "fetchmany", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		size := arraysize
		if len(args) >= 1 {
			if n, ok := args[0].(*object.Int); ok {
				size = int(n.Int64())
			}
		}
		var result []object.Object
		for count := 0; count < size && pos < len(bufferedRows); count++ {
			result = append(result, makeRow(bufferedRows[pos]))
			pos++
		}
		return &object.List{V: result}, nil
	}}
	d.SetStr("fetchmany", fetchmany)

	iterFn := &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if pos >= len(bufferedRows) {
				return nil, false, nil
			}
			row := bufferedRows[pos]
			pos++
			return makeRow(row), true, nil
		}}, nil
	}}
	d.SetStr("__iter__", iterFn)

	closeFn := &object.BuiltinFunc{Name: "close", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		bufferedRows = nil
		pos = 0
		return object.None, nil
	}}
	d.SetStr("close", closeFn)

	return inst
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sqlite3ExecScript(db *sql.DB, script string, wrapErr func(error) error) error {
	for _, stmt := range strings.Split(script, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return wrapErr(err)
		}
	}
	return nil
}

func sqlite3ConvertParams(obj object.Object) ([]any, error) {
	switch v := obj.(type) {
	case *object.Tuple:
		out := make([]any, len(v.V))
		for idx, item := range v.V {
			out[idx] = sqlite3ObjToGo(item)
		}
		return out, nil
	case *object.List:
		out := make([]any, len(v.V))
		for idx, item := range v.V {
			out[idx] = sqlite3ObjToGo(item)
		}
		return out, nil
	}
	return nil, fmt.Errorf("params must be a sequence (tuple or list)")
}

func sqlite3ObjToGo(obj object.Object) any {
	switch v := obj.(type) {
	case *object.NoneType:
		return nil
	case *object.Bool:
		if v.V {
			return int64(1)
		}
		return int64(0)
	case *object.Int:
		return v.Int64()
	case *object.Float:
		return v.V
	case *object.Str:
		return v.V
	case *object.Bytes:
		cp := make([]byte, len(v.V))
		copy(cp, v.V)
		return cp
	default:
		return fmt.Sprintf("%v", obj)
	}
}

func sqlite3GoToObj(v any) object.Object {
	if v == nil {
		return object.None
	}
	switch n := v.(type) {
	case int64:
		return object.IntFromInt64(n)
	case float64:
		return &object.Float{V: n}
	case string:
		return &object.Str{V: n}
	case []byte:
		cp := make([]byte, len(n))
		copy(cp, n)
		return &object.Bytes{V: cp}
	case bool:
		return object.BoolOf(n)
	default:
		return &object.Str{V: fmt.Sprintf("%v", n)}
	}
}
