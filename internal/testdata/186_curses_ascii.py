"""Tests for curses.ascii module."""
import curses.ascii as a

# --- Constants ---
print(a.NUL == 0)     # True
print(a.SOH == 1)     # True
print(a.BEL == 7)     # True
print(a.BS  == 8)     # True
print(a.TAB == 9)     # True
print(a.HT  == 9)     # True  (alias)
print(a.LF  == 10)    # True
print(a.NL  == 10)    # True  (alias)
print(a.CR  == 13)    # True
print(a.ESC == 27)    # True
print(a.SP  == 32)    # True
print(a.DEL == 127)   # True
print(a.EOF == -1)    # True

# --- isascii ---
print(a.isascii(0))         # True
print(a.isascii(127))       # True
print(a.isascii(128))       # False
print(a.isascii('A'))       # True
print(a.isascii(b'Z'))      # True

# --- isdigit ---
print(a.isdigit('0'))       # True
print(a.isdigit('9'))       # True
print(a.isdigit('a'))       # False
print(a.isdigit(48))        # True   (ord('0'))
print(a.isdigit(57))        # True   (ord('9'))
print(a.isdigit(58))        # False

# --- isalpha ---
print(a.isalpha('a'))       # True
print(a.isalpha('Z'))       # True
print(a.isalpha('1'))       # False
print(a.isalpha(' '))       # False

# --- isupper / islower ---
print(a.isupper('A'))       # True
print(a.isupper('z'))       # False
print(a.islower('a'))       # True
print(a.islower('A'))       # False

# --- isalnum ---
print(a.isalnum('a'))       # True
print(a.isalnum('5'))       # True
print(a.isalnum('!'))       # False

# --- isspace ---
print(a.isspace(' '))       # True
print(a.isspace('\t'))      # True
print(a.isspace('\n'))      # True
print(a.isspace('\r'))      # True
print(a.isspace('x'))       # False

# --- isblank ---
print(a.isblank(' '))       # True
print(a.isblank('\t'))      # True
print(a.isblank('\n'))      # False  (only SP and TAB)

# --- iscntrl ---
print(a.iscntrl(0))         # True
print(a.iscntrl(31))        # True
print(a.iscntrl(127))       # True
print(a.iscntrl(32))        # False
print(a.iscntrl('A'))       # False

# --- isprint ---
print(a.isprint(' '))       # True  (32)
print(a.isprint('~'))       # True  (126)
print(a.isprint(127))       # False (DEL)
print(a.isprint(31))        # False

# --- isgraph ---
print(a.isgraph('!'))       # True  (33)
print(a.isgraph('~'))       # True  (126)
print(a.isgraph(' '))       # False (space excluded)
print(a.isgraph(127))       # False

# --- ispunct ---
print(a.ispunct('!'))       # True
print(a.ispunct('.'))       # True
print(a.ispunct('a'))       # False
print(a.ispunct('1'))       # False

# --- isxdigit ---
print(a.isxdigit('0'))      # True
print(a.isxdigit('9'))      # True
print(a.isxdigit('a'))      # True
print(a.isxdigit('F'))      # True
print(a.isxdigit('g'))      # False
print(a.isxdigit('G'))      # False

# --- ascii (returns int ordinal) ---
print(a.ascii('A') == 65)   # True
print(a.ascii(65)  == 65)   # True
print(a.ascii(b'A') == 65)  # True

# --- toascii ---
print(a.toascii(65)   == 65)    # True
print(a.toascii(193)  == 65)    # True  (193 & 0x7F == 65)
print(a.toascii(128)  == 0)     # True  (128 & 0x7F == 0)
print(a.toascii('A')  == 65)    # True

# --- ctrl ---
print(a.ctrl('A') == 1)     # True  (65 & 0x1F == 1)
print(a.ctrl('@') == 0)     # True  (64 & 0x1F == 0)
print(a.ctrl(ord('J')) == 10)  # True  (LF)

# --- alt ---
print(a.alt(65) == 193)     # True  (65 | 0x80)
print(a.alt(0)  == 128)     # True  (0  | 0x80)

# --- unctrl ---
print(a.unctrl('A') == 'A')      # True  (printable)
print(a.unctrl(' ') == ' ')      # True  (printable)
print(a.unctrl(1)   == '^A')     # True  (ctrl-A)
print(a.unctrl(27)  == '^[')     # True  (ESC = 27, 27+64=91='[')
print(a.unctrl(127) == '^?')     # True  (DEL)
