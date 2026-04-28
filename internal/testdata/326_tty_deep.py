import tty

# ── struct-index constants (defined in tty.py, not in termios) ─────────────────
assert tty.IFLAG == 0
assert tty.OFLAG == 1
assert tty.CFLAG == 2
assert tty.LFLAG == 3
assert tty.ISPEED == 4
assert tty.OSPEED == 5
assert tty.CC == 6

# ── error is an exception class ────────────────────────────────────────────────
assert issubclass(tty.error, Exception)

# ── key termios constants are re-exported ──────────────────────────────────────
assert tty.TCSANOW == 0
assert tty.TCSADRAIN == 1
assert tty.TCSAFLUSH == 2
assert tty.ECHO == 8
assert tty.ICANON == 256
assert tty.ISIG == 128
assert tty.OPOST == 1
assert tty.CS8 == 768
assert tty.VMIN == 16
assert tty.VTIME == 17
assert tty.B9600 == 9600
assert tty.NCCS == 20

# ── all functions callable ─────────────────────────────────────────────────────
assert callable(tty.cfmakeraw)
assert callable(tty.cfmakecbreak)
assert callable(tty.setraw)
assert callable(tty.setcbreak)
assert callable(tty.tcgetattr)
assert callable(tty.tcsetattr)
assert callable(tty.tcdrain)
assert callable(tty.tcflush)
assert callable(tty.tcflow)
assert callable(tty.tcsendbreak)
assert callable(tty.tcgetwinsize)
assert callable(tty.tcsetwinsize)

# ── cfmakeraw modifies mode in place ──────────────────────────────────────────
mode = [0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 9600, 9600, list(range(20))]
result = tty.cfmakeraw(mode)
assert result is None
# ECHO and ICANON must be cleared
assert (mode[tty.LFLAG] & tty.ECHO) == 0
assert (mode[tty.LFLAG] & tty.ICANON) == 0
# CS8 must be set
assert (mode[tty.CFLAG] & tty.CS8) == tty.CS8
# CC[VMIN] = 1, CC[VTIME] = 0
assert mode[tty.CC][tty.VMIN] == 1
assert mode[tty.CC][tty.VTIME] == 0

# ── cfmakecbreak modifies mode in place ───────────────────────────────────────
mode2 = [0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 9600, 9600, list(range(20))]
result2 = tty.cfmakecbreak(mode2)
assert result2 is None
assert (mode2[tty.LFLAG] & tty.ECHO) == 0
assert (mode2[tty.LFLAG] & tty.ICANON) == 0
assert mode2[tty.CC][tty.VMIN] == 1
assert mode2[tty.CC][tty.VTIME] == 0

# ── cfmakeraw also clears OPOST in OFLAG ──────────────────────────────────────
mode3 = [0xFFFF, 0xFFFF, 0xFFFF, 0xFFFF, 9600, 9600, list(range(20))]
tty.cfmakeraw(mode3)
assert (mode3[tty.OFLAG] & tty.OPOST) == 0

# ── setraw / setcbreak raise on non-tty fd ────────────────────────────────────
try:
    tty.setraw(0)
    raw_raised = False
except tty.error:
    raw_raised = True
assert raw_raised

try:
    tty.setcbreak(0)
    cbreak_raised = False
except tty.error:
    cbreak_raised = True
assert cbreak_raised

# ── tcgetattr raises on non-tty fd ────────────────────────────────────────────
try:
    tty.tcgetattr(0)
except tty.error:
    pass

# ── tcgetwinsize raises on non-tty fd ─────────────────────────────────────────
try:
    tty.tcgetwinsize(1)
except tty.error:
    pass

print('ok')
