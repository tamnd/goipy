package object

import "fmt"

// Exception is a raised Python exception (also implements Go error).
type Exception struct {
	Class *Class
	Args  *Tuple
	Cause *Exception
	Ctx   *Exception
	Msg   string
	// Traceback is the innermost frame of the stack at raise time; each
	// frame the exception propagates through prepends another node.
	Traceback *Traceback
	// Dict holds extra attributes set on the exception instance (e.g. .msg,
	// .filename, .lineno for NetrcParseError). Nil when unused.
	Dict *Dict
}

// Traceback records one frame in an exception's backtrace. Matches the
// surface shape of Python's types.TracebackType: a linked list from
// innermost to outermost frame.
type Traceback struct {
	Code     *Code
	Lasti    int // bytecode offset at which the frame was executing
	Lineno   int // source line (0 means "no info")
	FuncName string
	Next     *Traceback // outer frame (caller)
}

// TracebackFrame is the lightweight shape exposed as `traceback.tb_frame`.
// CPython's frame object is much richer; we only surface `f_code`, which
// is the shape most traceback consumers rely on.
type TracebackFrame struct {
	Code *Code
}

func (e *Exception) Error() string {
	if e.Msg != "" {
		return e.Class.Name + ": " + e.Msg
	}
	if e.Args != nil && len(e.Args.V) == 1 {
		return e.Class.Name + ": " + Str_(e.Args.V[0])
	}
	return e.Class.Name
}

// NewException builds an exception with a single string arg.
func NewException(cls *Class, msg string) *Exception {
	return &Exception{
		Class: cls,
		Args:  &Tuple{V: []Object{&Str{V: msg}}},
		Msg:   msg,
	}
}

// Errorf is a convenience wrapper for Sprintf-style messages.
func Errorf(cls *Class, format string, args ...any) *Exception {
	return NewException(cls, fmt.Sprintf(format, args...))
}

// IsSubclass reports whether c is a subclass of base (reflexive).
func IsSubclass(c, base *Class) bool {
	if c == nil || base == nil {
		return false
	}
	if c == base {
		return true
	}
	for _, b := range c.Bases {
		if IsSubclass(b, base) {
			return true
		}
	}
	return false
}
