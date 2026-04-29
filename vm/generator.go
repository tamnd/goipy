package vm

import (
	"errors"

	"github.com/tamnd/goipy/object"
)

// errYielded is a sentinel error returned by dispatch when the frame hit
// YIELD_VALUE and should be resumable. The yielded value lives in
// Frame.Yielded.
var errYielded = errors.New("goipy:yielded")

// resumeGenerator steps a generator. `sent` is pushed onto the frame's stack
// before dispatching, which becomes the result value of the pending
// YIELD_VALUE (or is popped by the POP_TOP that follows RETURN_GENERATOR on
// the very first resume).
//
// Returns: (yieldedValue, nil) on yield; (nil, StopIteration(returnValue))
// on return; (nil, err) on unhandled exception.
func (i *Interp) resumeGenerator(gen *object.Generator, sent object.Object) (object.Object, error) {
	if gen.Done {
		return nil, object.Errorf(i.stopIter, "")
	}
	frame := gen.Frame.(*Frame)
	frame.push(sent)
	gen.Started = true
	result, err := i.runFrame(frame)
	if errors.Is(err, errYielded) {
		v := frame.Yielded
		frame.Yielded = nil
		return v, nil
	}
	gen.Done = true
	if err != nil {
		return nil, err
	}
	// Generator returned: raise StopIteration(value).
	exc := object.NewException(i.stopIter, "")
	if result != nil && result != object.None {
		exc.Args = &object.Tuple{V: []object.Object{result}}
	} else {
		exc.Args = &object.Tuple{V: nil}
	}
	return nil, exc
}

// throwGenerator injects an exception into a suspended generator at the last
// yield point. The exception is handled by the generator's own exception table
// (allowing try/except around yield to catch it).
func (i *Interp) throwGenerator(gen *object.Generator, exc *object.Exception) (object.Object, error) {
	if gen.Done {
		return nil, exc
	}
	frame := gen.Frame.(*Frame)
	// Rewind IP to the YIELD_VALUE so the exception table covers it.
	frame.IP = frame.YieldIP
	frame.PendingThrow = exc
	gen.Started = true
	result, err := i.runFrame(frame)
	if errors.Is(err, errYielded) {
		v := frame.Yielded
		frame.Yielded = nil
		return v, nil
	}
	gen.Done = true
	if err != nil {
		return nil, err
	}
	exc2 := object.NewException(i.stopIter, "")
	if result != nil && result != object.None {
		exc2.Args = &object.Tuple{V: []object.Object{result}}
	}
	return nil, exc2
}

// genMethod resolves generator attribute lookups for send/close/throw.
func (i *Interp) genMethod(gen *object.Generator, name string) (object.Object, bool) {
	switch name {
	case "send":
		return &object.BuiltinFunc{Name: "send", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if !gen.Started && len(a) > 0 && a[0] != object.None {
				return nil, object.Errorf(i.typeErr, "can't send non-None value to a just-started generator")
			}
			v := object.Object(object.None)
			if len(a) > 0 {
				v = a[0]
			}
			return i.resumeGenerator(gen, v)
		}}, true
	case "throw":
		return &object.BuiltinFunc{Name: "throw", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "throw() requires an exception argument")
			}
			var exc *object.Exception
			switch v := a[0].(type) {
			case *object.Exception:
				exc = v
			case *object.Class:
				exc = object.NewException(v, "")
			default:
				return nil, object.Errorf(i.typeErr, "exceptions must derive from BaseException")
			}
			return i.throwGenerator(gen, exc)
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			gen.Done = true
			return object.None, nil
		}}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return gen, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return i.resumeGenerator(gen, object.None)
		}}, true
	case "__await__":
		// A coroutine/generator is its own iterator under __await__.
		return &object.BuiltinFunc{Name: "__await__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return gen, nil
		}}, true
	}
	if isAsyncGen(gen) {
		switch name {
		case "__aiter__":
			return &object.BuiltinFunc{Name: "__aiter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return gen, nil
			}}, true
		case "__anext__":
			return &object.BuiltinFunc{Name: "__anext__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return i.makeAsyncGenStep(gen, object.None, nil), nil
			}}, true
		case "asend":
			return &object.BuiltinFunc{Name: "asend", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				v := object.Object(object.None)
				if len(a) > 0 {
					v = a[0]
				}
				return i.makeAsyncGenStep(gen, v, nil), nil
			}}, true
		case "athrow":
			return &object.BuiltinFunc{Name: "athrow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "athrow() requires an exception argument")
				}
				var exc *object.Exception
				switch v := a[0].(type) {
				case *object.Exception:
					exc = v
				case *object.Class:
					exc = object.NewException(v, "")
				default:
					return nil, object.Errorf(i.typeErr, "exceptions must derive from BaseException")
				}
				return i.makeAsyncGenStep(gen, nil, exc), nil
			}}, true
		case "aclose":
			return &object.BuiltinFunc{Name: "aclose", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return i.makeAsyncGenClose(gen), nil
			}}, true
		}
	}
	return nil, false
}

// isAsyncGen reports whether gen wraps an async-generator code object.
func isAsyncGen(gen *object.Generator) bool {
	f, ok := gen.Frame.(*Frame)
	if !ok || f == nil || f.Code == nil {
		return false
	}
	return f.Code.Flags&CO_ASYNC_GENERATOR != 0
}

// makeAsyncGenClose returns the awaitable for agen.aclose(). It throws
// GeneratorExit into the gen; if the gen returns (or finishes with
// GeneratorExit propagating), the coroutine completes with None. If the
// gen yields after GeneratorExit, this is a programming error.
func (i *Interp) makeAsyncGenClose(gen *object.Generator) *object.Iter {
	done := false
	return &object.Iter{
		Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, object.Errorf(i.stopIter, "")
			}
			done = true
			if gen.Done {
				stop := object.NewException(i.stopIter, "")
				stop.Args = &object.Tuple{V: []object.Object{object.None}}
				return nil, false, stop
			}
			exc := object.NewException(i.generatorExit, "")
			_, err := i.throwGenerator(gen, exc)
			if err != nil {
				if e, ok := err.(*object.Exception); ok {
					if object.IsSubclass(e.Class, i.stopIter) ||
						object.IsSubclass(e.Class, i.generatorExit) ||
						object.IsSubclass(e.Class, i.stopAsyncIter) {
						stop := object.NewException(i.stopIter, "")
						stop.Args = &object.Tuple{V: []object.Object{object.None}}
						return nil, false, stop
					}
				}
				return nil, false, err
			}
			// Gen yielded — that's a misuse, but be lenient: still return
			// None to preserve the contract that aclose can't fail.
			stop := object.NewException(i.stopIter, "")
			stop.Args = &object.Tuple{V: []object.Object{object.None}}
			return nil, false, stop
		},
	}
}

// makeAsyncGenStep returns a one-shot awaitable that drives the
// underlying async generator one step. When a coroutine driver SENDs
// into it (or `await` walks it), Next() calls resumeGenerator (or
// throwGenerator if throwExc is set) and reshapes the outcome into the
// StopIteration(value) / StopAsyncIteration that the await machinery
// understands.
func (i *Interp) makeAsyncGenStep(gen *object.Generator, sent object.Object, throwExc *object.Exception) *object.Iter {
	done := false
	return &object.Iter{
		Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, object.Errorf(i.stopAsyncIter, "")
			}
			done = true
			var v object.Object
			var err error
			if throwExc != nil {
				v, err = i.throwGenerator(gen, throwExc)
			} else {
				v, err = i.resumeGenerator(gen, sent)
			}
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					// The async generator returned: surface as StopAsyncIteration
					// so async-for loops terminate cleanly.
					return nil, false, object.Errorf(i.stopAsyncIter, "")
				}
				return nil, false, err
			}
			// The agen yielded v. Deliver as StopIteration(v) so the
			// caller's `await` lifts v to its result.
			yieldedExc := object.NewException(i.stopIter, "")
			yieldedExc.Args = &object.Tuple{V: []object.Object{v}}
			return nil, false, yieldedExc
		},
	}
}

// genIter wraps a Generator as an Iter so for-loops and list()/tuple() work.
func (i *Interp) genIter(gen *object.Generator) *object.Iter {
	return &object.Iter{
		Next: func() (object.Object, bool, error) {
			v, err := i.resumeGenerator(gen, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					return nil, false, nil
				}
				return nil, false, err
			}
			return v, true, nil
		},
	}
}
