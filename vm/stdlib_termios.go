package vm

import (
	"github.com/tamnd/goipy/object"
)

// termiosConstants returns the full map of termios constant name -> value.
// Values are from macOS / BSD; Linux uses different values for some.
func termiosConstants() map[string]int64 {
	return map[string]int64{
		// action constants for tcsetattr
		"TCSANOW": 0, "TCSADRAIN": 1, "TCSAFLUSH": 2, "TCSASOFT": 0x10,
		// flush constants for tcflush
		"TCIFLUSH": 1, "TCOFLUSH": 2, "TCIOFLUSH": 3,
		// flow control for tcflow
		"TCOOFF": 1, "TCOON": 2, "TCIOFF": 3, "TCION": 4,
		// c_cc array indices
		"VEOF": 0, "VEOL": 1, "VEOL2": 2, "VERASE": 3, "VWERASE": 4,
		"VKILL": 5, "VREPRINT": 6, "VINTR": 8, "VQUIT": 9, "VSUSP": 10,
		"VDSUSP": 11, "VSTART": 12, "VSTOP": 13, "VLNEXT": 14, "VDISCARD": 15,
		"VMIN": 16, "VTIME": 17, "VSTATUS": 18, "NCCS": 20, "CEOF": 4,
		"CEOL": 0xff, "CEOT": 4, "CERASE": 0x7f, "CFLUSH": 0xf,
		"CINTR": 0x3, "CKILL": 0x15, "CLNEXT": 0x16, "CQUIT": 0x1c,
		"CSTART": 0x11, "CSTOP": 0x13, "CSUSP": 0x1a, "CWERASE": 0x17,
		"CDSUSP": 0x19, "CDISCARD": 0xf,
		// c_iflag bits
		"IGNBRK": 0x1, "BRKINT": 0x2, "IGNPAR": 0x4, "PARMRK": 0x8,
		"INPCK": 0x10, "ISTRIP": 0x20, "INLCR": 0x40, "IGNCR": 0x80,
		"ICRNL": 0x100, "IXON": 0x200, "IXOFF": 0x400, "IXANY": 0x800,
		"IMAXBEL": 0x2000, "IUTF8": 0x4000,
		// c_oflag bits
		"OPOST": 0x1, "ONLCR": 0x2, "OXTABS": 0x4, "ONOEOT": 0x8,
		"OCRNL": 0x10, "ONOCR": 0x20, "ONLRET": 0x40, "OFILL": 0x80,
		"OFDEL": 0x20000, "NLDLY": 0x300, "NL0": 0, "NL1": 0x100, "NL2": 0x200, "NL3": 0x300,
		"TABDLY": 0x4, "TAB0": 0, "TAB1": 0x400, "TAB2": 0x800, "TAB3": 0x4,
		"CRDLY": 0x3000, "CR0": 0, "CR1": 0x1000, "CR2": 0x2000, "CR3": 0x3000,
		"FFDLY": 0x4000, "FF0": 0, "FF1": 0x4000,
		"BSDLY": 0x8000, "BS0": 0, "BS1": 0x8000,
		"VTDLY": 0x10000, "VT0": 0, "VT1": 0x10000,
		// c_cflag bits
		"CSIZE": 0x300, "CS5": 0, "CS6": 0x100, "CS7": 0x200, "CS8": 0x300,
		"CSTOPB": 0x400, "CREAD": 0x800, "PARENB": 0x1000, "PARODD": 0x2000,
		"HUPCL": 0x4000, "CLOCAL": 0x8000, "CIGNORE": 0x1,
		"CRTSCTS": 0x30000, "CRTS_IFLOW": 0x10000, "CCTS_OFLOW": 0x20000,
		"CDTR_IFLOW": 0x40000, "CDSR_OFLOW": 0x80000, "CCAR_OFLOW": 0x100000,
		"MDMBUF": 0x100000,
		// c_lflag bits
		"ECHOKE": 0x1, "ECHOE": 0x2, "ECHOK": 0x4, "ECHO": 0x8,
		"ECHONL": 0x10, "ECHOPRT": 0x20, "ECHOCTL": 0x40,
		"ISIG": 0x80, "ICANON": 0x100, "ALTWERASE": 0x200,
		"IEXTEN": 0x400, "EXTPROC": 0x800,
		"TOSTOP": 0x400000, "FLUSHO": 0x800000, "NOKERNINFO": 0x2000000,
		"PENDIN": 0x20000000, "NOFLSH": 0x80000000,
		// baud rates
		"B0": 0, "B50": 50, "B75": 75, "B110": 110, "B134": 134,
		"B150": 150, "B200": 200, "B300": 300, "B600": 600, "B1200": 1200,
		"B1800": 1800, "B2400": 2400, "B4800": 4800, "B9600": 9600,
		"B14400": 14400, "B19200": 19200, "B28800": 28800, "B38400": 38400,
		"B57600": 57600, "B76800": 76800, "B115200": 115200,
		"B230400": 230400, "B7200": 7200,
		"EXTA": 19200, "EXTB": 38400,
		// struct indices for tcgetattr return
		"IFLAG": 0, "OFLAG": 1, "CFLAG": 2, "LFLAG": 3, "ISPEED": 4, "OSPEED": 5, "CC": 6,
		// ioctl constants
		"TIOCGPGRP": 0x40047477, "TIOCSPGRP": 0x80047476,
		"TIOCGWINSZ": 0x40087468, "TIOCSWINSZ": 0x80087467,
		"TIOCGSIZE": 0x40087468, "TIOCSSIZE": 0x80087467,
		"TIOCEXCL": 0x2000740d, "TIOCNXCL": 0x2000740e,
		"TIOCNOTTY": 0x20007471, "TIOCSCTTY": 0x20007461,
		"TIOCSTI": 0x80017472, "TIOCOUTQ": 0x40047473,
		"TIOCSETD": 0x8004741b, "TIOCGETD": 0x4004741a,
		"TIOCCONS": 0x80047462,
		"TIOCPKT": 0x80047470, "TIOCPKT_DATA": 0, "TIOCPKT_FLUSHREAD": 1,
		"TIOCPKT_FLUSHWRITE": 2, "TIOCPKT_STOP": 4, "TIOCPKT_START": 8,
		"TIOCPKT_NOSTOP": 16, "TIOCPKT_DOSTOP": 32,
		"TIOCMBIS": 0x8004746c, "TIOCMBIC": 0x8004746b, "TIOCMSET": 0x8004746d,
		"TIOCMGET": 0x4004746a,
		"TIOCM_LE": 0x001, "TIOCM_DTR": 0x002, "TIOCM_RTS": 0x004,
		"TIOCM_ST": 0x008, "TIOCM_SR": 0x010, "TIOCM_CTS": 0x020,
		"TIOCM_CAR": 0x040, "TIOCM_CD": 0x040, "TIOCM_RNG": 0x080,
		"TIOCM_RI": 0x080, "TIOCM_DSR": 0x100,
		"FIOASYNC": 0x8004667d, "FIONCLEX": 0x20006602, "FIOCLEX": 0x20006601,
		"FIONBIO": 0x8004667e, "FIONREAD": 0x4004667f,
		"CRPRNT": 0x20,
	}
}

func (i *Interp) buildTermios() *object.Module {
	m := &object.Module{Name: "termios", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("error", i.osErr)

	for name, val := range termiosConstants() {
		d.SetStr(name, intObj(val))
	}

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	// tcgetattr returns a list [iflag, oflag, cflag, lflag, ispeed, ospeed, cc]
	d.SetStr("tcgetattr", &object.BuiltinFunc{
		Name: "tcgetattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cc := make([]object.Object, 20)
			for k := range cc {
				cc[k] = intObj(0)
			}
			return &object.List{V: []object.Object{
				intObj(0), intObj(0), intObj(0), intObj(0),
				intObj(9600), intObj(9600),
				&object.List{V: cc},
			}}, nil
		},
	})
	d.SetStr("tcsetattr", noneStub("tcsetattr"))
	d.SetStr("tcdrain", noneStub("tcdrain"))
	d.SetStr("tcflush", noneStub("tcflush"))
	d.SetStr("tcflow", noneStub("tcflow"))
	d.SetStr("tcsendbreak", noneStub("tcsendbreak"))
	d.SetStr("tcgetwinsize", &object.BuiltinFunc{
		Name: "tcgetwinsize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(24), intObj(80)}}, nil
		},
	})
	d.SetStr("tcsetwinsize", noneStub("tcsetwinsize"))

	return m
}
