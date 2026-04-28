import termios

# ── all constants with exact CPython macOS values ──────────────────────────────
_EXPECTED = {
    'ALTWERASE': 512,
    'B0': 0, 'B50': 50, 'B75': 75, 'B110': 110, 'B134': 134,
    'B150': 150, 'B200': 200, 'B300': 300, 'B600': 600, 'B1200': 1200,
    'B1800': 1800, 'B2400': 2400, 'B4800': 4800, 'B7200': 7200, 'B9600': 9600,
    'B14400': 14400, 'B19200': 19200, 'B28800': 28800, 'B38400': 38400,
    'B57600': 57600, 'B76800': 76800, 'B115200': 115200, 'B230400': 230400,
    'BRKINT': 2,
    'BS0': 0, 'BS1': 32768, 'BSDLY': 32768,
    'CCAR_OFLOW': 1048576,
    'CCTS_OFLOW': 65536,
    'CDSR_OFLOW': 524288,
    'CDSUSP': 25,
    'CDTR_IFLOW': 262144,
    'CEOF': 4, 'CEOL': 255, 'CEOT': 4, 'CERASE': 127,
    'CFLUSH': 15, 'CIGNORE': 1, 'CINTR': 3, 'CKILL': 21,
    'CLNEXT': 22, 'CLOCAL': 32768, 'CQUIT': 28,
    'CR0': 0, 'CR1': 4096, 'CR2': 8192, 'CR3': 12288, 'CRDLY': 12288,
    'CREAD': 2048,
    'CRPRNT': 18,
    'CRTSCTS': 196608, 'CRTS_IFLOW': 131072,
    'CS5': 0, 'CS6': 256, 'CS7': 512, 'CS8': 768, 'CSIZE': 768,
    'CSTART': 17, 'CSTOP': 19, 'CSTOPB': 1024,
    'CSUSP': 26, 'CWERASE': 23,
    'ECHO': 8, 'ECHOCTL': 64, 'ECHOE': 2, 'ECHOK': 4, 'ECHOKE': 1,
    'ECHONL': 16, 'ECHOPRT': 32,
    'EXTA': 19200, 'EXTB': 38400, 'EXTPROC': 2048,
    'FF0': 0, 'FF1': 16384, 'FFDLY': 16384,
    'FIOASYNC': 2147772029,
    'FIOCLEX': 536897025,
    'FIONBIO': 2147772030,
    'FIONCLEX': 536897026,
    'FIONREAD': 1074030207,
    'FLUSHO': 8388608,
    'HUPCL': 16384,
    'ICANON': 256, 'ICRNL': 256, 'IEXTEN': 1024,
    'IGNBRK': 1, 'IGNCR': 128, 'IGNPAR': 4,
    'IMAXBEL': 8192, 'INLCR': 64, 'INPCK': 16,
    'ISIG': 128, 'ISTRIP': 32, 'IUTF8': 16384,
    'IXANY': 2048, 'IXOFF': 1024, 'IXON': 512,
    'MDMBUF': 1048576,
    'NCCS': 20,
    'NL0': 0, 'NL1': 256, 'NL2': 512, 'NL3': 768, 'NLDLY': 768,
    'NOFLSH': 2147483648, 'NOKERNINFO': 33554432,
    'OCRNL': 16, 'OFDEL': 131072, 'OFILL': 128,
    'ONLCR': 2, 'ONLRET': 64, 'ONOCR': 32, 'ONOEOT': 8, 'OPOST': 1, 'OXTABS': 4,
    'PARENB': 4096, 'PARMRK': 8, 'PARODD': 8192,
    'PENDIN': 536870912,
    'TAB0': 0, 'TAB1': 1024, 'TAB2': 2048, 'TAB3': 4, 'TABDLY': 3076,
    'TCIFLUSH': 1, 'TCIOFF': 3, 'TCIOFLUSH': 3, 'TCION': 4,
    'TCOFLUSH': 2, 'TCOOFF': 1, 'TCOON': 2,
    'TCSADRAIN': 1, 'TCSAFLUSH': 2, 'TCSANOW': 0, 'TCSASOFT': 16,
    'TIOCCONS': 2147775586,
    'TIOCEXCL': 536900621,
    'TIOCGETD': 1074033690,
    'TIOCGPGRP': 1074033783,
    'TIOCGSIZE': 1074295912, 'TIOCGWINSZ': 1074295912,
    'TIOCMBIC': 2147775595, 'TIOCMBIS': 2147775596,
    'TIOCMGET': 1074033770, 'TIOCMSET': 2147775597,
    'TIOCM_CAR': 64, 'TIOCM_CD': 64, 'TIOCM_CTS': 32,
    'TIOCM_DSR': 256, 'TIOCM_DTR': 2, 'TIOCM_LE': 1,
    'TIOCM_RI': 128, 'TIOCM_RNG': 128, 'TIOCM_RTS': 4,
    'TIOCM_SR': 16, 'TIOCM_ST': 8,
    'TIOCNOTTY': 536900721, 'TIOCNXCL': 536900622,
    'TIOCOUTQ': 1074033779,
    'TIOCPKT': 2147775600, 'TIOCPKT_DATA': 0, 'TIOCPKT_DOSTOP': 32,
    'TIOCPKT_FLUSHREAD': 1, 'TIOCPKT_FLUSHWRITE': 2,
    'TIOCPKT_NOSTOP': 16, 'TIOCPKT_START': 8, 'TIOCPKT_STOP': 4,
    'TIOCSCTTY': 536900705,
    'TIOCSETD': 2147775515, 'TIOCSPGRP': 2147775606,
    'TIOCSSIZE': 2148037735, 'TIOCSTI': 2147578994, 'TIOCSWINSZ': 2148037735,
    'TOSTOP': 4194304,
    'VDISCARD': 15, 'VDSUSP': 11, 'VEOF': 0, 'VEOL': 1, 'VEOL2': 2,
    'VERASE': 3, 'VINTR': 8, 'VKILL': 5, 'VLNEXT': 14, 'VMIN': 16,
    'VQUIT': 9, 'VREPRINT': 6, 'VSTART': 12, 'VSTATUS': 18,
    'VSTOP': 13, 'VSUSP': 10,
    'VT0': 0, 'VT1': 65536, 'VTDLY': 65536,
    'VTIME': 17, 'VWERASE': 4,
}

for name, expected in _EXPECTED.items():
    actual = getattr(termios, name)
    assert actual == expected, f'{name}: expected {expected}, got {actual}'

# ── non-CPython constants must NOT be present ──────────────────────────────────
for fake in ('IFLAG', 'OFLAG', 'CFLAG', 'LFLAG', 'ISPEED', 'OSPEED', 'CC'):
    assert not hasattr(termios, fake), f'unexpected attribute: {fake}'

# ── error is an exception class ────────────────────────────────────────────────
assert issubclass(termios.error, Exception)

# ── all functions callable ─────────────────────────────────────────────────────
assert callable(termios.tcgetattr)
assert callable(termios.tcsetattr)
assert callable(termios.tcdrain)
assert callable(termios.tcflush)
assert callable(termios.tcflow)
assert callable(termios.tcsendbreak)
assert callable(termios.tcgetwinsize)
assert callable(termios.tcsetwinsize)

# ── tty-dependent ops -- raise termios.error on non-tty fd ────────────────────
try:
    termios.tcgetattr(0)
except termios.error:
    pass

try:
    termios.tcdrain(1)
except termios.error:
    pass

try:
    termios.tcflush(0, termios.TCIOFLUSH)
except termios.error:
    pass

try:
    termios.tcflow(0, termios.TCOON)
except termios.error:
    pass

try:
    termios.tcsendbreak(0, 0)
except termios.error:
    pass

try:
    termios.tcgetwinsize(1)
except termios.error:
    pass

print('ok')
