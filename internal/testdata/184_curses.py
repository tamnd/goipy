"""Tests for curses module (stub implementation)."""
import curses
import curses.ascii
import curses.textpad
import curses.panel

# --- Constants ---
print(curses.A_NORMAL == 0)         # True
print(curses.A_BOLD > 0)            # True
print(curses.A_UNDERLINE > 0)       # True
print(curses.A_REVERSE > 0)         # True
print(curses.COLOR_BLACK == 0)      # True
print(curses.COLOR_RED == 1)        # True
print(curses.COLOR_WHITE == 7)      # True
print(curses.KEY_UP > 0)            # True
print(curses.KEY_DOWN > 0)          # True
print(curses.ERR == -1)             # True
print(curses.OK == 0)               # True
print(curses.COLS == 80)            # True
print(curses.LINES == 24)           # True

# --- error exception ---
print(issubclass(curses.error, Exception))   # True

# --- Module functions exist and are callable ---
print(callable(curses.initscr))      # True
print(callable(curses.endwin))       # True
print(callable(curses.newwin))       # True
print(callable(curses.wrapper))      # True
print(callable(curses.start_color))  # True
print(callable(curses.has_colors))   # True
print(callable(curses.color_pair))   # True
print(callable(curses.init_pair))    # True
print(callable(curses.isendwin))     # True
print(callable(curses.cbreak))       # True
print(callable(curses.noecho))       # True
print(callable(curses.echo))         # True
print(callable(curses.curs_set))     # True
print(callable(curses.flash))        # True
print(callable(curses.beep))         # True
print(callable(curses.doupdate))     # True
print(callable(curses.napms))        # True
print(callable(curses.mousemask))    # True

# --- newwin returns a window ---
win = curses.newwin(10, 20, 0, 0)
print(win is not None)               # True

# --- window methods ---
print(win.getmaxyx() == (10, 20))    # True
print(win.getbegyx() == (0, 0))      # True
print(win.getyx() == (0, 0))         # True
win.addstr(0, 0, 'hello')
print(True)                          # True
win.move(1, 0)
print(True)                          # True
win.clear()
print(True)                          # True
win.refresh()
print(True)                          # True
print(win.getch() == -1)             # True
print(win.getkey() == '')            # True
win.keypad(True)
print(True)                          # True
win.timeout(100)
print(True)                          # True
print(win.inch() == 0)               # True
win.border()
print(True)                          # True
win.box()
print(True)                          # True
sub = win.subwin(3, 5, 0, 0)
print(sub is not None)               # True
print(sub.getmaxyx() == (3, 5))      # True

# --- color functions ---
curses.start_color()
print(True)                          # True
print(curses.has_colors() == False)  # True
curses.init_pair(1, curses.COLOR_RED, curses.COLOR_BLACK)
print(True)                          # True
print(isinstance(curses.color_pair(1), int))   # True
print(curses.isendwin() == True)     # True

# --- curses.ascii ---
print(curses.ascii.isdigit('5'))     # True
print(not curses.ascii.isdigit('a')) # True
print(curses.ascii.isalpha('a'))     # True
print(not curses.ascii.isalpha('1')) # True
print(curses.ascii.isupper('A'))     # True
print(curses.ascii.islower('a'))     # True
print(curses.ascii.isspace(' '))     # True
print(not curses.ascii.isspace('x')) # True
print(curses.ascii.toascii(65) == 65)    # True
print(curses.ascii.ctrl(ord('A')) == 1)  # True
print(curses.ascii.NUL == 0)         # True
print(curses.ascii.BEL == 7)         # True
print(curses.ascii.SP == 32)         # True
print(curses.ascii.DEL == 127)       # True

# --- curses.textpad ---
print(callable(curses.textpad.rectangle))   # True
tb = curses.textpad.Textbox(win)
print(tb is not None)                # True
print(tb.gather() == '')             # True

# --- curses.panel ---
print(callable(curses.panel.new_panel))      # True
print(callable(curses.panel.update_panels))  # True
pan = curses.panel.new_panel(win)
print(pan is not None)               # True
print(pan.window() is not None)      # True
print(pan.hidden() == False)         # True

# --- wrapper calls function with stdscr ---
def test_func(stdscr):
    return 'ok'
result = curses.wrapper(test_func)
print(result == 'ok')                # True
