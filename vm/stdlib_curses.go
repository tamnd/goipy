package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// cursesWindowState holds the dimensions and cursor position of a stub window.
type cursesWindowState struct {
	lines int
	cols  int
	begy  int
	begx  int
}

// makeCursesWindow creates a curses.window instance with all methods stubbed.
func (i *Interp) makeCursesWindow(cls *object.Class, lines, cols, begy, begx int) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	st := &cursesWindowState{lines: lines, cols: cols, begy: begy, begx: begx}

	none := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	inst.Dict.SetStr("addch", none("addch"))
	inst.Dict.SetStr("addstr", none("addstr"))
	inst.Dict.SetStr("addnstr", none("addnstr"))
	inst.Dict.SetStr("attroff", none("attroff"))
	inst.Dict.SetStr("attron", none("attron"))
	inst.Dict.SetStr("attrset", none("attrset"))
	inst.Dict.SetStr("bkgd", none("bkgd"))
	inst.Dict.SetStr("bkgdset", none("bkgdset"))
	inst.Dict.SetStr("border", none("border"))
	inst.Dict.SetStr("box", none("box"))
	inst.Dict.SetStr("clear", none("clear"))
	inst.Dict.SetStr("clearok", none("clearok"))
	inst.Dict.SetStr("clrtobot", none("clrtobot"))
	inst.Dict.SetStr("clrtoeol", none("clrtoeol"))
	inst.Dict.SetStr("cursyncup", none("cursyncup"))
	inst.Dict.SetStr("delch", none("delch"))
	inst.Dict.SetStr("deleteln", none("deleteln"))
	inst.Dict.SetStr("echochar", none("echochar"))
	inst.Dict.SetStr("erase", none("erase"))
	inst.Dict.SetStr("idcok", none("idcok"))
	inst.Dict.SetStr("idlok", none("idlok"))
	inst.Dict.SetStr("immedok", none("immedok"))
	inst.Dict.SetStr("insch", none("insch"))
	inst.Dict.SetStr("insdelln", none("insdelln"))
	inst.Dict.SetStr("insertln", none("insertln"))
	inst.Dict.SetStr("insnstr", none("insnstr"))
	inst.Dict.SetStr("insstr", none("insstr"))
	inst.Dict.SetStr("is_linetouched", &object.BuiltinFunc{Name: "is_linetouched", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
	inst.Dict.SetStr("is_wintouched", &object.BuiltinFunc{Name: "is_wintouched", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
	inst.Dict.SetStr("keypad", none("keypad"))
	inst.Dict.SetStr("leaveok", none("leaveok"))
	inst.Dict.SetStr("move", none("move"))
	inst.Dict.SetStr("mvderwin", none("mvderwin"))
	inst.Dict.SetStr("mvwin", none("mvwin"))
	inst.Dict.SetStr("nodelay", none("nodelay"))
	inst.Dict.SetStr("notimeout", none("notimeout"))
	inst.Dict.SetStr("noutrefresh", none("noutrefresh"))
	inst.Dict.SetStr("overlay", none("overlay"))
	inst.Dict.SetStr("overwrite", none("overwrite"))
	inst.Dict.SetStr("putwin", none("putwin"))
	inst.Dict.SetStr("redrawln", none("redrawln"))
	inst.Dict.SetStr("redrawwin", none("redrawwin"))
	inst.Dict.SetStr("refresh", none("refresh"))
	inst.Dict.SetStr("resize", none("resize"))
	inst.Dict.SetStr("scroll", none("scroll"))
	inst.Dict.SetStr("scrollok", none("scrollok"))
	inst.Dict.SetStr("setscrreg", none("setscrreg"))
	inst.Dict.SetStr("standend", none("standend"))
	inst.Dict.SetStr("standout", none("standout"))
	inst.Dict.SetStr("syncdown", none("syncdown"))
	inst.Dict.SetStr("syncok", none("syncok"))
	inst.Dict.SetStr("syncup", none("syncup"))
	inst.Dict.SetStr("timeout", none("timeout"))
	inst.Dict.SetStr("touchline", none("touchline"))
	inst.Dict.SetStr("touchwin", none("touchwin"))
	inst.Dict.SetStr("untouchwin", none("untouchwin"))

	inst.Dict.SetStr("enclose", &object.BuiltinFunc{Name: "enclose", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	inst.Dict.SetStr("getbegyx", &object.BuiltinFunc{Name: "getbegyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(int64(st.begy)), object.NewInt(int64(st.begx))}}, nil
	}})
	inst.Dict.SetStr("getch", &object.BuiltinFunc{Name: "getch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(-1), nil
	}})
	inst.Dict.SetStr("getkey", &object.BuiltinFunc{Name: "getkey", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})
	inst.Dict.SetStr("getmaxyx", &object.BuiltinFunc{Name: "getmaxyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(int64(st.lines)), object.NewInt(int64(st.cols))}}, nil
	}})
	inst.Dict.SetStr("getparyx", &object.BuiltinFunc{Name: "getparyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(-1), object.NewInt(-1)}}, nil
	}})
	inst.Dict.SetStr("getstr", &object.BuiltinFunc{Name: "getstr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{}}, nil
	}})
	inst.Dict.SetStr("getyx", &object.BuiltinFunc{Name: "getyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
	}})
	inst.Dict.SetStr("hline", none("hline"))
	inst.Dict.SetStr("inch", &object.BuiltinFunc{Name: "inch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})
	inst.Dict.SetStr("instr", &object.BuiltinFunc{Name: "instr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{}}, nil
	}})
	inst.Dict.SetStr("vline", none("vline"))

	// subwin/derwin/subpad create sub-windows
	subwinFn := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nl, nc, by, bx := st.lines, st.cols, st.begy, st.begx
			if len(a) >= 4 {
				if n, ok := toInt64(a[0]); ok {
					nl = int(n)
				}
				if n, ok := toInt64(a[1]); ok {
					nc = int(n)
				}
				if n, ok := toInt64(a[2]); ok {
					by = int(n)
				}
				if n, ok := toInt64(a[3]); ok {
					bx = int(n)
				}
			}
			return i.makeCursesWindow(cls, nl, nc, by, bx), nil
		}}
	}
	inst.Dict.SetStr("subwin", subwinFn("subwin"))
	inst.Dict.SetStr("derwin", subwinFn("derwin"))
	inst.Dict.SetStr("subpad", subwinFn("subpad"))

	return inst
}

// buildCurses constructs the curses module stub.
func (i *Interp) buildCurses() *object.Module {
	m := &object.Module{Name: "curses", Dict: object.NewDict()}

	// window class
	windowCls := &object.Class{Name: "window", Dict: object.NewDict()}
	m.Dict.SetStr("window", windowCls)

	// error exception — subclass of Exception
	errorCls := &object.Class{Name: "error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("error", errorCls)

	// --- Attribute constants ---
	m.Dict.SetStr("A_NORMAL", object.NewInt(0))
	m.Dict.SetStr("A_ALTCHARSET", object.NewInt(1<<22))
	m.Dict.SetStr("A_BLINK", object.NewInt(1<<19))
	m.Dict.SetStr("A_BOLD", object.NewInt(1<<21))
	m.Dict.SetStr("A_DIM", object.NewInt(1<<20))
	m.Dict.SetStr("A_INVIS", object.NewInt(1<<23))
	m.Dict.SetStr("A_ITALIC", object.NewInt(1<<30))
	m.Dict.SetStr("A_PROTECT", object.NewInt(1<<24))
	m.Dict.SetStr("A_REVERSE", object.NewInt(1<<18))
	m.Dict.SetStr("A_STANDOUT", object.NewInt(1<<16))
	m.Dict.SetStr("A_UNDERLINE", object.NewInt(1<<17))
	m.Dict.SetStr("A_HORIZONTAL", object.NewInt(1<<25))
	m.Dict.SetStr("A_LEFT", object.NewInt(1<<26))
	m.Dict.SetStr("A_LOW", object.NewInt(1<<27))
	m.Dict.SetStr("A_RIGHT", object.NewInt(1<<28))
	m.Dict.SetStr("A_TOP", object.NewInt(1<<29))
	m.Dict.SetStr("A_VERTICAL", object.NewInt(1<<31))
	m.Dict.SetStr("A_CHARTEXT", object.NewInt(0xFF))

	// --- Color constants ---
	m.Dict.SetStr("COLOR_BLACK", object.NewInt(0))
	m.Dict.SetStr("COLOR_RED", object.NewInt(1))
	m.Dict.SetStr("COLOR_GREEN", object.NewInt(2))
	m.Dict.SetStr("COLOR_YELLOW", object.NewInt(3))
	m.Dict.SetStr("COLOR_BLUE", object.NewInt(4))
	m.Dict.SetStr("COLOR_MAGENTA", object.NewInt(5))
	m.Dict.SetStr("COLOR_CYAN", object.NewInt(6))
	m.Dict.SetStr("COLOR_WHITE", object.NewInt(7))
	m.Dict.SetStr("COLORS", object.NewInt(8))
	m.Dict.SetStr("COLOR_PAIRS", object.NewInt(64))

	// --- Size constants ---
	m.Dict.SetStr("COLS", object.NewInt(80))
	m.Dict.SetStr("LINES", object.NewInt(24))

	// --- Error/OK ---
	m.Dict.SetStr("ERR", object.NewInt(-1))
	m.Dict.SetStr("OK", object.NewInt(0))

	// --- Key constants ---
	m.Dict.SetStr("KEY_MIN", object.NewInt(257))
	m.Dict.SetStr("KEY_BREAK", object.NewInt(257))
	m.Dict.SetStr("KEY_DOWN", object.NewInt(258))
	m.Dict.SetStr("KEY_UP", object.NewInt(259))
	m.Dict.SetStr("KEY_LEFT", object.NewInt(260))
	m.Dict.SetStr("KEY_RIGHT", object.NewInt(261))
	m.Dict.SetStr("KEY_HOME", object.NewInt(262))
	m.Dict.SetStr("KEY_BACKSPACE", object.NewInt(263))
	m.Dict.SetStr("KEY_F0", object.NewInt(264))
	for f := 1; f <= 12; f++ {
		m.Dict.SetStr(fmt.Sprintf("KEY_F%d", f), object.NewInt(int64(264+f)))
	}
	m.Dict.SetStr("KEY_DL", object.NewInt(328))
	m.Dict.SetStr("KEY_IL", object.NewInt(329))
	m.Dict.SetStr("KEY_DC", object.NewInt(330))
	m.Dict.SetStr("KEY_IC", object.NewInt(331))
	m.Dict.SetStr("KEY_EIC", object.NewInt(332))
	m.Dict.SetStr("KEY_CLEAR", object.NewInt(333))
	m.Dict.SetStr("KEY_EOS", object.NewInt(334))
	m.Dict.SetStr("KEY_EOL", object.NewInt(335))
	m.Dict.SetStr("KEY_SF", object.NewInt(336))
	m.Dict.SetStr("KEY_SR", object.NewInt(337))
	m.Dict.SetStr("KEY_NPAGE", object.NewInt(338))
	m.Dict.SetStr("KEY_PPAGE", object.NewInt(339))
	m.Dict.SetStr("KEY_STAB", object.NewInt(340))
	m.Dict.SetStr("KEY_CTAB", object.NewInt(341))
	m.Dict.SetStr("KEY_CATAB", object.NewInt(342))
	m.Dict.SetStr("KEY_ENTER", object.NewInt(343))
	m.Dict.SetStr("KEY_SRESET", object.NewInt(344))
	m.Dict.SetStr("KEY_RESET", object.NewInt(345))
	m.Dict.SetStr("KEY_PRINT", object.NewInt(346))
	m.Dict.SetStr("KEY_LL", object.NewInt(347))
	m.Dict.SetStr("KEY_A1", object.NewInt(348))
	m.Dict.SetStr("KEY_A3", object.NewInt(349))
	m.Dict.SetStr("KEY_B2", object.NewInt(350))
	m.Dict.SetStr("KEY_C1", object.NewInt(351))
	m.Dict.SetStr("KEY_C3", object.NewInt(352))
	m.Dict.SetStr("KEY_BTAB", object.NewInt(353))
	m.Dict.SetStr("KEY_BEG", object.NewInt(354))
	m.Dict.SetStr("KEY_CANCEL", object.NewInt(355))
	m.Dict.SetStr("KEY_CLOSE", object.NewInt(356))
	m.Dict.SetStr("KEY_COMMAND", object.NewInt(357))
	m.Dict.SetStr("KEY_COPY", object.NewInt(358))
	m.Dict.SetStr("KEY_CREATE", object.NewInt(359))
	m.Dict.SetStr("KEY_END", object.NewInt(360))
	m.Dict.SetStr("KEY_EXIT", object.NewInt(361))
	m.Dict.SetStr("KEY_FIND", object.NewInt(362))
	m.Dict.SetStr("KEY_HELP", object.NewInt(363))
	m.Dict.SetStr("KEY_MARK", object.NewInt(364))
	m.Dict.SetStr("KEY_MESSAGE", object.NewInt(365))
	m.Dict.SetStr("KEY_MOUSE", object.NewInt(409))
	m.Dict.SetStr("KEY_MOVE", object.NewInt(366))
	m.Dict.SetStr("KEY_NEXT", object.NewInt(367))
	m.Dict.SetStr("KEY_OPEN", object.NewInt(368))
	m.Dict.SetStr("KEY_OPTIONS", object.NewInt(369))
	m.Dict.SetStr("KEY_PREVIOUS", object.NewInt(370))
	m.Dict.SetStr("KEY_REDO", object.NewInt(371))
	m.Dict.SetStr("KEY_REFERENCE", object.NewInt(372))
	m.Dict.SetStr("KEY_REFRESH", object.NewInt(373))
	m.Dict.SetStr("KEY_REPLACE", object.NewInt(374))
	m.Dict.SetStr("KEY_RESIZE", object.NewInt(410))
	m.Dict.SetStr("KEY_RESTART", object.NewInt(375))
	m.Dict.SetStr("KEY_RESUME", object.NewInt(376))
	m.Dict.SetStr("KEY_SAVE", object.NewInt(377))
	m.Dict.SetStr("KEY_SBEG", object.NewInt(378))
	m.Dict.SetStr("KEY_SCANCEL", object.NewInt(379))
	m.Dict.SetStr("KEY_SCOMMAND", object.NewInt(380))
	m.Dict.SetStr("KEY_SCOPY", object.NewInt(381))
	m.Dict.SetStr("KEY_SCREATE", object.NewInt(382))
	m.Dict.SetStr("KEY_SDC", object.NewInt(383))
	m.Dict.SetStr("KEY_SDL", object.NewInt(384))
	m.Dict.SetStr("KEY_SELECT", object.NewInt(385))
	m.Dict.SetStr("KEY_SEND", object.NewInt(386))
	m.Dict.SetStr("KEY_SEOL", object.NewInt(387))
	m.Dict.SetStr("KEY_SEXIT", object.NewInt(388))
	m.Dict.SetStr("KEY_SFIND", object.NewInt(389))
	m.Dict.SetStr("KEY_SHELP", object.NewInt(390))
	m.Dict.SetStr("KEY_SHOME", object.NewInt(391))
	m.Dict.SetStr("KEY_SIC", object.NewInt(392))
	m.Dict.SetStr("KEY_SLEFT", object.NewInt(393))
	m.Dict.SetStr("KEY_SMESSAGE", object.NewInt(394))
	m.Dict.SetStr("KEY_SMOVE", object.NewInt(395))
	m.Dict.SetStr("KEY_SNEXT", object.NewInt(396))
	m.Dict.SetStr("KEY_SOPTIONS", object.NewInt(397))
	m.Dict.SetStr("KEY_SPREVIOUS", object.NewInt(398))
	m.Dict.SetStr("KEY_SPRINT", object.NewInt(399))
	m.Dict.SetStr("KEY_SREDO", object.NewInt(400))
	m.Dict.SetStr("KEY_SREPLACE", object.NewInt(401))
	m.Dict.SetStr("KEY_SRIGHT", object.NewInt(402))
	m.Dict.SetStr("KEY_SRSUME", object.NewInt(403))
	m.Dict.SetStr("KEY_SSAVE", object.NewInt(404))
	m.Dict.SetStr("KEY_SSUSPEND", object.NewInt(405))
	m.Dict.SetStr("KEY_SUNDO", object.NewInt(406))
	m.Dict.SetStr("KEY_SUSPEND", object.NewInt(407))
	m.Dict.SetStr("KEY_UNDO", object.NewInt(408))
	m.Dict.SetStr("KEY_MAX", object.NewInt(511))

	// --- ACS (alternate character set) line-drawing constants ---
	// ASCII fallbacks since we are a stub without a real terminal
	m.Dict.SetStr("ACS_ULCORNER", object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_LLCORNER", object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_URCORNER", object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_LRCORNER", object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_LTEE",     object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_RTEE",     object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_BTEE",     object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_TTEE",     object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_HLINE",    object.NewInt(int64('-')))
	m.Dict.SetStr("ACS_VLINE",    object.NewInt(int64('|')))
	m.Dict.SetStr("ACS_PLUS",     object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_S1",       object.NewInt(int64('-')))
	m.Dict.SetStr("ACS_S9",       object.NewInt(int64('_')))
	m.Dict.SetStr("ACS_DIAMOND",  object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_CKBOARD",  object.NewInt(int64(':')))
	m.Dict.SetStr("ACS_DEGREE",   object.NewInt(int64('\'')))
	m.Dict.SetStr("ACS_PLMINUS",  object.NewInt(int64('+')))
	m.Dict.SetStr("ACS_BULLET",   object.NewInt(int64('.')))
	m.Dict.SetStr("ACS_LARROW",   object.NewInt(int64('<')))
	m.Dict.SetStr("ACS_RARROW",   object.NewInt(int64('>')))
	m.Dict.SetStr("ACS_DARROW",   object.NewInt(int64('v')))
	m.Dict.SetStr("ACS_UARROW",   object.NewInt(int64('^')))
	m.Dict.SetStr("ACS_BOARD",    object.NewInt(int64('#')))
	m.Dict.SetStr("ACS_LANTERN",  object.NewInt(int64('*')))
	m.Dict.SetStr("ACS_BLOCK",    object.NewInt(int64('#')))

	// --- Module-level functions ---

	// initscr() → window
	m.Dict.SetStr("initscr", &object.BuiltinFunc{Name: "initscr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeCursesWindow(windowCls, 24, 80, 0, 0), nil
	}})

	// endwin() → None
	m.Dict.SetStr("endwin", &object.BuiltinFunc{Name: "endwin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// newwin(nlines, ncols, begin_y=0, begin_x=0) → window
	m.Dict.SetStr("newwin", &object.BuiltinFunc{Name: "newwin", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		nl, nc, by, bx := 24, 80, 0, 0
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				nl = int(n)
			}
		}
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				nc = int(n)
			}
		}
		if len(a) >= 3 {
			if n, ok := toInt64(a[2]); ok {
				by = int(n)
			}
		}
		if len(a) >= 4 {
			if n, ok := toInt64(a[3]); ok {
				bx = int(n)
			}
		}
		return i.makeCursesWindow(windowCls, nl, nc, by, bx), nil
	}})

	// newpad(nlines, ncols) → window
	m.Dict.SetStr("newpad", &object.BuiltinFunc{Name: "newpad", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		nl, nc := 24, 80
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				nl = int(n)
			}
		}
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				nc = int(n)
			}
		}
		return i.makeCursesWindow(windowCls, nl, nc, 0, 0), nil
	}})

	// wrapper(func, *args, **kwargs) → calls func(stdscr, ...)
	m.Dict.SetStr("wrapper", &object.BuiltinFunc{Name: "wrapper", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "wrapper() requires a callable")
		}
		fn := a[0]
		stdscr := i.makeCursesWindow(windowCls, 24, 80, 0, 0)
		callArgs := make([]object.Object, 0, 1+len(a)-1)
		callArgs = append(callArgs, stdscr)
		callArgs = append(callArgs, a[1:]...)
		return i.callObject(fn, callArgs, kw)
	}})

	// start_color() → None
	m.Dict.SetStr("start_color", &object.BuiltinFunc{Name: "start_color", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// use_default_colors() → None
	m.Dict.SetStr("use_default_colors", &object.BuiltinFunc{Name: "use_default_colors", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// init_pair(pair_number, fg, bg) → None
	m.Dict.SetStr("init_pair", &object.BuiltinFunc{Name: "init_pair", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// init_color(color_number, r, g, b) → None
	m.Dict.SetStr("init_color", &object.BuiltinFunc{Name: "init_color", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// color_pair(pair_number) → int
	m.Dict.SetStr("color_pair", &object.BuiltinFunc{Name: "color_pair", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				return object.NewInt(n << 8), nil
			}
		}
		return object.NewInt(0), nil
	}})

	// color_content(color_number) → (0, 0, 0)
	m.Dict.SetStr("color_content", &object.BuiltinFunc{Name: "color_content", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0), object.NewInt(0)}}, nil
	}})

	// pair_content(pair_number) → (-1, -1)
	m.Dict.SetStr("pair_content", &object.BuiltinFunc{Name: "pair_content", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(-1), object.NewInt(-1)}}, nil
	}})

	// pair_number(attr) → attr >> 8
	m.Dict.SetStr("pair_number", &object.BuiltinFunc{Name: "pair_number", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				return object.NewInt(n >> 8), nil
			}
		}
		return object.NewInt(0), nil
	}})

	// has_colors() → False
	m.Dict.SetStr("has_colors", &object.BuiltinFunc{Name: "has_colors", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// can_change_color() → False
	m.Dict.SetStr("can_change_color", &object.BuiltinFunc{Name: "can_change_color", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// has_extended_color_support() → False
	m.Dict.SetStr("has_extended_color_support", &object.BuiltinFunc{Name: "has_extended_color_support", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// has_key(ch) → False
	m.Dict.SetStr("has_key", &object.BuiltinFunc{Name: "has_key", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// isendwin() → True (stub always ended)
	m.Dict.SetStr("isendwin", &object.BuiltinFunc{Name: "isendwin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.True, nil
	}})

	// cbreak / nocbreak / echo / noecho / nl / nonl / raw / noraw → None
	for _, name := range []string{"cbreak", "nocbreak", "echo", "noecho", "nl", "nonl", "raw", "noraw"} {
		n := name
		m.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}

	// curs_set(visibility) → 0
	m.Dict.SetStr("curs_set", &object.BuiltinFunc{Name: "curs_set", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})

	// halfdelay(tenths) → None
	m.Dict.SetStr("halfdelay", &object.BuiltinFunc{Name: "halfdelay", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// intrflush(flag) → None
	m.Dict.SetStr("intrflush", &object.BuiltinFunc{Name: "intrflush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// meta(flag) → None
	m.Dict.SetStr("meta", &object.BuiltinFunc{Name: "meta", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// mouseinterval(interval) → 0
	m.Dict.SetStr("mouseinterval", &object.BuiltinFunc{Name: "mouseinterval", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})

	// mousemask(newmask) → (0, 0)
	m.Dict.SetStr("mousemask", &object.BuiltinFunc{Name: "mousemask", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
	}})

	// getmouse() → (0, 0, 0, 0, 0)
	m.Dict.SetStr("getmouse", &object.BuiltinFunc{Name: "getmouse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0),
		}}, nil
	}})

	// ungetmouse(id, x, y, z, bstate) → None
	m.Dict.SetStr("ungetmouse", &object.BuiltinFunc{Name: "ungetmouse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// napms(ms) → None
	m.Dict.SetStr("napms", &object.BuiltinFunc{Name: "napms", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// doupdate() → None
	m.Dict.SetStr("doupdate", &object.BuiltinFunc{Name: "doupdate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// flash() → None
	m.Dict.SetStr("flash", &object.BuiltinFunc{Name: "flash", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// beep() → None
	m.Dict.SetStr("beep", &object.BuiltinFunc{Name: "beep", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// erasechar() → b'\x7f'
	m.Dict.SetStr("erasechar", &object.BuiltinFunc{Name: "erasechar", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{0x7f}}, nil
	}})

	// killchar() → b'\x15'
	m.Dict.SetStr("killchar", &object.BuiltinFunc{Name: "killchar", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{0x15}}, nil
	}})

	// longname() → "stub terminal"
	m.Dict.SetStr("longname", &object.BuiltinFunc{Name: "longname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "stub terminal"}, nil
	}})

	// termname() → "stub"
	m.Dict.SetStr("termname", &object.BuiltinFunc{Name: "termname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "stub"}, nil
	}})

	// termattrs() → 0
	m.Dict.SetStr("termattrs", &object.BuiltinFunc{Name: "termattrs", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})

	// tigetflag(capname) → -1
	m.Dict.SetStr("tigetflag", &object.BuiltinFunc{Name: "tigetflag", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(-1), nil
	}})

	// tigetnum(capname) → -2
	m.Dict.SetStr("tigetnum", &object.BuiltinFunc{Name: "tigetnum", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(-2), nil
	}})

	// tigetstr(capname) → None
	m.Dict.SetStr("tigetstr", &object.BuiltinFunc{Name: "tigetstr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// tparm(str, *args) → str
	m.Dict.SetStr("tparm", &object.BuiltinFunc{Name: "tparm", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			return a[0], nil
		}
		return &object.Bytes{V: []byte{}}, nil
	}})

	// typeahead(fd) → None
	m.Dict.SetStr("typeahead", &object.BuiltinFunc{Name: "typeahead", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// unctrl(ch) → str
	m.Dict.SetStr("unctrl", &object.BuiltinFunc{Name: "unctrl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				if n < 32 {
					return &object.Str{V: fmt.Sprintf("^%c", rune('A'+n-1))}, nil
				}
				return &object.Str{V: string(rune(n))}, nil
			}
		}
		return &object.Str{V: ""}, nil
	}})

	// ungetch(ch) → None
	m.Dict.SetStr("ungetch", &object.BuiltinFunc{Name: "ungetch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// unget_wch(ch) → None
	m.Dict.SetStr("unget_wch", &object.BuiltinFunc{Name: "unget_wch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// use_env(flag) → None
	m.Dict.SetStr("use_env", &object.BuiltinFunc{Name: "use_env", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// filter() → None
	m.Dict.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// reset_prog_mode / reset_shell_mode / resetty / savetty → None
	for _, name := range []string{"reset_prog_mode", "reset_shell_mode", "resetty", "savetty"} {
		n := name
		m.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}

	// setupterm(term=None, fd=-1) → None
	m.Dict.SetStr("setupterm", &object.BuiltinFunc{Name: "setupterm", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// get_tabsize() → 8
	m.Dict.SetStr("get_tabsize", &object.BuiltinFunc{Name: "get_tabsize", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(8), nil
	}})

	// set_tabsize(size) → None
	m.Dict.SetStr("set_tabsize", &object.BuiltinFunc{Name: "set_tabsize", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// resize_term / resizeterm / update_lines_cols → None
	for _, name := range []string{"resize_term", "resizeterm", "update_lines_cols"} {
		n := name
		m.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}

	// getwin(file) → window stub
	m.Dict.SetStr("getwin", &object.BuiltinFunc{Name: "getwin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeCursesWindow(windowCls, 24, 80, 0, 0), nil
	}})

	// setsyx(y, x) → None
	m.Dict.SetStr("setsyx", &object.BuiltinFunc{Name: "setsyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// getsyx() → (0, 0)
	m.Dict.SetStr("getsyx", &object.BuiltinFunc{Name: "getsyx", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
	}})

	// flushinp() → None
	m.Dict.SetStr("flushinp", &object.BuiltinFunc{Name: "flushinp", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return m
}
