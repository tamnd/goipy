"""Tests for curses.textpad module."""
import curses
import curses.textpad

# --- rectangle exists and is callable ---
print(callable(curses.textpad.rectangle))          # True

# --- rectangle runs without error on a stub window ---
win = curses.newwin(10, 40, 0, 0)
curses.textpad.rectangle(win, 1, 1, 8, 38)
print(True)   # no exception raised              # True

# --- Textbox class exists ---
print(hasattr(curses.textpad, 'Textbox'))          # True

# --- Textbox can be constructed ---
editwin = curses.newwin(5, 30, 0, 0)
tb = curses.textpad.Textbox(editwin)
print(tb is not None)                             # True

# --- Textbox with insert_mode ---
tb2 = curses.textpad.Textbox(editwin, insert_mode=True)
print(tb2 is not None)                            # True

# --- gather() returns a string ---
result = tb.gather()
print(isinstance(result, str))                    # True

# --- edit() returns a string ---
edited = tb.edit()
print(isinstance(edited, str))                    # True

# --- do_command with printable char ---
tb3 = curses.textpad.Textbox(curses.newwin(5, 30, 0, 0))
tb3.do_command(ord('H'))
tb3.do_command(ord('i'))
content = tb3.gather()
print(isinstance(content, str))                   # True
print(content == 'Hi')                            # True

# --- do_command with ctrl-A (go to BOL) ---
tb4 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb4.do_command(ord('A'))
tb4.do_command(ord('B'))
tb4.do_command(1)   # Ctrl-A -> go to beginning of line
tb4.do_command(ord('X'))
content4 = tb4.gather()
print(isinstance(content4, str))                  # True
print(len(content4) >= 3)                         # True (X, A, B all inserted)

# --- do_command with ctrl-H (backspace) ---
tb5 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb5.do_command(ord('A'))
tb5.do_command(ord('B'))
tb5.do_command(ord('C'))
tb5.do_command(8)   # Ctrl-H = backspace
content5 = tb5.gather()
print(content5 == 'AB')                           # True

# --- do_command with ctrl-K (kill to end of line) ---
tb6 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb6.do_command(ord('H'))
tb6.do_command(ord('e'))
tb6.do_command(ord('l'))
tb6.do_command(ord('l'))
tb6.do_command(ord('o'))
tb6.do_command(1)   # Ctrl-A -> go to BOL
tb6.do_command(11)  # Ctrl-K -> kill to EOL
content6 = tb6.gather()
print(content6 == '')                             # True

# --- do_command with ctrl-E (go to EOL) ---
tb7 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb7.do_command(ord('A'))
tb7.do_command(ord('B'))
tb7.do_command(1)   # Ctrl-A -> BOL
tb7.do_command(5)   # Ctrl-E -> EOL
tb7.do_command(ord('C'))   # insert at end
content7 = tb7.gather()
print(content7 == 'ABC')                          # True

# --- do_command ctrl-B / ctrl-F (cursor movement) ---
tb8 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb8.do_command(ord('X'))
tb8.do_command(ord('Y'))
tb8.do_command(2)   # Ctrl-B -> back one char
tb8.do_command(ord('Z'))   # insert between X and Y
content8 = tb8.gather()
print(content8 == 'XZY')                          # True

# --- do_command ctrl-D (delete char at cursor) ---
tb9 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
tb9.do_command(ord('A'))
tb9.do_command(ord('B'))
tb9.do_command(ord('C'))
tb9.do_command(1)   # Ctrl-A -> BOL
tb9.do_command(4)   # Ctrl-D -> delete 'A'
content9 = tb9.gather()
print(content9 == 'BC')                           # True

# --- edit() with a validate function ---
def stopper(ch):
    if ch == ord('!'):
        return 7   # Ctrl-G = terminate
    return ch

tb10 = curses.textpad.Textbox(curses.newwin(3, 20, 0, 0))
# Without actual input stream, edit() just returns current content
result10 = tb10.edit(validate=stopper)
print(isinstance(result10, str))                  # True

# --- ACS constants in curses module ---
print(isinstance(curses.ACS_HLINE, int))          # True
print(isinstance(curses.ACS_VLINE, int))          # True
print(isinstance(curses.ACS_ULCORNER, int))       # True
print(isinstance(curses.ACS_LRCORNER, int))       # True

# --- rectangle with minimal window (2x2) ---
win2 = curses.newwin(3, 3, 0, 0)
curses.textpad.rectangle(win2, 0, 0, 2, 2)
print(True)   # no exception                     # True
