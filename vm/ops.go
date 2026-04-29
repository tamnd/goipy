package vm

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) setitem(container, key, val object.Object) error {
	if inst, ok := container.(*object.Instance); ok {
		if _, ok, err := i.callInstanceDunder(inst, "__setitem__", key, val); ok {
			return err
		}
	}
	switch c := container.(type) {
	case *object.List:
		if sl, ok := key.(*object.Slice); ok {
			return i.listSetSlice(c, sl, val)
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "list index out of range")
		}
		c.V[n] = val
		return nil
	case *object.Dict:
		return c.Set(key, val)
	case *object.PyArray:
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "array indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "array index out of range")
		}
		v, err := arrayValidate(c.Typecode, val)
		if err != nil {
			return err
		}
		c.V[n] = v
		return nil
	case *object.Deque:
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "deque indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "deque index out of range")
		}
		c.V[n] = val
		return nil
	case *object.Counter:
		return c.D.Set(key, val)
	case *object.DefaultDict:
		return c.D.Set(key, val)
	case *object.OrderedDict:
		return c.D.Set(key, val)
	case *object.Memoryview:
		if c.Readonly {
			return object.Errorf(i.typeErr, "cannot modify read-only memoryview")
		}
		buf := c.Buf()
		if sl, ok := key.(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(buf))
			if err != nil {
				return err
			}
			if step != 1 {
				return object.Errorf(i.valueErr, "memoryview extended slice assignment not supported")
			}
			src, ok := bytesBytesOrArray(val)
			if !ok {
				items, err := iterate(i, val)
				if err != nil {
					return err
				}
				src = make([]byte, len(items))
				for k, x := range items {
					n, ok := toInt64(x)
					if !ok || n < 0 || n > 255 {
						return object.Errorf(i.valueErr, "byte must be in range(0, 256)")
					}
					src[k] = byte(n)
				}
			}
			if len(src) != stop-start {
				return object.Errorf(i.valueErr, "memoryview assignment: lvalue and rvalue have different structures")
			}
			copy(buf[start:stop], src)
			return nil
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "memoryview indices must be integers")
		}
		L := int64(len(buf))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "memoryview index out of range")
		}
		bv, ok := toInt64(val)
		if !ok || bv < 0 || bv > 255 {
			return object.Errorf(i.valueErr, "byte must be in range(0, 256)")
		}
		buf[n] = byte(bv)
		return nil
	case *object.Bytearray:
		if sl, ok := key.(*object.Slice); ok {
			return i.bytearraySetSlice(c, sl, val)
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "bytearray indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "bytearray index out of range")
		}
		bv, ok := toInt64(val)
		if !ok || bv < 0 || bv > 255 {
			return object.Errorf(i.valueErr, "byte must be in range(0, 256)")
		}
		c.V[n] = byte(bv)
		return nil
	}
	return object.Errorf(i.typeErr, "'%s' does not support item assignment", object.TypeName(container))
}

func (i *Interp) bytearraySetSlice(ba *object.Bytearray, sl *object.Slice, val object.Object) error {
	start, stop, step, err := i.resolveSlice(sl, len(ba.V))
	if err != nil {
		return err
	}
	if step != 1 {
		return object.Errorf(i.valueErr, "extended slice assignment not supported")
	}
	var src []byte
	if bb, ok := bytesBytesOrArray(val); ok {
		src = bb
	} else {
		items, err := iterate(i, val)
		if err != nil {
			return err
		}
		src = make([]byte, len(items))
		for k, x := range items {
			n, ok := toInt64(x)
			if !ok || n < 0 || n > 255 {
				return object.Errorf(i.valueErr, "byte must be in range(0, 256)")
			}
			src[k] = byte(n)
		}
	}
	out := make([]byte, 0, len(ba.V)-(stop-start)+len(src))
	out = append(out, ba.V[:start]...)
	out = append(out, src...)
	out = append(out, ba.V[stop:]...)
	ba.V = out
	return nil
}

func (i *Interp) listSetSlice(l *object.List, sl *object.Slice, val object.Object) error {
	start, stop, step, err := i.resolveSlice(sl, len(l.V))
	if err != nil {
		return err
	}
	if step != 1 {
		return object.Errorf(i.valueErr, "extended slice assignment not supported")
	}
	items, err := iterate(i, val)
	if err != nil {
		return err
	}
	newV := make([]object.Object, 0, len(l.V)-(stop-start)+len(items))
	newV = append(newV, l.V[:start]...)
	newV = append(newV, items...)
	newV = append(newV, l.V[stop:]...)
	l.V = newV
	return nil
}

func (i *Interp) delitem(container, key object.Object) error {
	if inst, ok := container.(*object.Instance); ok {
		if _, ok, err := i.callInstanceDunder(inst, "__delitem__", key); ok {
			return err
		}
	}
	switch c := container.(type) {
	case *object.List:
		if sl, ok := key.(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(c.V))
			if err != nil {
				return err
			}
			if step != 1 {
				return object.Errorf(i.valueErr, "extended slice deletion not supported")
			}
			c.V = append(c.V[:start], c.V[stop:]...)
			return nil
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "list index out of range")
		}
		c.V = append(c.V[:n], c.V[n+1:]...)
		return nil
	case *object.Dict:
		ok, err := c.Delete(key)
		if err != nil {
			return err
		}
		if !ok {
			return object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return nil
	case *object.PyArray:
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "array indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "array index out of range")
		}
		c.V = append(c.V[:n], c.V[n+1:]...)
		return nil
	case *object.Deque:
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "deque indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "deque index out of range")
		}
		c.V = append(c.V[:n], c.V[n+1:]...)
		return nil
	case *object.Counter:
		ok, err := c.D.Delete(key)
		if err != nil {
			return err
		}
		if !ok {
			return object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return nil
	case *object.DefaultDict:
		ok, err := c.D.Delete(key)
		if err != nil {
			return err
		}
		if !ok {
			return object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return nil
	case *object.OrderedDict:
		ok, err := c.D.Delete(key)
		if err != nil {
			return err
		}
		if !ok {
			return object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return nil
	case *object.Bytearray:
		if sl, ok := key.(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(c.V))
			if err != nil {
				return err
			}
			if step != 1 {
				return object.Errorf(i.valueErr, "extended slice deletion not supported")
			}
			c.V = append(c.V[:start], c.V[stop:]...)
			return nil
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "bytearray indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "bytearray index out of range")
		}
		c.V = append(c.V[:n], c.V[n+1:]...)
		return nil
	}
	return object.Errorf(i.typeErr, "'%s' does not support item deletion", object.TypeName(container))
}

func (i *Interp) unaryNeg(v object.Object) (object.Object, error) {
	if inst, ok := v.(*object.Instance); ok {
		if r, ok, err := i.callInstanceDunder(inst, "__neg__"); ok {
			return r, err
		}
	}
	// -Counter: negate counts, keep only positives of the negated values
	if c, ok := v.(*object.Counter); ok {
		out := &object.Counter{D: object.NewDict()}
		keys, vals := c.D.Items()
		for k, key := range keys {
			n, _ := toInt64(vals[k])
			if n < 0 {
				_ = out.D.Set(key, object.NewInt(-n))
			}
		}
		return out, nil
	}
	switch x := v.(type) {
	case *object.Bool:
		r := int64(0)
		if x.V {
			r = -1
		}
		return object.NewInt(r), nil
	case *object.Int:
		return object.IntFromBig(new(big.Int).Neg(&x.V)), nil
	case *object.Float:
		return &object.Float{V: -x.V}, nil
	case *object.Complex:
		return &object.Complex{Real: -x.Real, Imag: -x.Imag}, nil
	}
	return nil, object.Errorf(i.typeErr, "bad operand for unary -: '%s'", object.TypeName(v))
}

// collectionLen returns the length of dict-like collection types.
func collectionLen(v object.Object) (int64, bool) {
	switch x := v.(type) {
	case *object.Dict:
		return int64(x.Len()), true
	case *object.Set:
		return int64(x.Len()), true
	case *object.Frozenset:
		return int64(x.Len()), true
	case *object.Counter:
		return int64(x.D.Len()), true
	case *object.DefaultDict:
		return int64(x.D.Len()), true
	case *object.OrderedDict:
		return int64(x.D.Len()), true
	}
	return 0, false
}

// seqContainerLen returns the length of sequence types (List, Tuple, PyArray, Deque, Range).
func seqContainerLen(v object.Object) (int64, bool) {
	switch x := v.(type) {
	case *object.List:
		return int64(len(x.V)), true
	case *object.Tuple:
		return int64(len(x.V)), true
	case *object.PyArray:
		return int64(len(x.V)), true
	case *object.Deque:
		return int64(len(x.V)), true
	case *object.Range:
		return rangeLen(x), true
	}
	return 0, false
}

func (i *Interp) length(v object.Object) (int64, error) {
	if inst, ok := v.(*object.Instance); ok {
		if r, ok, err := i.callInstanceDunder(inst, "__len__"); ok {
			if err != nil {
				return 0, err
			}
			n, ok := toInt64(r)
			if !ok {
				return 0, object.Errorf(i.typeErr, "__len__ should return an integer")
			}
			return n, nil
		}
	}
	if n, ok := collectionLen(v); ok {
		return n, nil
	}
	if n, ok := seqContainerLen(v); ok {
		return n, nil
	}
	switch x := v.(type) {
	case *object.Str:
		return int64(len(x.Runes())), nil
	case *object.Bytes:
		return int64(len(x.V)), nil
	case *object.Bytearray:
		return int64(len(x.V)), nil
	case *object.Memoryview:
		return int64(x.Stop - x.Start), nil
	case *object.Class:
		if x.EnumData != nil {
			return int64(len(x.EnumData.Members)), nil
		}
	}
	return 0, object.Errorf(i.typeErr, "object of type '%s' has no len()", object.TypeName(v))
}

// --- attribute access ---

func (i *Interp) getAttr(o object.Object, name string) (object.Object, error) {
	// Explicit super(StartCls, Self): walk MRO past StartCls and bind to Self.
	// Mirrors what LOAD_SUPER_ATTR does inline for zero-arg super().
	if sup, ok := o.(*object.Super); ok {
		if sup.Self == nil {
			return nil, object.Errorf(i.attrErr, "'super' object has no attribute '%s'", name)
		}
		var instCls *object.Class
		switch s := sup.Self.(type) {
		case *object.Instance:
			instCls = s.Class
		case *object.Class:
			instCls = s
		default:
			if c, ok := sup.Self.(*object.Class); ok {
				instCls = c
			}
		}
		if instCls == nil {
			return nil, object.Errorf(i.typeErr, "super(): instance has no class")
		}
		v, found := lookupAfter(instCls, sup.StartCls, name)
		if !found {
			return nil, object.Errorf(i.attrErr, "'super' object has no attribute '%s'", name)
		}
		if inst, ok := sup.Self.(*object.Instance); ok {
			return i.bindDescriptor(v, inst, instCls)
		}
		return i.bindDescriptor(v, nil, instCls)
	}
	// Method lookup for str.
	if s, ok := o.(*object.Str); ok {
		if m, ok := strMethod(s, name); ok {
			return m, nil
		}
	}
	if n, ok := o.(*object.Int); ok {
		switch name {
		case "real":
			return n, nil
		case "imag":
			return object.NewInt(0), nil
		case "numerator":
			return n, nil
		case "denominator":
			return object.NewInt(1), nil
		}
		if m, ok := intMethod(n, name); ok {
			return m, nil
		}
	}
	if f, ok := o.(*object.Float); ok {
		switch name {
		case "real":
			return f, nil
		case "imag":
			return &object.Float{V: 0}, nil
		}
		if m, ok := floatMethod(f, name); ok {
			return m, nil
		}
	}
	if r, ok := o.(*object.Range); ok {
		switch name {
		case "start":
			return object.NewInt(r.Start), nil
		case "stop":
			return object.NewInt(r.Stop), nil
		case "step":
			return object.NewInt(r.Step), nil
		}
		if m, ok := rangeMethod(r, name); ok {
			return m, nil
		}
	}
	if l, ok := o.(*object.List); ok {
		if m, ok := listMethod(l, name); ok {
			return m, nil
		}
	}
	if d, ok := o.(*object.Dict); ok {
		if m, ok := dictMethod(d, name); ok {
			return m, nil
		}
	}
	if s, ok := o.(*object.Set); ok {
		if m, ok := setMethod(s, name); ok {
			return m, nil
		}
	}
	if s, ok := o.(*object.Frozenset); ok {
		if m, ok := frozensetMethod(s, name); ok {
			return m, nil
		}
	}
	if b, ok := o.(*object.Bytes); ok {
		if m, ok := bytesMethod(b, name); ok {
			return m, nil
		}
	}
	if ba, ok := o.(*object.Bytearray); ok {
		if m, ok := bytearrayMethod(i, ba, name); ok {
			return m, nil
		}
	}
	if arr, ok := o.(*object.PyArray); ok {
		if m, ok := arrayMethod(i, arr, name); ok {
			return m, nil
		}
	}
	if wr, ok := o.(*object.PyWeakRef); ok {
		switch name {
		case "__callback__":
			if wr.Callback == nil {
				return object.None, nil
			}
			return wr.Callback, nil
		}
		return nil, object.Errorf(i.attrErr, "'weakref' object has no attribute '%s'", name)
	}
	if px, ok := o.(*object.PyProxy); ok {
		// Forward attribute access to the proxied target.
		if px.Target == nil {
			return nil, object.Errorf(i.runtimeErr, "weakly-referenced object no longer exists")
		}
		return i.getAttr(px.Target, name)
	}
	if fin, ok := o.(*object.PyFinalizer); ok {
		switch name {
		case "alive":
			return object.BoolOf(fin.Alive), nil
		case "atexit":
			return object.BoolOf(fin.Atexit), nil
		}
		return nil, object.Errorf(i.attrErr, "'finalize' object has no attribute '%s'", name)
	}
	if dq, ok := o.(*object.Deque); ok {
		if m, ok := dequeMethod(i, dq, name); ok {
			return m, nil
		}
	}
	if c, ok := o.(*object.Counter); ok {
		if m, ok := counterMethod(i, c, name); ok {
			return m, nil
		}
	}
	if dd, ok := o.(*object.DefaultDict); ok {
		if m, ok := defaultDictMethod(i, dd, name); ok {
			return m, nil
		}
	}
	if od, ok := o.(*object.OrderedDict); ok {
		if m, ok := orderedDictMethod(i, od, name); ok {
			return m, nil
		}
	}
	if p, ok := o.(*object.Pattern); ok {
		if m, ok := patternAttr(i, p, name); ok {
			return m, nil
		}
	}
	if mt, ok := o.(*object.Match); ok {
		if m, ok := matchAttr(i, mt, name); ok {
			return m, nil
		}
	}
	if sio, ok := o.(*object.StringIO); ok {
		if m, ok := stringIOAttr(i, sio, name); ok {
			return m, nil
		}
	}
	if bio, ok := o.(*object.BytesIO); ok {
		if m, ok := bytesIOAttr(i, bio, name); ok {
			return m, nil
		}
	}
	if fo, ok := o.(*object.File); ok {
		if m, ok := fileAttr(i, fo, name); ok {
			return m, nil
		}
	}
	if ts, ok := o.(*object.TextStream); ok {
		if m, ok := textStreamAttr(i, ts, name); ok {
			return m, nil
		}
	}
	if h, ok := o.(*object.Hasher); ok {
		if m, ok := hasherAttr(i, h, name); ok {
			return m, nil
		}
	}
	if r, ok := o.(*object.CSVReader); ok {
		if m, ok := csvReaderAttr(i, r, name); ok {
			return m, nil
		}
	}
	if w, ok := o.(*object.CSVWriter); ok {
		if m, ok := csvWriterAttr(i, w, name); ok {
			return m, nil
		}
	}
	if dw, ok := o.(*object.CSVDictWriter); ok {
		if m, ok := csvDictWriterAttr(i, dw, name); ok {
			return m, nil
		}
	}
	if dr, ok := o.(*object.CSVDictReader); ok {
		if m, ok := csvDictReaderAttr(i, dr, name); ok {
			return m, nil
		}
	}
	if do, ok := o.(*object.CSVDialectObj); ok {
		if m, ok := csvDialectObjAttr(do.D, name); ok {
			return m, nil
		}
	}
	if cp, ok := o.(*object.ConfigParserObj); ok {
		if m, ok := configParserAttr(i, cp, name); ok {
			return m, nil
		}
	}
	if sp, ok := o.(*object.SectionProxyObj); ok {
		if m, ok := sectionProxyAttr(i, sp, name); ok {
			return m, nil
		}
	}
	if r, ok := o.(*object.URLParseResult); ok {
		if m, ok := i.urlParseResultAttr(r, name); ok {
			return m, nil
		}
	}
	if u, ok := o.(*object.UUID); ok {
		if m, ok := uuidAttr(u, name); ok {
			return m, nil
		}
	}
	if sm, ok := o.(*object.SequenceMatcher); ok {
		if m, ok := sequenceMatcherAttr(i, sm, name); ok {
			return m, nil
		}
	}
	if mv, ok := o.(*object.Memoryview); ok {
		if a, ok := memoryviewAttr(mv, name); ok {
			return a, nil
		}
	}
	if sl, ok := o.(*object.Slice); ok {
		switch name {
		case "start":
			return sl.Start, nil
		case "stop":
			return sl.Stop, nil
		case "step":
			return sl.Step, nil
		case "indices":
			return &object.BuiltinFunc{Name: "indices", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) != 1 {
					return nil, object.Errorf(i.typeErr, "slice.indices() takes exactly one argument (%d given)", len(a))
				}
				n, ok := toInt64(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "slice indices must be integers or have an __index__ method")
				}
				start, stop, step, err := i.sliceIndices(sl, n)
				if err != nil {
					return nil, err
				}
				return &object.Tuple{V: []object.Object{
					object.NewInt(start),
					object.NewInt(stop),
					object.NewInt(step),
				}}, nil
			}}, nil
		}
	}
	if t, ok := o.(*object.Tuple); ok {
		if m, ok := tupleMethod(t, name); ok {
			return m, nil
		}
	}
	if interp, ok := o.(*object.Interpolation); ok {
		switch name {
		case "value":
			return interp.Value, nil
		case "expression":
			return &object.Str{V: interp.Expression}, nil
		case "conversion":
			if interp.Conversion == "" {
				return object.None, nil
			}
			return &object.Str{V: interp.Conversion}, nil
		case "format_spec":
			return &object.Str{V: interp.FormatSpec}, nil
		}
	}
	if tmpl, ok := o.(*object.Template); ok {
		switch name {
		case "strings":
			elems := make([]object.Object, len(tmpl.Strings))
			for idx, s := range tmpl.Strings {
				elems[idx] = s
			}
			return &object.Tuple{V: elems}, nil
		case "interpolations":
			elems := make([]object.Object, len(tmpl.Interpolations))
			for idx, interp := range tmpl.Interpolations {
				elems[idx] = interp
			}
			return &object.Tuple{V: elems}, nil
		case "values":
			elems := make([]object.Object, len(tmpl.Interpolations))
			for idx, interp := range tmpl.Interpolations {
				elems[idx] = interp.Value
			}
			return &object.Tuple{V: elems}, nil
		}
	}
	if g, ok := o.(*object.Generator); ok {
		if v, ok := genIntrospect(g, name); ok {
			return v, nil
		}
		if m, ok := i.genMethod(g, name); ok {
			return m, nil
		}
	}
	if c, ok := o.(*object.Complex); ok {
		switch name {
		case "real":
			return &object.Float{V: c.Real}, nil
		case "imag":
			return &object.Float{V: c.Imag}, nil
		case "conjugate":
			return &object.BuiltinFunc{Name: "conjugate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Complex{Real: c.Real, Imag: -c.Imag}, nil
			}}, nil
		}
	}
	// Class attr lookup on instance, honoring the descriptor protocol:
	// data descriptors take precedence over inst.Dict; non-data descriptors
	// yield to it.
	if inst, ok := o.(*object.Instance); ok {
		switch name {
		case "__dict__":
			if inst.Class != nil && inst.Class.NoDict {
				return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '__dict__'", inst.Class.Name)
			}
			// If any class in the MRO declared __slots__, slot values
			// shadow inst.Dict on read: present a filtered snapshot
			// so __dict__ holds only the dynamic (non-slot) attrs,
			// matching CPython's surface contract.
			if inst.Class != nil && hasSlotsInMRO(inst.Class) {
				out := object.NewDict()
				ks, vs := inst.Dict.Items()
				for idx, k := range ks {
					kstr, ok := k.(*object.Str)
					if !ok {
						out.Set(k, vs[idx])
						continue
					}
					if slotAllowed(inst.Class, kstr.V) {
						continue
					}
					out.SetStr(kstr.V, vs[idx])
				}
				return out, nil
			}
			return inst.Dict, nil
		case "__class__":
			return inst.Class, nil
		}
		var clsAttr object.Object
		var clsFound bool
		if inst.Class != nil {
			clsAttr, clsFound = classLookup(inst.Class, name)
		}
		if clsFound && isDataDescriptor(clsAttr) {
			return i.bindDescriptor(clsAttr, inst, inst.Class)
		}
		if v, ok := inst.Dict.GetStr(name); ok {
			return v, nil
		}
		if clsFound {
			return i.bindDescriptor(clsAttr, inst, inst.Class)
		}
		// __getattr__ fallback — called when normal lookup fails.
		if inst.Class != nil {
			if gaFn, ok := classLookup(inst.Class, "__getattr__"); ok {
				if bf, ok := gaFn.(*object.BuiltinFunc); ok {
					return bf.Call(nil, []object.Object{inst, &object.Str{V: name}}, nil)
				}
			}
		}
		clsName := "object"
		if inst.Class != nil {
			clsName = inst.Class.Name
		}
		return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", clsName, name)
	}
	if bf, ok := o.(*object.BuiltinFunc); ok {
		if name == "__name__" {
			return &object.Str{V: bf.Name}, nil
		}
		if bf.Attrs != nil {
			if v, ok := bf.Attrs.GetStr(name); ok {
				return v, nil
			}
		}
	}
	if p, ok := o.(*object.Property); ok {
		switch name {
		case "fget":
			if p.Fget == nil {
				return object.None, nil
			}
			return p.Fget, nil
		case "fset":
			if p.Fset == nil {
				return object.None, nil
			}
			return p.Fset, nil
		case "fdel":
			if p.Fdel == nil {
				return object.None, nil
			}
			return p.Fdel, nil
		case "setter":
			return &object.BuiltinFunc{Name: "setter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Property{Fget: p.Fget, Fset: a[0], Fdel: p.Fdel}, nil
			}}, nil
		case "deleter":
			return &object.BuiltinFunc{Name: "deleter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Property{Fget: p.Fget, Fset: p.Fset, Fdel: a[0]}, nil
			}}, nil
		case "getter":
			return &object.BuiltinFunc{Name: "getter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Property{Fget: a[0], Fset: p.Fset, Fdel: p.Fdel}, nil
			}}, nil
		}
	}
	if fn, ok := o.(*object.Function); ok {
		switch name {
		case "__name__":
			return &object.Str{V: fn.Name}, nil
		case "__qualname__":
			if fn.QualName != "" {
				return &object.Str{V: fn.QualName}, nil
			}
			return &object.Str{V: fn.Name}, nil
		case "__doc__":
			if fn.Doc != nil && fn.Doc != object.None {
				return fn.Doc, nil
			}
			return object.None, nil
		case "__code__":
			if fn.Code == nil {
				return object.None, nil
			}
			return fn.Code, nil
		case "__defaults__":
			if fn.Defaults == nil {
				return object.None, nil
			}
			return fn.Defaults, nil
		case "__kwdefaults__":
			if fn.KwDefaults == nil {
				return object.None, nil
			}
			return fn.KwDefaults, nil
		case "__annotations__":
			if fn.Annotations != nil {
				return fn.Annotations, nil
			}
			if fn.Annotate != nil {
				// PEP 649: call __annotate__(1) for VALUE format.
				v, err := i.callObject(fn.Annotate, []object.Object{object.NewInt(1)}, nil)
				if err != nil {
					return nil, err
				}
				fn.Annotations = v
				return v, nil
			}
			d := object.NewDict()
			fn.Annotations = d
			return d, nil
		case "__closure__":
			if fn.Closure == nil || len(fn.Closure.V) == 0 {
				return object.None, nil
			}
			return fn.Closure, nil
		case "__annotate__":
			if fn.Annotate == nil {
				return object.None, nil
			}
			return fn.Annotate, nil
		case "__globals__":
			if fn.Globals == nil {
				return object.NewDict(), nil
			}
			return fn.Globals, nil
		case "__module__":
			if fn.Module != nil && fn.Module != object.None {
				return fn.Module, nil
			}
			if fn.Globals != nil {
				if v, ok := fn.Globals.GetStr("__name__"); ok {
					return v, nil
				}
			}
			return object.None, nil
		case "__dict__":
			if fn.Dict == nil {
				fn.Dict = object.NewDict()
			}
			return fn.Dict, nil
		}
		if fn.Dict != nil {
			if v, ok := fn.Dict.GetStr(name); ok {
				return v, nil
			}
		}
	}
	if bm, ok := o.(*object.BoundMethod); ok {
		switch name {
		case "__self__":
			if bm.Self == nil {
				return object.None, nil
			}
			return bm.Self, nil
		case "__func__":
			if bm.Fn == nil {
				return object.None, nil
			}
			return bm.Fn, nil
		}
		if bm.Fn != nil {
			return i.getAttr(bm.Fn, name)
		}
	}
	if cls, ok := o.(*object.Class); ok {
		switch name {
		case "__name__":
			return &object.Str{V: cls.Name}, nil
		case "__bases__":
			bs := make([]object.Object, len(cls.Bases))
			for k, b := range cls.Bases {
				bs[k] = b
			}
			return &object.Tuple{V: bs}, nil
		case "__mro__":
			mro := computeMRO(cls)
			out := make([]object.Object, len(mro))
			for k, c := range mro {
				out[k] = c
			}
			return &object.Tuple{V: out}, nil
		}
		if v, ok := classLookup(cls, name); ok {
			return i.bindDescriptor(v, nil, cls)
		}
		return nil, object.Errorf(i.attrErr, "type object '%s' has no attribute '%s'", cls.Name, name)
	}
	if m, ok := o.(*object.Module); ok {
		if name == "__name__" {
			return &object.Str{V: m.Name}, nil
		}
		if name == "__spec__" || name == "__loader__" || name == "__package__" {
			return object.None, nil
		}
		if v, ok := m.Dict.GetStr(name); ok {
			return v, nil
		}
	}
	if e, ok := o.(*object.Exception); ok {
		switch name {
		case "args":
			return e.Args, nil
		case "__traceback__":
			if e.Traceback == nil {
				return object.None, nil
			}
			return e.Traceback, nil
		case "__cause__":
			if e.Cause == nil {
				return object.None, nil
			}
			return e.Cause, nil
		case "__context__":
			if e.Ctx == nil {
				return object.None, nil
			}
			return e.Ctx, nil
		case "__suppress_context__":
			if e.Dict != nil {
				if v, ok := e.Dict.GetStr("__suppress_context__"); ok {
					return v, nil
				}
			}
			return object.False, nil
		}
		// PEP 654 group attributes — exposed only when this is a group.
		if object.IsSubclass(e.Class, i.baseExcGroup) {
			switch name {
			case "message":
				return &object.Str{V: egMessage(e)}, nil
			case "exceptions":
				inners := egInners(e)
				out := make([]object.Object, len(inners))
				copy(out, inners)
				return &object.Tuple{V: out}, nil
			}
		}
		// Check extra instance attributes (e.g. NetrcParseError.msg).
		if e.Dict != nil {
			if v, ok := e.Dict.GetStr(name); ok {
				return v, nil
			}
		}
		// Fall through to class dict lookup for method access.
		if e.Class != nil {
			if v, found := classLookup(e.Class, name); found {
				switch v.(type) {
				case *object.BuiltinFunc, *object.Function:
					return &object.BoundMethod{Self: e, Fn: v}, nil
				}
				return v, nil
			}
			// __getattr__ fallback for exceptions with custom attribute logic.
			if gaFn, ok := classLookup(e.Class, "__getattr__"); ok {
				return i.callObject(gaFn, []object.Object{e, &object.Str{V: name}}, nil)
			}
		}
		return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", e.Class.Name, name)
	}
	if tb, ok := o.(*object.Traceback); ok {
		switch name {
		case "tb_next":
			if tb.Next == nil {
				return object.None, nil
			}
			return tb.Next, nil
		case "tb_lineno":
			return object.NewInt(int64(tb.Lineno)), nil
		case "tb_lasti":
			return object.NewInt(int64(tb.Lasti)), nil
		case "tb_frame":
			return &object.TracebackFrame{Code: tb.Code}, nil
		}
	}
	if tf, ok := o.(*object.TracebackFrame); ok {
		switch name {
		case "f_code":
			return tf.Code, nil
		}
	}
	if c, ok := o.(*object.Code); ok {
		switch name {
		case "co_name":
			return &object.Str{V: c.Name}, nil
		case "co_qualname":
			if c.QualName != "" {
				return &object.Str{V: c.QualName}, nil
			}
			return &object.Str{V: c.Name}, nil
		case "co_filename":
			return &object.Str{V: c.Filename}, nil
		case "co_firstlineno":
			return object.NewInt(int64(c.FirstLineNo)), nil
		case "co_argcount":
			return object.NewInt(int64(c.ArgCount)), nil
		case "co_posonlyargcount":
			return object.NewInt(int64(c.PosOnlyArgCount)), nil
		case "co_kwonlyargcount":
			return object.NewInt(int64(c.KwOnlyArgCount)), nil
		case "co_nlocals":
			return object.NewInt(int64(c.NLocals)), nil
		case "co_stacksize":
			return object.NewInt(int64(c.Stacksize)), nil
		case "co_flags":
			return object.NewInt(int64(c.Flags)), nil
		case "co_consts":
			out := make([]object.Object, len(c.Consts))
			copy(out, c.Consts)
			return &object.Tuple{V: out}, nil
		case "co_names":
			out := make([]object.Object, len(c.Names))
			for k, s := range c.Names {
				out[k] = &object.Str{V: s}
			}
			return &object.Tuple{V: out}, nil
		case "co_varnames":
			// Fast locals only (CO_FAST_LOCAL bit set, CO_FAST_HIDDEN clear).
			var vs []object.Object
			for k, n := range c.LocalsPlusNames {
				if k >= len(c.LocalsPlusKinds) {
					break
				}
				kind := c.LocalsPlusKinds[k]
				if kind&object.FastLocal != 0 && kind&object.FastHidden == 0 {
					vs = append(vs, &object.Str{V: n})
				}
			}
			return &object.Tuple{V: vs}, nil
		case "co_freevars":
			out := make([]object.Object, len(c.FreeVars))
			for k, s := range c.FreeVars {
				out[k] = &object.Str{V: s}
			}
			return &object.Tuple{V: out}, nil
		case "co_cellvars":
			out := make([]object.Object, len(c.CellVars))
			for k, s := range c.CellVars {
				out[k] = &object.Str{V: s}
			}
			return &object.Tuple{V: out}, nil
		case "co_code":
			b := make([]byte, len(c.Bytecode))
			copy(b, c.Bytecode)
			return &object.Bytes{V: b}, nil
		case "co_lnotab":
			b := make([]byte, len(c.LineTable))
			copy(b, c.LineTable)
			return &object.Bytes{V: b}, nil
		case "co_linetable":
			b := make([]byte, len(c.LineTable))
			copy(b, c.LineTable)
			return &object.Bytes{V: b}, nil
		case "co_lines":
			cc := c
			return &object.BuiltinFunc{Name: "co_lines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				// PEP 626: yield (start, end, line) triples covering the
				// bytecode. Without a full linetable decoder we emit a
				// single span (0, len(bytecode), FirstLineNo) — accurate
				// enough for tools that just want *some* line attribution.
				yielded := false
				return &object.Iter{Next: func() (object.Object, bool, error) {
					if yielded {
						return nil, false, nil
					}
					yielded = true
					return &object.Tuple{V: []object.Object{
						object.NewInt(0),
						object.NewInt(int64(len(cc.Bytecode))),
						object.NewInt(int64(cc.FirstLineNo)),
					}}, true, nil
				}}, nil
			}}, nil
		}
	}
	if fr, ok := o.(*Frame); ok {
		switch name {
		case "f_code":
			return fr.Code, nil
		case "f_globals":
			if fr.Globals == nil {
				return object.NewDict(), nil
			}
			return fr.Globals, nil
		case "f_builtins":
			if fr.Builtins == nil {
				return object.NewDict(), nil
			}
			return fr.Builtins, nil
		case "f_locals":
			return frameLocalsView(fr), nil
		case "f_back":
			if fr.Back == nil {
				return object.None, nil
			}
			return fr.Back, nil
		case "f_lineno":
			line := 0
			if fr.Code != nil {
				line = fr.Code.FirstLineNo
			}
			return object.NewInt(int64(line)), nil
		case "f_lasti":
			return object.NewInt(int64(fr.LastIP)), nil
		case "f_trace":
			if fr.LocalTrace == nil {
				return object.None, nil
			}
			return fr.LocalTrace, nil
		case "f_trace_lines":
			return object.True, nil
		case "f_trace_opcodes":
			return object.False, nil
		}
	}
	return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", object.TypeName(o), name)
}

func (i *Interp) setAttr(o object.Object, name string, val object.Object) error {
	if inst, ok := o.(*object.Instance); ok {
		if inst.Class != nil {
			if desc, ok := classLookup(inst.Class, name); ok {
				if p, ok := desc.(*object.Property); ok && p.Fset != nil {
					_, err := i.callObject(p.Fset, []object.Object{inst, val}, nil)
					return err
				}
				if dinst, ok := desc.(*object.Instance); ok {
					if setFn, ok := classLookup(dinst.Class, "__set__"); ok {
						_, err := i.callObject(setFn, []object.Object{dinst, inst, val}, nil)
						return err
					}
				}
			}
			if saFn, ok := classLookup(inst.Class, "__setattr__"); ok {
				_, err := i.callObject(saFn, []object.Object{inst, &object.Str{V: name}, val}, nil)
				return err
			}
			if inst.Class.NoDict && !slotAllowed(inst.Class, name) {
				return object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", inst.Class.Name, name)
			}
		}
		inst.Dict.SetStr(name, val)
		return nil
	}
	if cls, ok := o.(*object.Class); ok {
		cls.Dict.SetStr(name, val)
		object.BumpClassEpoch()
		return nil
	}
	if m, ok := o.(*object.Module); ok {
		m.Dict.SetStr(name, val)
		return nil
	}
	if fn, ok := o.(*object.Function); ok {
		if fn.Dict == nil {
			fn.Dict = object.NewDict()
		}
		fn.Dict.SetStr(name, val)
		return nil
	}
	if fr, ok := o.(*Frame); ok {
		switch name {
		case "f_trace":
			if _, isNone := val.(*object.NoneType); isNone {
				fr.LocalTrace = nil
			} else {
				fr.LocalTrace = val
			}
			return nil
		case "f_trace_lines", "f_trace_opcodes":
			// Accepted but ignored — goipy fires line events whenever
			// f_trace is set (matching default 3.13 behaviour) and never
			// fires opcode events (out of scope).
			return nil
		}
	}
	if e, ok := o.(*object.Exception); ok {
		switch name {
		case "__traceback__":
			switch tb := val.(type) {
			case *object.NoneType:
				e.Traceback = nil
			case *object.Traceback:
				e.Traceback = tb
			default:
				return object.Errorf(i.typeErr, "__traceback__ must be a traceback or None")
			}
			return nil
		case "__cause__":
			if _, isNone := val.(*object.NoneType); isNone {
				e.Cause = nil
			} else if cx, ok := val.(*object.Exception); ok {
				e.Cause = cx
			} else {
				return object.Errorf(i.typeErr, "exception cause must be None or derive from BaseException")
			}
			// Setting __cause__ implicitly sets __suppress_context__ to True.
			if e.Dict == nil {
				e.Dict = object.NewDict()
			}
			e.Dict.SetStr("__suppress_context__", object.True)
			return nil
		case "__context__":
			if _, isNone := val.(*object.NoneType); isNone {
				e.Ctx = nil
			} else if cx, ok := val.(*object.Exception); ok {
				e.Ctx = cx
			} else {
				return object.Errorf(i.typeErr, "exception context must be None or derive from BaseException")
			}
			return nil
		}
		if e.Dict == nil {
			e.Dict = object.NewDict()
		}
		e.Dict.SetStr(name, val)
		return nil
	}
	if fin, ok := o.(*object.PyFinalizer); ok {
		if name == "atexit" {
			fin.Atexit = object.Truthy(val)
			return nil
		}
	}
	return object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", object.TypeName(o), name)
}

func (i *Interp) delAttr(o object.Object, name string) error {
	if inst, ok := o.(*object.Instance); ok {
		if desc, ok := classLookup(inst.Class, name); ok {
			if p, ok := desc.(*object.Property); ok && p.Fdel != nil {
				_, err := i.callObject(p.Fdel, []object.Object{inst}, nil)
				return err
			}
			if dinst, ok := desc.(*object.Instance); ok {
				if delFn, ok := classLookup(dinst.Class, "__delete__"); ok {
					_, err := i.callObject(delFn, []object.Object{dinst, inst}, nil)
					return err
				}
			}
		}
		if inst.Class != nil && inst.Class.NoDict && !slotAllowed(inst.Class, name) {
			return object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", inst.Class.Name, name)
		}
		ok, _ := inst.Dict.Delete(&object.Str{V: name})
		if !ok {
			return object.Errorf(i.attrErr, "no attribute '%s'", name)
		}
		return nil
	}
	return object.Errorf(i.attrErr, "can't delete attribute")
}

// isDataDescriptor reports whether v has __set__ or __delete__ on its type,
// making it take precedence over instance dict in attribute lookup.
func isDataDescriptor(v object.Object) bool {
	if _, ok := v.(*object.Property); ok {
		return true
	}
	inst, ok := v.(*object.Instance)
	if !ok {
		return false
	}
	if _, ok := classLookup(inst.Class, "__set__"); ok {
		return true
	}
	if _, ok := classLookup(inst.Class, "__delete__"); ok {
		return true
	}
	return false
}

// matchClass implements PEP 634 class-pattern extraction. Returns a tuple of
// attributes on match or None on miss. count is positional sub-pattern count;
// kwnamesObj is a tuple of keyword attribute names that follow them.
func (i *Interp) matchClass(subject, cls, kwnamesObj object.Object, count int) (object.Object, error) {
	kwnames, _ := kwnamesObj.(*object.Tuple)
	nkw := 0
	if kwnames != nil {
		nkw = len(kwnames.V)
	}
	if !matchTypeCheck(subject, cls) {
		return object.None, nil
	}
	if bf, ok := cls.(*object.BuiltinFunc); ok && isSpecialMatchClass(bf.Name) {
		if count == 0 && nkw == 0 {
			return &object.Tuple{V: nil}, nil
		}
		if count == 1 && nkw == 0 {
			return &object.Tuple{V: []object.Object{subject}}, nil
		}
		if count > 1 {
			return nil, object.Errorf(i.typeErr, "match() accepts at most 1 positional sub-pattern for builtin class %s", bf.Name)
		}
	}
	var matchArgs []string
	if count > 0 {
		uc, ok := cls.(*object.Class)
		if !ok {
			return nil, object.Errorf(i.typeErr, "class pattern requires __match_args__")
		}
		v, ok := classLookup(uc, "__match_args__")
		if !ok {
			return nil, object.Errorf(i.typeErr, "%s has no __match_args__", uc.Name)
		}
		t, ok := v.(*object.Tuple)
		if !ok {
			return nil, object.Errorf(i.typeErr, "__match_args__ must be a tuple")
		}
		if len(t.V) < count {
			return nil, object.Errorf(i.typeErr, "%s has %d positional patterns but __match_args__ has %d", uc.Name, count, len(t.V))
		}
		for k := 0; k < count; k++ {
			s, ok := t.V[k].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "__match_args__ elements must be strings")
			}
			matchArgs = append(matchArgs, s.V)
		}
	}
	attrs := make([]object.Object, 0, count+nkw)
	for _, name := range matchArgs {
		v, gerr := i.getAttr(subject, name)
		if gerr != nil {
			return object.None, nil
		}
		attrs = append(attrs, v)
	}
	if kwnames != nil {
		for _, n := range kwnames.V {
			s, ok := n.(*object.Str)
			if !ok {
				continue
			}
			v, gerr := i.getAttr(subject, s.V)
			if gerr != nil {
				return object.None, nil
			}
			attrs = append(attrs, v)
		}
	}
	return &object.Tuple{V: attrs}, nil
}

func matchTypeCheck(o, t object.Object) bool {
	if cls, ok := t.(*object.Class); ok {
		if inst, ok := o.(*object.Instance); ok {
			return object.IsSubclass(inst.Class, cls)
		}
		if e, ok := o.(*object.Exception); ok {
			return object.IsSubclass(e.Class, cls)
		}
		return false
	}
	if bf, ok := t.(*object.BuiltinFunc); ok {
		return matchBuiltinType(o, bf.Name)
	}
	return false
}

func isSpecialMatchClass(name string) bool {
	switch name {
	case "bool", "bytearray", "bytes", "dict", "float", "frozenset",
		"int", "list", "set", "str", "tuple":
		return true
	}
	return false
}

func matchBuiltinTypeScalar(o object.Object, name string) (bool, bool) {
	switch name {
	case "int":
		if _, ok := o.(*object.Int); ok {
			return true, true
		}
		_, ok := o.(*object.Bool)
		return ok, true
	case "bool":
		_, ok := o.(*object.Bool)
		return ok, true
	case "float":
		_, ok := o.(*object.Float)
		return ok, true
	case "str":
		_, ok := o.(*object.Str)
		return ok, true
	case "bytes":
		_, ok := o.(*object.Bytes)
		return ok, true
	case "bytearray":
		_, ok := o.(*object.Bytearray)
		return ok, true
	case "memoryview":
		_, ok := o.(*object.Memoryview)
		return ok, true
	}
	return false, false
}

func matchBuiltinTypeContainer(o object.Object, name string) (bool, bool) {
	switch name {
	case "list":
		_, ok := o.(*object.List)
		return ok, true
	case "tuple":
		if _, ok := o.(*object.Tuple); ok {
			return true, true
		}
		// Named-tuple-like instances (e.g. ModuleInfo) declare __namedtuple__ = True.
		if inst, ok := o.(*object.Instance); ok {
			if _, ok2 := inst.Class.Dict.GetStr("__namedtuple__"); ok2 {
				return true, true
			}
		}
		return false, true
	case "dict":
		_, ok := o.(*object.Dict)
		return ok, true
	case "set":
		_, ok := o.(*object.Set)
		return ok, true
	case "frozenset":
		_, ok := o.(*object.Frozenset)
		return ok, true
	case "type":
		_, ok := o.(*object.Class)
		return ok, true
	case "Template":
		_, ok := o.(*object.Template)
		return ok, true
	case "Interpolation":
		_, ok := o.(*object.Interpolation)
		return ok, true
	case "weakref.ref":
		_, ok := o.(*object.PyWeakRef)
		return ok, true
	case "module":
		_, ok := o.(*object.Module)
		return ok, true
	}
	return false, false
}

func matchBuiltinType(o object.Object, name string) bool {
	if r, ok := matchBuiltinTypeScalar(o, name); ok {
		return r
	}
	r, _ := matchBuiltinTypeContainer(o, name)
	return r
}

// bindDescriptor applies the descriptor protocol to v found in a class MRO.
// inst is nil when the lookup came from a class rather than an instance.
func (i *Interp) bindDescriptor(v object.Object, inst *object.Instance, cls *object.Class) (object.Object, error) {
	switch d := v.(type) {
	case *object.Property:
		if inst == nil {
			return d, nil // accessed on the class itself
		}
		return i.callObject(d.Fget, []object.Object{inst}, nil)
	case *object.ClassMethod:
		return &object.BoundMethod{Self: cls, Fn: d.Fn}, nil
	case *object.StaticMethod:
		return d.Fn, nil
	case *object.Function:
		if inst != nil {
			return &object.BoundMethod{Self: inst, Fn: d}, nil
		}
		return d, nil
	case *object.BuiltinFunc:
		if inst != nil {
			return &object.BoundMethod{Self: inst, Fn: d}, nil
		}
		return d, nil
	case *object.Instance:
		// User descriptor: if its class defines __get__, invoke it.
		if getFn, ok := classLookup(d.Class, "__get__"); ok {
			owner := object.Object(cls)
			self := object.Object(object.None)
			if inst != nil {
				self = inst
			}
			return i.callObject(getFn, []object.Object{d, self, owner}, nil)
		}
	case *cachedProperty:
		if inst == nil {
			return d, nil
		}
		r, err := i.callObject(d.fn, []object.Object{inst}, nil)
		if err != nil {
			return nil, err
		}
		if d.name != "" {
			inst.Dict.SetStr(d.name, r)
		}
		return r, nil
	}
	return v, nil
}

// sliceIndices implements slice.indices(n): normalises sl's start/stop/step
// against length n, mirroring CPython's PySlice_GetIndices semantics. Used by
// C-extensions and pandas-like indexers to turn a possibly-Pythonic slice
// (negatives, None) into concrete integer bounds.
func (i *Interp) sliceIndices(sl *object.Slice, length int64) (int64, int64, int64, error) {
	step := int64(1)
	if sl.Step != nil && sl.Step != object.None {
		s, ok := toInt64(sl.Step)
		if !ok {
			return 0, 0, 0, object.Errorf(i.typeErr, "slice indices must be integers or have an __index__ method")
		}
		if s == 0 {
			return 0, 0, 0, object.Errorf(i.valueErr, "slice step cannot be zero")
		}
		step = s
	}
	negStep := step < 0
	defaultStart := int64(0)
	defaultStop := length
	if negStep {
		defaultStart = length - 1
		defaultStop = -1
	}
	clip := func(v object.Object, dflt int64) (int64, error) {
		if v == nil || v == object.None {
			return dflt, nil
		}
		x, ok := toInt64(v)
		if !ok {
			return 0, object.Errorf(i.typeErr, "slice indices must be integers or have an __index__ method")
		}
		if x < 0 {
			x += length
			if x < 0 {
				if negStep {
					x = -1
				} else {
					x = 0
				}
			}
		} else if x >= length {
			if negStep {
				x = length - 1
			} else {
				x = length
			}
		}
		return x, nil
	}
	start, err := clip(sl.Start, defaultStart)
	if err != nil {
		return 0, 0, 0, err
	}
	stop, err := clip(sl.Stop, defaultStop)
	if err != nil {
		return 0, 0, 0, err
	}
	return start, stop, step, nil
}

// lookupAfter walks MRO(instCls) and returns the first attribute named `name`
// found in a class that appears strictly after `startCls` in the order.
func lookupAfter(instCls, startCls *object.Class, name string) (object.Object, bool) {
	mro := computeMRO(instCls)
	past := false
	for _, c := range mro {
		if past {
			if v, ok := c.Dict.GetStr(name); ok {
				return v, true
			}
		}
		if c == startCls {
			past = true
		}
	}
	return nil, false
}

// computeMRO returns the C3 linearization of the class hierarchy.
// L[C] = C + merge(L[B1], L[B2], ..., L[Bn], [B1, B2, ..., Bn])
// On unresolvable conflict (no good head exists), falls back to a
// depth-first traversal so callers never observe a panic; CPython would
// raise TypeError there, but we prefer graceful degradation to avoid
// crashing existing fixtures whose hierarchies happen to be ambiguous.
func computeMRO(c *object.Class) []*object.Class {
	if len(c.Bases) == 0 {
		return []*object.Class{c}
	}
	if len(c.Bases) == 1 {
		return append([]*object.Class{c}, computeMRO(c.Bases[0])...)
	}
	parents := make([][]*object.Class, 0, len(c.Bases)+1)
	for _, b := range c.Bases {
		parents = append(parents, computeMRO(b))
	}
	parents = append(parents, append([]*object.Class{}, c.Bases...))
	merged, ok := c3Merge(parents)
	if !ok {
		return computeMRODepthFirst(c)
	}
	return append([]*object.Class{c}, merged...)
}

// c3Merge implements the C3 merge operation. Returns (result, true) on
// success, or (nil, false) if no consistent linearization exists.
func c3Merge(lists [][]*object.Class) ([]*object.Class, bool) {
	working := make([][]*object.Class, len(lists))
	for i, l := range lists {
		working[i] = append([]*object.Class{}, l...)
	}
	var out []*object.Class
	for {
		nonEmpty := false
		for _, l := range working {
			if len(l) > 0 {
				nonEmpty = true
				break
			}
		}
		if !nonEmpty {
			return out, true
		}
		var pick *object.Class
		for _, l := range working {
			if len(l) == 0 {
				continue
			}
			head := l[0]
			inTail := false
			for _, other := range working {
				for j := 1; j < len(other); j++ {
					if other[j] == head {
						inTail = true
						break
					}
				}
				if inTail {
					break
				}
			}
			if !inTail {
				pick = head
				break
			}
		}
		if pick == nil {
			return nil, false
		}
		out = append(out, pick)
		for i, l := range working {
			if len(l) > 0 && l[0] == pick {
				working[i] = l[1:]
			}
		}
	}
}

// computeMRODepthFirst is the legacy fallback used when C3 cannot find a
// consistent ordering. Depth-first, dedup, no rebalancing.
func computeMRODepthFirst(c *object.Class) []*object.Class {
	var out []*object.Class
	seen := map[*object.Class]bool{}
	var walk func(*object.Class)
	walk = func(x *object.Class) {
		if seen[x] {
			return
		}
		seen[x] = true
		out = append(out, x)
		for _, b := range x.Bases {
			walk(b)
		}
	}
	walk(c)
	return out
}

func classLookup(c *object.Class, name string) (object.Object, bool) {
	epoch := object.ClassEpoch()
	c.Mu.Lock()
	if c.MethodCache != nil {
		if e, ok := c.MethodCache[name]; ok && e.Epoch == epoch {
			c.Mu.Unlock()
			return e.Val, e.Found
		}
	}
	c.Mu.Unlock()
	v, found := classLookupSlow(c, name)
	c.Mu.Lock()
	if c.MethodCache == nil {
		c.MethodCache = make(map[string]object.MethodCacheEntry, 8)
	}
	c.MethodCache[name] = object.MethodCacheEntry{Val: v, Found: found, Epoch: epoch}
	c.Mu.Unlock()
	return v, found
}

func classLookupSlow(c *object.Class, name string) (object.Object, bool) {
	if v, ok := c.Dict.GetStr(name); ok {
		return v, true
	}
	for _, b := range c.Bases {
		if v, ok := classLookup(b, name); ok {
			return v, true
		}
	}
	return nil, false
}

// --- iteration ---

func (i *Interp) getIter(v object.Object) (*object.Iter, error) {
	if inst, ok := v.(*object.Instance); ok {
		if it, ok, err := i.instanceIter(inst); ok {
			return it, err
		}
	}
	switch x := v.(type) {
	case *object.Iter:
		return x, nil
	case *object.Generator:
		return i.genIter(x), nil
	case *object.List:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Tuple:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Str:
		rs := x.Runes()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(rs) {
				return nil, false, nil
			}
			r := &object.Str{V: string(rs[idx])}
			idx++
			return r, true, nil
		}}, nil
	case *object.Bytes:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := object.NewInt(int64(x.V[idx]))
			idx++
			return r, true, nil
		}}, nil
	case *object.Bytearray:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := object.NewInt(int64(x.V[idx]))
			idx++
			return r, true, nil
		}}, nil
	case *object.URLParseResult:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if v, ok := urlParseResultGetItem(x, idx); ok {
				idx++
				return v, true, nil
			}
			return nil, false, nil
		}}, nil
	case *object.CSVReader:
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if x.Pos >= len(x.Rows) {
				return nil, false, nil
			}
			row := x.Rows[x.Pos]
			x.Pos++
			x.LineNo++
			vs := make([]object.Object, len(row))
			for k, s := range row {
				vs[k] = &object.Str{V: s}
			}
			return &object.List{V: vs}, true, nil
		}}, nil
	case *object.CSVDictReader:
		return &object.Iter{Next: func() (object.Object, bool, error) {
			return csvDictReaderNextRow(x)
		}}, nil
	case *object.Memoryview:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			buf := x.Buf()
			if idx >= len(buf) {
				return nil, false, nil
			}
			r := object.NewInt(int64(buf[idx]))
			idx++
			return r, true, nil
		}}, nil
	case *object.Dict:
		keys, _ := x.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			r := keys[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.PyArray:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Deque:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Counter:
		keys, _ := x.D.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			r := keys[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.DefaultDict:
		keys, _ := x.D.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			r := keys[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.OrderedDict:
		keys, _ := x.D.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			r := keys[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Set:
		items := x.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			r := items[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Frozenset:
		items := x.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			r := items[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Range:
		cur := x.Start
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if x.Step > 0 && cur >= x.Stop {
				return nil, false, nil
			}
			if x.Step < 0 && cur <= x.Stop {
				return nil, false, nil
			}
			r := object.NewInt(cur)
			cur += x.Step
			return r, true, nil
		}}, nil
	case *object.Template:
		// Iteration yields interleaved non-empty str parts and all interpolations.
		var flat []object.Object
		for idx, s := range x.Strings {
			if s.V != "" {
				flat = append(flat, s)
			}
			if idx < len(x.Interpolations) {
				flat = append(flat, x.Interpolations[idx])
			}
		}
		pos := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if pos >= len(flat) {
				return nil, false, nil
			}
			r := flat[pos]
			pos++
			return r, true, nil
		}}, nil
	case *object.StringIO:
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if x.Pos >= len(x.V) {
				return nil, false, nil
			}
			start := x.Pos
			end := start
			for end < len(x.V) && x.V[end] != '\n' {
				end++
			}
			if end < len(x.V) {
				end++
			}
			x.Pos = end
			return &object.Str{V: string(x.V[start:end])}, true, nil
		}}, nil
	case *object.BytesIO:
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if x.Pos >= len(x.V) {
				return nil, false, nil
			}
			end := x.Pos
			for end < len(x.V) && x.V[end] != '\n' {
				end++
			}
			if end < len(x.V) {
				end++
			}
			line := append([]byte(nil), x.V[x.Pos:end]...)
			x.Pos = end
			return &object.Bytes{V: line}, true, nil
		}}, nil
	case *object.File:
		return i.fileIter(x), nil
	case *object.Class:
		if x.EnumData != nil {
			idx := 0
			members := x.EnumData.Members
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(members) {
					return nil, false, nil
				}
				r := members[idx]
				idx++
				return r, true, nil
			}}, nil
		}
	}
	return nil, object.Errorf(i.typeErr, "'%s' object is not iterable", object.TypeName(v))
}

// iterate exhausts an iterable into a slice.
func iterate(i *Interp, v object.Object) ([]object.Object, error) {
	it, err := i.getIter(v)
	if err != nil {
		return nil, err
	}
	var out []object.Object
	for {
		x, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		out = append(out, x)
	}
}

// --- format ---

// instanceFormat tries o.__format__(spec) for a user Instance. Returns
// (string, handled, err).
func (i *Interp) instanceFormat(o object.Object, spec string) (string, bool, error) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return "", false, nil
	}
	r, ok, err := i.callInstanceDunder(inst, "__format__", &object.Str{V: spec})
	if !ok {
		return "", false, nil
	}
	if err != nil {
		return "", true, err
	}
	s, ok := r.(*object.Str)
	if !ok {
		return "", true, object.Errorf(i.typeErr, "__format__ must return a str")
	}
	return s.V, true, nil
}

func formatValue(v object.Object, spec string) (string, error) {
	if spec == "" {
		return object.Str_(v), nil
	}
	// Parse a minimal subset: [fill][align][sign][#][0][width][,][.precision][type]
	s := spec
	fill := byte(' ')
	align := byte(0)
	sign := byte(0)
	width := 0
	precision := -1
	typ := byte(0)

	if len(s) >= 2 && (s[1] == '<' || s[1] == '>' || s[1] == '^' || s[1] == '=') {
		fill = s[0]
		align = s[1]
		s = s[2:]
	} else if len(s) >= 1 && (s[0] == '<' || s[0] == '>' || s[0] == '^' || s[0] == '=') {
		align = s[0]
		s = s[1:]
	}
	if len(s) > 0 && (s[0] == '+' || s[0] == '-' || s[0] == ' ') {
		sign = s[0]
		s = s[1:]
	}
	altForm := false
	if len(s) > 0 && s[0] == '#' {
		altForm = true
		s = s[1:]
	}
	if len(s) > 0 && s[0] == '0' {
		if align == 0 {
			align = '='
			fill = '0'
		}
		s = s[1:]
	}
	// width
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		width = width*10 + int(s[0]-'0')
		s = s[1:]
	}
	group := byte(0)
	if len(s) > 0 && (s[0] == ',' || s[0] == '_') {
		group = s[0]
		s = s[1:]
	}
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		precision = 0
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			precision = precision*10 + int(s[0]-'0')
			s = s[1:]
		}
	}
	if len(s) > 0 {
		typ = s[0]
	}

	var body string
	zeroPadHandled := false
	switch x := v.(type) {
	case *object.Int:
		it := typ
		if it == 'n' {
			it = 'd'
		}
		absV := new(big.Int).Abs(&x.V)
		var prefix, digits string
		switch it {
		case 'b':
			digits = absV.Text(2)
			if altForm {
				prefix = "0b"
			}
		case 'o':
			digits = absV.Text(8)
			if altForm {
				prefix = "0o"
			}
		case 'x':
			digits = absV.Text(16)
			if altForm {
				prefix = "0x"
			}
		case 'X':
			digits = strings.ToUpper(absV.Text(16))
			if altForm {
				prefix = "0X"
			}
		case 'c':
			cp := absV.Int64()
			body = string(rune(cp))
		case 'd', 0:
			digits = absV.String()
		case 'f', 'F':
			fv, _ := new(big.Float).SetInt(&x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			digits = strconv.FormatFloat(fv, 'f', p, 64)
		case 'e', 'E':
			fv, _ := new(big.Float).SetInt(&x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			digits = strconv.FormatFloat(fv, byte(it), p, 64)
		default:
			digits = absV.String()
		}
		if it == 'c' {
			// no sign/prefix/group
		} else {
			var signHead string
			switch {
			case x.V.Sign() < 0:
				signHead = "-"
			case sign == '+':
				signHead = "+"
			case sign == ' ':
				signHead = " "
			}
			head := signHead + prefix
			grouped := group != 0 && (it == 0 || it == 'd' || it == 'b' || it == 'o' || it == 'x' || it == 'X')
			stride := 3
			if it == 'b' || it == 'o' || it == 'x' || it == 'X' {
				stride = 4
			}
			if align == '=' && fill == '0' && width > len(head)+len(digits) {
				target := width - len(head)
				if grouped {
					d := len(digits)
					for d+(d-1)/stride < target {
						d++
					}
					if d > len(digits) {
						digits = strings.Repeat("0", d-len(digits)) + digits
					}
					digits = addGroups(digits, group, stride)
					if len(digits) < target {
						digits = strings.Repeat("0", target-len(digits)) + digits
					}
				} else {
					digits = strings.Repeat("0", target-len(digits)) + digits
				}
				body = head + digits
				zeroPadHandled = true
			} else {
				if grouped {
					digits = addGroups(digits, group, stride)
				}
				body = head + digits
			}
		}
	case *object.Float:
		p := precision
		ft := typ
		if ft == 0 {
			ft = 'g'
		}
		if ft == 'n' {
			ft = 'g'
		}
		pct := false
		if ft == '%' {
			ft = 'f'
			pct = true
		}
		if p < 0 {
			p = 6
		}
		fv := x.V
		if pct {
			fv *= 100
		}
		body = strconv.FormatFloat(fv, ft, p, 64)
		if altForm && (ft == 'g' || ft == 'G') {
			body = padGTrailingZeros(body, p)
		}
		if pct {
			body += "%"
		}
		signV := 0
		if x.V > 0 {
			signV = 1
		} else if x.V < 0 {
			signV = -1
		}
		if group != 0 {
			body = applyFloatGroup(body, group)
		}
		body = applySign(body, sign, signV)
	case *object.Str:
		body = x.V
		if precision >= 0 && precision < len(body) {
			body = body[:precision]
		}
	default:
		body = object.Str_(v)
	}

	if zeroPadHandled {
		return body, nil
	}
	if width > len(body) {
		pad := width - len(body)
		padStr := strings.Repeat(string(fill), pad)
		switch align {
		case '<':
			body = body + padStr
		case '>', 0:
			body = padStr + body
		case '^':
			l := pad / 2
			r := pad - l
			body = strings.Repeat(string(fill), l) + body + strings.Repeat(string(fill), r)
		case '=':
			if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
				body = string(body[0]) + padStr + body[1:]
			} else {
				body = padStr + body
			}
		}
	}
	return body, nil
}

// addGroups inserts sep every `stride` digits from the right of the digit
// portion of s. s may have an optional leading '-' sign; any non-digit prefix
// (e.g. 0x) should not be passed in.
func addGroups(s string, sep byte, stride int) string {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	n := len(s)
	if n <= stride {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	first := n % stride
	if first == 0 {
		first = stride
	}
	b.WriteString(s[:first])
	for j := first; j < n; j += stride {
		b.WriteByte(sep)
		b.WriteString(s[j : j+stride])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// padGTrailingZeros expands a 'g'/'G'-formatted float to carry its full
// significant-digit precision, matching CPython's alt-form `#g` behavior.
// The mantissa is padded with trailing zeros so it has exactly `prec`
// significant digits, and a trailing "." is added if no decimal exists.
func padGTrailingZeros(s string, prec int) string {
	if prec <= 0 {
		prec = 1
	}
	sign := ""
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		sign = string(s[0])
		s = s[1:]
	}
	mantissa, exp := s, ""
	if k := strings.IndexAny(s, "eE"); k >= 0 {
		mantissa, exp = s[:k], s[k:]
	}
	sig := 0
	seenNonZero := false
	for k := 0; k < len(mantissa); k++ {
		c := mantissa[k]
		if c == '.' {
			continue
		}
		if c >= '1' && c <= '9' {
			seenNonZero = true
			sig++
		} else if c == '0' && seenNonZero {
			sig++
		}
	}
	need := prec - sig
	if need < 0 {
		need = 0
	}
	if !strings.Contains(mantissa, ".") && need > 0 {
		mantissa += "."
	} else if !strings.Contains(mantissa, ".") && exp == "" {
		// alt form requires a trailing decimal point even if full precision.
		mantissa += "."
	}
	if need > 0 {
		mantissa += strings.Repeat("0", need)
	}
	return sign + mantissa + exp
}

// applyFloatGroup inserts group separators into the integer portion of a
// formatted float body, preserving any fractional/exponent tail and the
// optional leading sign.
func applyFloatGroup(body string, sep byte) string {
	neg := false
	s := body
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	end := len(s)
	for k := 0; k < len(s); k++ {
		if s[k] == '.' || s[k] == 'e' || s[k] == 'E' {
			end = k
			break
		}
	}
	intPart := s[:end]
	tail := s[end:]
	out := addGroups(intPart, sep, 3) + tail
	if neg {
		return "-" + out
	}
	return out
}

func applySign(body string, sign byte, signV int) string {
	if len(body) > 0 && body[0] == '-' {
		return body
	}
	if signV < 0 {
		return "-" + body
	}
	switch sign {
	case '+':
		return "+" + body
	case ' ':
		return " " + body
	}
	return body
}

// --- helpers for toException ---

func (i *Interp) toException(v object.Object) error {
	if e, ok := v.(*object.Exception); ok {
		return e
	}
	if cls, ok := v.(*object.Class); ok {
		return object.NewException(cls, "")
	}
	return object.Errorf(i.typeErr, "exceptions must derive from BaseException (got %s)", object.TypeName(v))
}

// Debug (unused).
var _ = fmt.Sprintf
