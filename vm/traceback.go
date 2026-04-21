package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// FormatException renders a Python-style traceback for an uncaught
// exception. The output mirrors CPython's `traceback.print_exception`:
//
//	Traceback (most recent call last):
//	  File "path.py", line N, in <name>
//	  ...
//	ClassName: message
//
// Cause (`raise X from Y`) and implicit context chains are printed too.
func FormatException(e *object.Exception) string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	writeExceptionChain(&b, e, map[*object.Exception]bool{})
	return b.String()
}

func writeExceptionChain(b *strings.Builder, e *object.Exception, seen map[*object.Exception]bool) {
	if e == nil || seen[e] {
		return
	}
	seen[e] = true
	if e.Cause != nil {
		writeExceptionChain(b, e.Cause, seen)
		b.WriteString("\nThe above exception was the direct cause of the following exception:\n\n")
	} else if e.Ctx != nil {
		writeExceptionChain(b, e.Ctx, seen)
		b.WriteString("\nDuring handling of the above exception, another exception occurred:\n\n")
	}
	writeSingleException(b, e)
}

func writeSingleException(b *strings.Builder, e *object.Exception) {
	// Walk the traceback from outermost to innermost — CPython prints
	// "most recent call last".
	frames := []*object.Traceback{}
	for tb := e.Traceback; tb != nil; tb = tb.Next {
		frames = append(frames, tb)
	}
	if len(frames) > 0 {
		b.WriteString("Traceback (most recent call last):\n")
		for i := len(frames) - 1; i >= 0; i-- {
			tb := frames[i]
			file := "<unknown>"
			name := "<module>"
			line := tb.Lineno
			if tb.Code != nil {
				if tb.Code.Filename != "" {
					file = tb.Code.Filename
				}
				if tb.Code.Name != "" {
					name = tb.Code.Name
				}
			}
			b.WriteString("  File \"")
			b.WriteString(file)
			b.WriteString("\", line ")
			b.WriteString(itoa(line))
			b.WriteString(", in ")
			b.WriteString(name)
			b.WriteByte('\n')
		}
	}
	cls := "Exception"
	if e.Class != nil {
		cls = e.Class.Name
	}
	b.WriteString(cls)
	if e.Msg != "" {
		b.WriteString(": ")
		b.WriteString(e.Msg)
	} else if e.Args != nil && len(e.Args.V) == 1 {
		b.WriteString(": ")
		b.WriteString(object.Str_(e.Args.V[0]))
	}
	b.WriteByte('\n')
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
