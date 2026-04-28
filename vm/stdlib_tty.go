package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTty() *object.Module {
	m := &object.Module{Name: "tty", Dict: object.NewDict()}
	d := m.Dict

	// re-export all termios constants
	for name, val := range termiosConstants() {
		d.SetStr(name, intObj(val))
	}

	// struct-index constants defined in tty.py directly (not in termios)
	d.SetStr("IFLAG", intObj(0))
	d.SetStr("OFLAG", intObj(1))
	d.SetStr("CFLAG", intObj(2))
	d.SetStr("LFLAG", intObj(3))
	d.SetStr("ISPEED", intObj(4))
	d.SetStr("OSPEED", intObj(5))
	d.SetStr("CC", intObj(6))

	d.SetStr("error", i.osErr)

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	// bitClear clears bits in mode[idx] using &^mask
	bitClear := func(mode *object.List, idx int, mask int64) {
		if idx >= len(mode.V) {
			return
		}
		if n, ok := mode.V[idx].(*object.Int); ok {
			mode.V[idx] = intObj(n.V.Int64() &^ mask)
		}
	}
	// bitSet sets bits in mode[idx] using |mask
	bitSet := func(mode *object.List, idx int, mask int64) {
		if idx >= len(mode.V) {
			return
		}
		if n, ok := mode.V[idx].(*object.Int); ok {
			mode.V[idx] = intObj(n.V.Int64() | mask)
		}
	}
	// ccSet sets mode[6][ccIdx] = val
	ccSet := func(mode *object.List, ccIdx, val int) {
		if 6 >= len(mode.V) {
			return
		}
		cc, ok := mode.V[6].(*object.List)
		if !ok {
			return
		}
		if ccIdx >= len(cc.V) {
			return
		}
		cc.V[ccIdx] = intObj(int64(val))
	}

	// cfmakeraw makes termios mode raw (Python 3.12+)
	d.SetStr("cfmakeraw", &object.BuiltinFunc{
		Name: "cfmakeraw",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return object.None, nil
			}
			mode, ok := a[0].(*object.List)
			if !ok || len(mode.V) < 7 {
				return object.None, nil
			}
			// IFLAG: clear IGNBRK|BRKINT|IGNPAR|PARMRK|INPCK|ISTRIP|INLCR|IGNCR|ICRNL|IXON|IXANY|IXOFF
			bitClear(mode, 0, 1|2|4|8|16|32|64|128|256|512|1024|2048)
			// OFLAG: clear OPOST
			bitClear(mode, 1, 1)
			// CFLAG: clear PARENB|CSIZE, then set CS8
			bitClear(mode, 2, 4096|768)
			bitSet(mode, 2, 768)
			// LFLAG: clear ECHO|ECHOE|ECHOK|ECHONL|ICANON|IEXTEN|ISIG|NOFLSH|TOSTOP
			bitClear(mode, 3, 8|2|4|16|256|1024|128|2147483648|4194304)
			// CC[VMIN=16]=1, CC[VTIME=17]=0
			ccSet(mode, 16, 1)
			ccSet(mode, 17, 0)
			return object.None, nil
		},
	})

	// cfmakecbreak makes termios mode cbreak (Python 3.12+)
	d.SetStr("cfmakecbreak", &object.BuiltinFunc{
		Name: "cfmakecbreak",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return object.None, nil
			}
			mode, ok := a[0].(*object.List)
			if !ok || len(mode.V) < 7 {
				return object.None, nil
			}
			// LFLAG: clear ECHO|ICANON
			bitClear(mode, 3, 8|256)
			// CC[VMIN=16]=1, CC[VTIME=17]=0
			ccSet(mode, 16, 1)
			ccSet(mode, 17, 0)
			return object.None, nil
		},
	})

	// setraw and setcbreak raise on non-tty (call tcgetattr which fails)
	d.SetStr("setraw", &object.BuiltinFunc{
		Name: "setraw",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
		},
	})
	d.SetStr("setcbreak", &object.BuiltinFunc{
		Name: "setcbreak",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
		},
	})

	// re-export termios tty functions
	d.SetStr("tcgetattr", &object.BuiltinFunc{
		Name: "tcgetattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
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
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
		},
	})
	d.SetStr("tcsetwinsize", noneStub("tcsetwinsize"))

	return m
}
