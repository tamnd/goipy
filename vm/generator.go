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
	}
	return nil, false
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
