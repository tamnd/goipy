package vm

import (
	"math/big"
	"reflect"

	"github.com/tamnd/goipy/object"
)

// lookupInstanceDunder returns a bound method for `name` on inst's class, if
// the class (or a base) defines it.
func (i *Interp) lookupInstanceDunder(inst *object.Instance, name string) (object.Object, bool) {
	v, ok := classLookup(inst.Class, name)
	if !ok {
		return nil, false
	}
	bound, err := i.bindDescriptor(v, inst, inst.Class)
	if err != nil {
		return nil, false
	}
	return bound, true
}

// callInstanceDunder invokes name(args...) on inst if defined. Returns
// (result, handled, err). handled=false means the class doesn't define the
// method; the caller should fall back.
// Instance dict is checked first (used by stdlib types that store per-instance
// closures), then the class hierarchy.
func (i *Interp) callInstanceDunder(inst *object.Instance, name string, args ...object.Object) (object.Object, bool, error) {
	if fn, ok := inst.Dict.GetStr(name); ok {
		r, err := i.callObject(fn, args, nil)
		return r, true, err
	}
	fn, ok := i.lookupInstanceDunder(inst, name)
	if !ok {
		return nil, false, nil
	}
	r, err := i.callObject(fn, args, nil)
	return r, true, err
}

// tryBinaryDunder dispatches a binary op through __name__ on a, then __rname__
// on b. The pair ("__add__", "__radd__") handles a+b with user instances.
// Returns (result, handled, err).
func (i *Interp) tryBinaryDunder(a, b object.Object, fwd, rev string) (object.Object, bool, error) {
	if ia, ok := a.(*object.Instance); ok {
		r, ok, err := i.callInstanceDunder(ia, fwd, b)
		if ok {
			if err != nil {
				return nil, true, err
			}
			if !isNotImplemented(r) {
				return r, true, nil
			}
		}
	}
	if ib, ok := b.(*object.Instance); ok {
		r, ok, err := i.callInstanceDunder(ib, rev, a)
		if ok {
			if err != nil {
				return nil, true, err
			}
			if !isNotImplemented(r) {
				return r, true, nil
			}
		}
	}
	return nil, false, nil
}

// tryCompareDunder handles ==, !=, <, <=, >, >=. Falls back on NotImplemented.
func (i *Interp) tryCompareDunder(a, b object.Object, kind int) (object.Object, bool, error) {
	fwd, rev := compareDunderNames(kind)
	if fwd == "" {
		return nil, false, nil
	}
	if ia, ok := a.(*object.Instance); ok {
		r, ok, err := i.callInstanceDunder(ia, fwd, b)
		if ok {
			if err != nil {
				return nil, true, err
			}
			if !isNotImplemented(r) {
				return r, true, nil
			}
		}
	}
	if ib, ok := b.(*object.Instance); ok {
		r, ok, err := i.callInstanceDunder(ib, rev, a)
		if ok {
			if err != nil {
				return nil, true, err
			}
			if !isNotImplemented(r) {
				return r, true, nil
			}
		}
	}
	return nil, false, nil
}

func compareDunderNames(kind int) (fwd, rev string) {
	switch kind {
	case cmpEQ:
		return "__eq__", "__eq__"
	case cmpNE:
		return "__ne__", "__ne__"
	case cmpLT:
		return "__lt__", "__gt__"
	case cmpLE:
		return "__le__", "__ge__"
	case cmpGT:
		return "__gt__", "__lt__"
	case cmpGE:
		return "__ge__", "__le__"
	}
	return "", ""
}

func isNotImplemented(o object.Object) bool {
	_, ok := o.(*object.NotImplementedType)
	return ok
}

// instReprHook is installed at Interp construction so object.Repr routes
// through user-defined __repr__. Returns (string, handled).
func (i *Interp) instReprHook(o object.Object) (string, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return "", false
	}
	r, ok, err := i.callInstanceDunder(inst, "__repr__")
	if !ok || err != nil {
		return "", false
	}
	s, ok := r.(*object.Str)
	if !ok {
		return "", false
	}
	return s.V, true
}

func (i *Interp) instStrHook(o object.Object) (string, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return "", false
	}
	if r, ok, err := i.callInstanceDunder(inst, "__str__"); ok && err == nil {
		if s, ok := r.(*object.Str); ok {
			return s.V, true
		}
	}
	if r, ok, err := i.callInstanceDunder(inst, "__repr__"); ok && err == nil {
		if s, ok := r.(*object.Str); ok {
			return s.V, true
		}
	}
	return "", false
}

func (i *Interp) instTruthyHook(o object.Object) (bool, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return false, false
	}
	if r, ok, err := i.callInstanceDunder(inst, "__bool__"); ok && err == nil {
		if bo, ok := r.(*object.Bool); ok {
			return bo.V, true
		}
	}
	if r, ok, err := i.callInstanceDunder(inst, "__len__"); ok && err == nil {
		if n, ok := toInt64(r); ok {
			return n != 0, true
		}
	}
	return false, false
}

func (i *Interp) instHashHook(o object.Object) (uint64, bool, error) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return 0, false, nil
	}
	r, ok, err := i.callInstanceDunder(inst, "__hash__")
	if !ok {
		// No __hash__ defined: use identity hash (CPython default).
		p := reflect.ValueOf(inst).Pointer()
		h := uint64(p>>4) ^ uint64(p>>12)
		return h, true, nil
	}
	if err != nil {
		return 0, true, err
	}
	n, ok := toBigInt(r)
	if !ok {
		return 0, true, object.Errorf(i.typeErr, "__hash__ method should return an integer")
	}
	return new(big.Int).And(n, big.NewInt(0).SetUint64(^uint64(0))).Uint64(), true, nil
}

func (i *Interp) instEqHook(a, b object.Object) (bool, bool, error) {
	ia, aok := a.(*object.Instance)
	ib, bok := b.(*object.Instance)
	if !aok && !bok {
		return false, false, nil
	}
	if aok {
		if r, ok, err := i.callInstanceDunder(ia, "__eq__", b); ok {
			if err != nil {
				return false, true, err
			}
			if !isNotImplemented(r) {
				return object.Truthy(r), true, nil
			}
		}
	}
	if bok {
		if r, ok, err := i.callInstanceDunder(ib, "__eq__", a); ok {
			if err != nil {
				return false, true, err
			}
			if !isNotImplemented(r) {
				return object.Truthy(r), true, nil
			}
		}
	}
	// Fall back to identity for instances when no __eq__ defined.
	return a == b, true, nil
}

// instanceIter turns an instance's __iter__ into an *object.Iter. Also
// supports the __getitem__ iteration protocol (index from 0 until IndexError).
func (i *Interp) instanceIter(inst *object.Instance) (*object.Iter, bool, error) {
	if it, ok, err := i.callInstanceDunder(inst, "__iter__"); ok {
		if err != nil {
			return nil, true, err
		}
		target, tok := it.(*object.Instance)
		if !tok {
			// __iter__ returned a builtin iterable; delegate.
			itr, err := i.getIter(it)
			return itr, true, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			r, ok, err := i.callInstanceDunder(target, "__next__")
			if !ok {
				return nil, false, object.Errorf(i.typeErr, "iter returned non-iterator")
			}
			if err != nil {
				if exc, eok := err.(*object.Exception); eok && exc.Class != nil && exc.Class == i.stopIter {
					return nil, false, nil
				}
				return nil, false, err
			}
			return r, true, nil
		}}, true, nil
	}
	if _, ok := i.lookupInstanceDunder(inst, "__getitem__"); ok {
		idx := int64(0)
		return &object.Iter{Next: func() (object.Object, bool, error) {
			r, _, err := i.callInstanceDunder(inst, "__getitem__", object.NewInt(idx))
			if err != nil {
				if exc, eok := err.(*object.Exception); eok && exc.Class != nil && (exc.Class == i.indexErr || exc.Class == i.stopIter) {
					return nil, false, nil
				}
				return nil, false, err
			}
			idx++
			return r, true, nil
		}}, true, nil
	}
	return nil, false, nil
}

// installDunderHooks wires up object-package hooks so Hash/Eq/Repr/Str/Truthy
// all route through user dunders on Instance.
func (i *Interp) installDunderHooks() {
	object.InstanceReprHook = i.instReprHook
	object.InstanceStrHook = i.instStrHook
	object.InstanceTruthyHook = i.instTruthyHook
	object.InstanceHashHook = i.instHashHook
	object.InstanceEqHook = i.instEqHook
	object.TypeNameHook = func(o object.Object) string {
		if _, ok := o.(*Frame); ok {
			return "frame"
		}
		return ""
	}
}
