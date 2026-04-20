package object

import "fmt"

// Exception is a raised Python exception (also implements Go error).
type Exception struct {
	Class *Class
	Args  *Tuple
	Cause *Exception
	Ctx   *Exception
	Msg   string
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
