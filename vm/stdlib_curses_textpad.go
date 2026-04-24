package vm

import (
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// textboxState holds the internal buffer and cursor for a Textbox instance.
type textboxState struct {
	win        object.Object
	insertMode bool
	lines      [][]rune // text content [row][col]
	cy, cx     int      // cursor position
	maxy, maxx int      // window dimensions
	done       bool
}

func newTextboxState(win object.Object, insertMode bool) *textboxState {
	maxy, maxx := getWinMaxYX(win)
	if maxy < 1 {
		maxy = 1
	}
	if maxx < 1 {
		maxx = 1
	}
	lines := make([][]rune, maxy)
	for r := range lines {
		lines[r] = make([]rune, maxx)
		for c := range lines[r] {
			lines[r][c] = ' '
		}
	}
	return &textboxState{
		win:        win,
		insertMode: insertMode,
		lines:      lines,
		cy:         0,
		cx:         0,
		maxy:       maxy,
		maxx:       maxx,
	}
}

// getWinMaxYX calls win.getmaxyx() via the stub method and returns (lines, cols).
func getWinMaxYX(win object.Object) (int, int) {
	inst, ok := win.(*object.Instance)
	if !ok {
		return 24, 80
	}
	fn, ok2 := inst.Dict.GetStr("getmaxyx")
	if !ok2 {
		return 24, 80
	}
	bf, ok3 := fn.(*object.BuiltinFunc)
	if !ok3 {
		return 24, 80
	}
	result, err := bf.Call(nil, []object.Object{inst}, nil)
	if err != nil {
		return 24, 80
	}
	tup, ok4 := result.(*object.Tuple)
	if !ok4 || len(tup.V) < 2 {
		return 24, 80
	}
	y, _ := toInt64(tup.V[0])
	x, _ := toInt64(tup.V[1])
	return int(y), int(x)
}

// gather returns the current buffer contents: strips trailing spaces per line,
// joins with newlines, strips trailing newlines/spaces.
func (tb *textboxState) gather() string {
	rows := make([]string, 0, len(tb.lines))
	for _, line := range tb.lines {
		s := strings.TrimRight(string(line), " ")
		rows = append(rows, s)
	}
	return strings.TrimRight(strings.Join(rows, "\n"), "\n ")
}

// doCommand processes one keystroke, mirroring CPython's Textbox.do_command.
func (tb *textboxState) doCommand(ch int) {
	if tb.done {
		return
	}
	switch ch {
	case 1: // Ctrl-A — beginning of line
		tb.cx = 0
	case 2: // Ctrl-B — backward char
		if tb.cx > 0 {
			tb.cx--
		} else if tb.cy > 0 {
			tb.cy--
			tb.cx = tb.maxx - 1
		}
	case 4: // Ctrl-D — delete char at cursor
		line := tb.lines[tb.cy]
		if tb.cx < len(line)-1 {
			copy(line[tb.cx:], line[tb.cx+1:])
			line[len(line)-1] = ' '
		} else if tb.cx == len(line)-1 {
			line[tb.cx] = ' '
		}
	case 5: // Ctrl-E — end of line
		// Move to last non-space char + 1, or end of line
		line := tb.lines[tb.cy]
		end := len(line)
		for end > 0 && line[end-1] == ' ' {
			end--
		}
		tb.cx = end
		if tb.cx >= tb.maxx {
			tb.cx = tb.maxx - 1
		}
	case 6: // Ctrl-F — forward char
		if tb.cx < tb.maxx-1 {
			tb.cx++
		} else if tb.cy < tb.maxy-1 {
			tb.cy++
			tb.cx = 0
		}
	case 7: // Ctrl-G — terminate
		tb.done = true
	case 8, 263: // Ctrl-H / KEY_BACKSPACE — delete backward
		if tb.cx > 0 {
			line := tb.lines[tb.cy]
			copy(line[tb.cx-1:], line[tb.cx:])
			line[len(line)-1] = ' '
			tb.cx--
		} else if tb.cy > 0 {
			tb.cy--
			tb.cx = tb.maxx - 1
		}
	case 10, 13: // Ctrl-J / Ctrl-M — newline or terminate
		if tb.maxy == 1 {
			tb.done = true
		} else if tb.cy < tb.maxy-1 {
			tb.cy++
			tb.cx = 0
		}
	case 11: // Ctrl-K — kill to end of line
		line := tb.lines[tb.cy]
		for c := tb.cx; c < len(line); c++ {
			line[c] = ' '
		}
	case 12: // Ctrl-L — refresh/redraw (no-op in stub)
		// nothing to do in non-interactive mode
	case 14: // Ctrl-N — next line
		if tb.cy < tb.maxy-1 {
			tb.cy++
		}
	case 15: // Ctrl-O — insert blank line
		if tb.cy < tb.maxy-1 {
			// shift lines down, lose the last one
			copy(tb.lines[tb.cy+2:], tb.lines[tb.cy+1:])
			newLine := make([]rune, tb.maxx)
			for c := range newLine {
				newLine[c] = ' '
			}
			tb.lines[tb.cy+1] = newLine
		}
	case 16: // Ctrl-P — previous line
		if tb.cy > 0 {
			tb.cy--
		}
	default:
		// Printable character: insert at cursor (always shift right)
		r := rune(ch)
		if unicode.IsPrint(r) || ch > 31 {
			if tb.cx < tb.maxx {
				line := tb.lines[tb.cy]
				// Always shift right to insert (preserves existing chars)
				copy(line[tb.cx+1:], line[tb.cx:len(line)-1])
				line[tb.cx] = r
				tb.cx++
				if tb.cx >= tb.maxx {
					tb.cx = tb.maxx - 1
					if tb.cy < tb.maxy-1 {
						tb.cy++
						tb.cx = 0
					}
				}
			}
		}
	}
}

// callWinMethod calls a named method on a curses window instance with given args.
func callWinMethod(win object.Object, method string, args ...object.Object) {
	inst, ok := win.(*object.Instance)
	if !ok {
		return
	}
	fn, ok2 := inst.Dict.GetStr(method)
	if !ok2 {
		return
	}
	bf, ok3 := fn.(*object.BuiltinFunc)
	if !ok3 {
		return
	}
	allArgs := make([]object.Object, 0, 1+len(args))
	allArgs = append(allArgs, inst)
	allArgs = append(allArgs, args...)
	bf.Call(nil, allArgs, nil) //nolint:errcheck
}

// buildCursesTextpad constructs the curses.textpad submodule.
func (i *Interp) buildCursesTextpad() *object.Module {
	m := &object.Module{Name: "curses.textpad", Dict: object.NewDict()}

	// rectangle(win, uly, ulx, lry, lrx) → None
	m.Dict.SetStr("rectangle", &object.BuiltinFunc{Name: "rectangle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 5 {
			return nil, object.Errorf(i.typeErr, "rectangle() requires 5 arguments")
		}
		win := a[0]
		uly, _ := toInt64(a[1])
		ulx, _ := toInt64(a[2])
		lry, _ := toInt64(a[3])
		lrx, _ := toInt64(a[4])

		// Default ASCII fallback characters
		var hline object.Object = object.NewInt(int64('-'))
		var vline object.Object = object.NewInt(int64('|'))
		var ulc object.Object = object.NewInt(int64('+'))
		var urc object.Object = object.NewInt(int64('+'))
		var llc object.Object = object.NewInt(int64('+'))
		var lrc object.Object = object.NewInt(int64('+'))

		// Try to get actual ACS constants from the curses module
		if cursesM, ok2 := i.modules["curses"]; ok2 {
			getConst := func(name string) object.Object {
				if v, ok3 := cursesM.Dict.GetStr(name); ok3 {
					return v
				}
				return nil
			}
			if v := getConst("ACS_HLINE"); v != nil {
				hline = v
			}
			if v := getConst("ACS_VLINE"); v != nil {
				vline = v
			}
			if v := getConst("ACS_ULCORNER"); v != nil {
				ulc = v
			}
			if v := getConst("ACS_URCORNER"); v != nil {
				urc = v
			}
			if v := getConst("ACS_LLCORNER"); v != nil {
				llc = v
			}
			if v := getConst("ACS_LRCORNER"); v != nil {
				lrc = v
			}
		}

		height := lry - uly
		width := lrx - ulx

		if height > 1 {
			callWinMethod(win, "vline", object.NewInt(uly+1), object.NewInt(ulx), vline, object.NewInt(height-1))
			callWinMethod(win, "vline", object.NewInt(uly+1), object.NewInt(lrx), vline, object.NewInt(height-1))
		}
		if width > 1 {
			callWinMethod(win, "hline", object.NewInt(uly), object.NewInt(ulx+1), hline, object.NewInt(width-1))
			callWinMethod(win, "hline", object.NewInt(lry), object.NewInt(ulx+1), hline, object.NewInt(width-1))
		}
		callWinMethod(win, "addch", object.NewInt(uly), object.NewInt(ulx), ulc)
		callWinMethod(win, "addch", object.NewInt(uly), object.NewInt(lrx), urc)
		callWinMethod(win, "addch", object.NewInt(lry), object.NewInt(ulx), llc)
		callWinMethod(win, "addch", object.NewInt(lry), object.NewInt(lrx), lrc)

		return object.None, nil
	}})

	// Textbox class
	textboxCls := &object.Class{Name: "Textbox", Dict: object.NewDict()}

	// __init__(self, win, insert_mode=False) → None
	textboxCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		var win object.Object = object.None
		if len(a) >= 2 {
			win = a[1]
		}

		insertMode := false
		if len(a) >= 3 {
			if b, ok2 := a[2].(*object.Bool); ok2 {
				insertMode = b == object.True
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("insert_mode"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					insertMode = b == object.True
				}
			}
		}

		tb := newTextboxState(win, insertMode)

		self.Dict.SetStr("win", win)
		self.Dict.SetStr("insert_mode", object.BoolOf(insertMode))

		// do_command(ch) → None
		self.Dict.SetStr("do_command", &object.BuiltinFunc{Name: "do_command", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) < 1 {
				return object.None, nil
			}
			// a2[0] is self when called as method, a2[1] is ch
			var chObj object.Object
			if len(a2) >= 2 {
				chObj = a2[1]
			} else {
				chObj = a2[0]
			}
			ch, ok2 := toInt64(chObj)
			if !ok2 {
				return object.None, nil
			}
			tb.doCommand(int(ch))
			return object.None, nil
		}})

		// gather() → str
		self.Dict.SetStr("gather", &object.BuiltinFunc{Name: "gather", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: tb.gather()}, nil
		}})

		// edit(validate=None) → str
		// Non-interactive: just return current buffer
		self.Dict.SetStr("edit", &object.BuiltinFunc{Name: "edit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: tb.gather()}, nil
		}})

		return object.None, nil
	}})

	m.Dict.SetStr("Textbox", textboxCls)

	return m
}
