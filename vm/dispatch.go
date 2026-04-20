package vm

import (
	"math/big"
	"strings"

	"github.com/tamnd/goipy/object"
	"github.com/tamnd/goipy/op"
)

// dispatch runs a frame's bytecode until RETURN_VALUE or an unhandled
// exception.
func (i *Interp) dispatch(f *Frame) (object.Object, error) {
	code := f.Code.Bytecode
	excTable := decodeExceptionTable(f.Code.ExceptionTable)

	// EXTENDED_ARG carry.
	extArg := uint32(0)

	for {
		if f.IP >= len(code) {
			return object.None, nil
		}
		opcode := code[f.IP]
		oparg := uint32(code[f.IP+1]) | extArg
		extArg = 0

		// Advance IP past opcode + immediate arg; cache adjustment handled
		// per-branch OR at the end.
		startIP := f.IP
		f.IP += 2
		cache := int(op.Cache[opcode])
		if cache > 0 {
			f.IP += 2 * cache
		}

		var result object.Object
		var err error

		switch opcode {
		// --- no-ops & meta ---
		case op.NOP, op.RESUME, op.RESUME_CHECK, op.NOT_TAKEN,
			op.INSTRUMENTED_LINE, op.JUMP_BACKWARD_NO_INTERRUPT, op.ENTER_EXECUTOR:
			// fallthrough
			if opcode == op.JUMP_BACKWARD_NO_INTERRUPT {
				f.IP -= 2 * int(oparg)
			}
			continue
		case op.EXTENDED_ARG:
			extArg = oparg << 8
			continue
		case op.CACHE:
			// Raw cache entries should never execute, but skip gracefully.
			continue

		// --- stack manipulation ---
		case op.POP_TOP:
			f.pop()
		case op.PUSH_NULL:
			f.push(nil)
		case op.COPY:
			// COPY oparg: push a copy of stack[-oparg]
			f.push(f.peek(int(oparg) - 1))
		case op.SWAP:
			// SWAP oparg: swap TOS with stack[-oparg]
			i1 := f.SP - 1
			i2 := f.SP - int(oparg)
			f.Stack[i1], f.Stack[i2] = f.Stack[i2], f.Stack[i1]

		// --- constants ---
		case op.LOAD_SMALL_INT:
			f.push(object.NewInt(int64(oparg)))
		case op.LOAD_CONST, op.LOAD_CONST_IMMORTAL, op.LOAD_CONST_MORTAL:
			f.push(f.Code.Consts[oparg])
		case op.LOAD_COMMON_CONSTANT:
			// Map indices to their classes lazily.
			switch oparg {
			case 0:
				f.push(i.assertErr)
			case 1:
				f.push(i.notImpl)
			case 2:
				tup, _ := i.Builtins.GetStr("tuple")
				f.push(tup)
			case 3:
				allF, _ := i.Builtins.GetStr("all")
				f.push(allF)
			case 4:
				anyF, _ := i.Builtins.GetStr("any")
				f.push(anyF)
			default:
				return nil, object.Errorf(i.runtimeErr, "unknown common constant %d", oparg)
			}

		// --- names (module-level) ---
		case op.LOAD_NAME:
			name := f.Code.Names[oparg]
			if v, ok := f.Locals.GetStr(name); ok {
				f.push(v)
				break
			}
			if f.Locals != f.Globals {
				if v, ok := f.Globals.GetStr(name); ok {
					f.push(v)
					break
				}
			}
			if v, ok := f.Builtins.GetStr(name); ok {
				f.push(v)
				break
			}
			return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)
		case op.STORE_NAME:
			name := f.Code.Names[oparg]
			f.Locals.SetStr(name, f.pop())
		case op.DELETE_NAME:
			name := f.Code.Names[oparg]
			if _, err := f.Locals.Delete(&object.Str{V: name}); err != nil {
				return nil, err
			}

		// --- globals ---
		case op.LOAD_GLOBAL, op.LOAD_GLOBAL_BUILTIN, op.LOAD_GLOBAL_MODULE:
			name := f.Code.Names[oparg>>1]
			pushNull := oparg&1 != 0
			var v object.Object
			var ok bool
			if v, ok = f.Globals.GetStr(name); !ok {
				if v, ok = f.Builtins.GetStr(name); !ok {
					return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)
				}
			}
			f.push(v)
			if pushNull {
				f.push(nil)
			}
		case op.STORE_GLOBAL:
			f.Globals.SetStr(f.Code.Names[oparg], f.pop())
		case op.DELETE_GLOBAL:
			if _, err := f.Globals.Delete(&object.Str{V: f.Code.Names[oparg]}); err != nil {
				return nil, err
			}

		// --- fast locals ---
		case op.LOAD_FAST, op.LOAD_FAST_BORROW, op.LOAD_FAST_CHECK:
			v := f.Fast[oparg]
			if v == nil {
				return nil, object.Errorf(i.nameErr,
					"local variable '%s' referenced before assignment", f.Code.LocalsPlusNames[oparg])
			}
			f.push(v)
		case op.LOAD_FAST_AND_CLEAR:
			v := f.Fast[oparg]
			f.Fast[oparg] = nil
			if v == nil {
				f.push(nil)
			} else {
				f.push(v)
			}
		case op.LOAD_FAST_LOAD_FAST, op.LOAD_FAST_BORROW_LOAD_FAST_BORROW:
			// oparg packs two 4-bit indices: hi = oparg>>4, lo = oparg&0xf
			a := f.Fast[oparg>>4]
			b := f.Fast[oparg&0xf]
			if a == nil {
				return nil, object.Errorf(i.nameErr,
					"local variable '%s' referenced before assignment", f.Code.LocalsPlusNames[oparg>>4])
			}
			if b == nil {
				return nil, object.Errorf(i.nameErr,
					"local variable '%s' referenced before assignment", f.Code.LocalsPlusNames[oparg&0xf])
			}
			f.push(a)
			f.push(b)
		case op.STORE_FAST:
			f.Fast[oparg] = f.pop()
		case op.STORE_FAST_LOAD_FAST:
			f.Fast[oparg>>4] = f.pop()
			v := f.Fast[oparg&0xf]
			if v == nil {
				return nil, object.Errorf(i.nameErr, "local variable referenced before assignment")
			}
			f.push(v)
		case op.STORE_FAST_STORE_FAST:
			f.Fast[oparg>>4] = f.pop()
			f.Fast[oparg&0xf] = f.pop()
		case op.DELETE_FAST:
			if f.Fast[oparg] == nil {
				return nil, object.Errorf(i.nameErr, "local variable referenced before assignment")
			}
			f.Fast[oparg] = nil

		// --- cells / closure ---
		case op.MAKE_CELL:
			c := &object.Cell{}
			if v := f.Fast[oparg]; v != nil {
				c.V = v
				c.Set = true
			}
			f.Fast[oparg] = c
		case op.LOAD_DEREF:
			c, ok := f.Fast[oparg].(*object.Cell)
			if !ok || c == nil || !c.Set {
				return nil, object.Errorf(i.nameErr, "free variable referenced before assignment")
			}
			f.push(c.V)
		case op.STORE_DEREF:
			c, ok := f.Fast[oparg].(*object.Cell)
			if !ok || c == nil {
				c = &object.Cell{}
				f.Fast[oparg] = c
			}
			c.V = f.pop()
			c.Set = true
		case op.DELETE_DEREF:
			if c, ok := f.Fast[oparg].(*object.Cell); ok && c != nil {
				c.Set = false
				c.V = nil
			}
		case op.COPY_FREE_VARS:
			// no-op — free cells were already copied when frame was created
			_ = oparg
		case op.LOAD_FROM_DICT_OR_DEREF:
			name := f.Code.LocalsPlusNames[oparg]
			dict := f.pop()
			if d, ok := dict.(*object.Dict); ok {
				if v, ok := d.GetStr(name); ok {
					f.push(v)
					break
				}
			}
			c, ok := f.Fast[oparg].(*object.Cell)
			if !ok || c == nil || !c.Set {
				return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)
			}
			f.push(c.V)
		case op.LOAD_FROM_DICT_OR_GLOBALS:
			name := f.Code.Names[oparg]
			dict := f.pop()
			if d, ok := dict.(*object.Dict); ok {
				if v, ok := d.GetStr(name); ok {
					f.push(v)
					break
				}
			}
			if v, ok := f.Globals.GetStr(name); ok {
				f.push(v)
				break
			}
			if v, ok := f.Builtins.GetStr(name); ok {
				f.push(v)
				break
			}
			return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)

		// --- attribute access ---
		case op.LOAD_ATTR:
			pushSelf := oparg&1 != 0
			name := f.Code.Names[oparg>>1]
			obj := f.pop()
			val, err := i.getAttr(obj, name)
			if err != nil {
				return nil, err
			}
			if pushSelf {
				f.push(val)
				f.push(nil)
			} else {
				f.push(val)
			}
		case op.STORE_ATTR:
			name := f.Code.Names[oparg]
			obj := f.pop()
			val := f.pop()
			if err := i.setAttr(obj, name, val); err != nil {
				return nil, err
			}
		case op.DELETE_ATTR:
			name := f.Code.Names[oparg]
			obj := f.pop()
			if err := i.delAttr(obj, name); err != nil {
				return nil, err
			}

		// --- arithmetic ---
		case op.BINARY_OP:
			b := f.pop()
			a := f.pop()
			result, err = i.binaryOp(a, b, oparg)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_ADD_INT, op.BINARY_OP_ADD_FLOAT:
			b := f.pop()
			a := f.pop()
			result, err = i.add(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_ADD_UNICODE:
			b := f.pop()
			a := f.pop()
			result, err = i.add(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_SUBTRACT_INT, op.BINARY_OP_SUBTRACT_FLOAT:
			b := f.pop()
			a := f.pop()
			result, err = i.sub(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_MULTIPLY_INT, op.BINARY_OP_MULTIPLY_FLOAT:
			b := f.pop()
			a := f.pop()
			result, err = i.mul(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_SUBSCR_DICT, op.BINARY_OP_SUBSCR_GETITEM,
			op.BINARY_OP_SUBSCR_LIST_INT, op.BINARY_OP_SUBSCR_LIST_SLICE,
			op.BINARY_OP_SUBSCR_STR_INT, op.BINARY_OP_SUBSCR_TUPLE_INT:
			k := f.pop()
			c := f.pop()
			result, err = i.getitem(c, k)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_SLICE:
			stop := f.pop()
			start := f.pop()
			c := f.pop()
			result, err = i.getitem(c, &object.Slice{Start: start, Stop: stop, Step: object.None})
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.STORE_SUBSCR, op.STORE_SUBSCR_DICT, op.STORE_SUBSCR_LIST_INT:
			k := f.pop()
			c := f.pop()
			v := f.pop()
			if err = i.setitem(c, k, v); err != nil { goto handleErr }
		case op.STORE_SLICE:
			stop := f.pop()
			start := f.pop()
			c := f.pop()
			v := f.pop()
			if err = i.setitem(c, &object.Slice{Start: start, Stop: stop, Step: object.None}, v); err != nil { goto handleErr }
		case op.DELETE_SUBSCR:
			k := f.pop()
			c := f.pop()
			if err = i.delitem(c, k); err != nil { goto handleErr }

		// --- unary ---
		case op.UNARY_NEGATIVE:
			v := f.pop()
			result, err = i.unaryNeg(v)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.UNARY_NOT:
			v := f.pop()
			f.push(object.BoolOf(!object.Truthy(v)))
		case op.UNARY_INVERT:
			v := f.pop()
			bi, ok := toBigInt(v)
			if !ok {
				return nil, object.Errorf(i.typeErr, "bad operand for ~")
			}
			r := new(big.Int).Not(bi)
			f.push(&object.Int{V: r})
		case op.TO_BOOL, op.TO_BOOL_ALWAYS_TRUE, op.TO_BOOL_BOOL,
			op.TO_BOOL_INT, op.TO_BOOL_LIST, op.TO_BOOL_NONE, op.TO_BOOL_STR:
			f.setTop(object.BoolOf(object.Truthy(f.top())))

		// --- comparisons ---
		case op.COMPARE_OP, op.COMPARE_OP_INT, op.COMPARE_OP_FLOAT, op.COMPARE_OP_STR:
			b := f.pop()
			a := f.pop()
			kind := int(oparg >> 5)
			result, err = i.compare(a, b, kind)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.IS_OP:
			b := f.pop()
			a := f.pop()
			same := a == b
			if _, ok := a.(*object.NoneType); ok {
				_, ok2 := b.(*object.NoneType)
				same = ok2
			}
			if oparg == 1 {
				same = !same
			}
			f.push(object.BoolOf(same))
		case op.CONTAINS_OP, op.CONTAINS_OP_DICT, op.CONTAINS_OP_SET:
			b := f.pop()
			a := f.pop()
			result, err = i.containsOp(b, a, oparg == 1)
			if err != nil {
				goto handleErr
			}
			f.push(result)

		// --- builders ---
		case op.BUILD_TUPLE:
			n := int(oparg)
			items := make([]object.Object, n)
			copy(items, f.Stack[f.SP-n:f.SP])
			f.SP -= n
			f.push(&object.Tuple{V: items})
		case op.BUILD_LIST:
			n := int(oparg)
			items := make([]object.Object, n)
			copy(items, f.Stack[f.SP-n:f.SP])
			f.SP -= n
			f.push(&object.List{V: items})
		case op.BUILD_SET:
			n := int(oparg)
			s := object.NewSet()
			for k := f.SP - n; k < f.SP; k++ {
				if err = s.Add(f.Stack[k]); err != nil { goto handleErr }
			}
			f.SP -= n
			f.push(s)
		case op.BUILD_MAP:
			n := int(oparg)
			d := object.NewDict()
			base := f.SP - 2*n
			for k := 0; k < n; k++ {
				if err = d.Set(f.Stack[base+2*k], f.Stack[base+2*k+1]); err != nil { goto handleErr }
			}
			f.SP = base
			f.push(d)
		case op.BUILD_SLICE:
			n := int(oparg)
			if n == 3 {
				step := f.pop()
				stop := f.pop()
				start := f.pop()
				f.push(&object.Slice{Start: start, Stop: stop, Step: step})
			} else {
				stop := f.pop()
				start := f.pop()
				f.push(&object.Slice{Start: start, Stop: stop, Step: object.None})
			}
		case op.BUILD_STRING:
			n := int(oparg)
			var sb strings.Builder
			for k := f.SP - n; k < f.SP; k++ {
				s, ok := f.Stack[k].(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "BUILD_STRING expects str")
				}
				sb.WriteString(s.V)
			}
			f.SP -= n
			f.push(&object.Str{V: sb.String()})
		case op.LIST_APPEND:
			v := f.pop()
			l := f.peek(int(oparg) - 1).(*object.List)
			l.V = append(l.V, v)
		case op.LIST_EXTEND:
			it := f.pop()
			l := f.peek(int(oparg) - 1).(*object.List)
			var items []object.Object
			items, err = iterate(i, it)
			if err != nil {
				goto handleErr
			}
			l.V = append(l.V, items...)
		case op.SET_ADD:
			v := f.pop()
			s := f.peek(int(oparg) - 1).(*object.Set)
			if err = s.Add(v); err != nil { goto handleErr }
		case op.SET_UPDATE:
			it := f.pop()
			s := f.peek(int(oparg) - 1).(*object.Set)
			var items []object.Object
			items, err = iterate(i, it)
			if err != nil {
				goto handleErr
			}
			for _, x := range items {
				if err = s.Add(x); err != nil { goto handleErr }
			}
		case op.MAP_ADD:
			v := f.pop()
			k := f.pop()
			d := f.peek(int(oparg) - 1).(*object.Dict)
			if err = d.Set(k, v); err != nil { goto handleErr }
		case op.DICT_UPDATE, op.DICT_MERGE:
			src := f.pop()
			d := f.peek(int(oparg) - 1).(*object.Dict)
			sd, ok := src.(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "expected dict, got %s", object.TypeName(src))
			}
			ks, vs := sd.Items()
			for k, key := range ks {
				if err = d.Set(key, vs[k]); err != nil { goto handleErr }
			}
		case op.GET_LEN:
			v := f.top()
			var n int64
			n, err = i.length(v)
			if err != nil {
				goto handleErr
			}
			f.push(object.NewInt(n))

		// --- match/case (PEP 634) ---
		case op.MATCH_MAPPING:
			if _, ok := f.top().(*object.Dict); ok {
				f.push(object.True)
			} else {
				f.push(object.False)
			}
		case op.MATCH_SEQUENCE:
			switch f.top().(type) {
			case *object.List, *object.Tuple, *object.Range:
				f.push(object.True)
			default:
				f.push(object.False)
			}
		case op.MATCH_KEYS:
			keysObj := f.top()
			subject := f.peek(1)
			d, dok := subject.(*object.Dict)
			keysT, kok := keysObj.(*object.Tuple)
			if !dok || !kok {
				f.push(object.None)
				break
			}
			values := make([]object.Object, 0, len(keysT.V))
			miss := false
			for _, k := range keysT.V {
				v, found, gerr := d.Get(k)
				if gerr != nil {
					err = gerr
					goto handleErr
				}
				if !found {
					miss = true
					break
				}
				values = append(values, v)
			}
			if miss {
				f.push(object.None)
			} else {
				f.push(&object.Tuple{V: values})
			}
		case op.MATCH_CLASS:
			kwnames := f.pop()
			cls := f.pop()
			subject := f.pop()
			var attrs object.Object
			attrs, err = i.matchClass(subject, cls, kwnames, int(oparg))
			if err != nil {
				goto handleErr
			}
			f.push(attrs)

		// --- iteration ---
		case op.GET_ITER:
			v := f.pop()
			var it *object.Iter
			it, err = i.getIter(v)
			if err != nil {
				goto handleErr
			}
			f.push(it)
		case op.FOR_ITER, op.FOR_ITER_LIST, op.FOR_ITER_TUPLE, op.FOR_ITER_RANGE:
			it, ok := f.top().(*object.Iter)
			if !ok {
				conv, cerr := i.getIter(f.top())
				if cerr != nil {
					err = cerr
					goto handleErr
				}
				f.setTop(conv)
				it = conv
			}
			v, nok, ierr := it.Next()
			if ierr != nil {
				err = ierr; goto handleErr
			}
			if !nok {
				f.IP += int(oparg) * 2
			} else {
				f.push(v)
			}
		case op.END_FOR:
			// Sibling of POP_ITER; in 3.14 the compiler emits END_FOR after
			// the loop body completes. It's a no-op at the VM level.
		case op.POP_ITER:
			f.pop()

		// --- jumps ---
		case op.JUMP_FORWARD:
			f.IP = startIP + 2*(1+int(oparg))
		case op.JUMP_BACKWARD, op.JUMP_BACKWARD_JIT, op.JUMP_BACKWARD_NO_JIT:
			f.IP -= 2 * int(oparg)
		case op.POP_JUMP_IF_TRUE:
			v := f.pop()
			if object.Truthy(v) {
				f.IP = startIP + 2*(2+int(oparg))
			}
		case op.POP_JUMP_IF_FALSE:
			v := f.pop()
			if !object.Truthy(v) {
				f.IP = startIP + 2*(2+int(oparg))
			}
		case op.POP_JUMP_IF_NONE:
			v := f.pop()
			if _, ok := v.(*object.NoneType); ok {
				f.IP = startIP + 2*(2+int(oparg))
			}
		case op.POP_JUMP_IF_NOT_NONE:
			v := f.pop()
			if _, ok := v.(*object.NoneType); !ok {
				f.IP = startIP + 2*(2+int(oparg))
			}

		// --- unpacking ---
		case op.UNPACK_SEQUENCE, op.UNPACK_SEQUENCE_LIST,
			op.UNPACK_SEQUENCE_TUPLE, op.UNPACK_SEQUENCE_TWO_TUPLE:
			seq := f.pop()
			var items []object.Object
			items, err = iterate(i, seq)
			if err != nil {
				goto handleErr
			}
			if len(items) != int(oparg) {
				return nil, object.Errorf(i.valueErr,
					"expected %d values, got %d", oparg, len(items))
			}
			for k := len(items) - 1; k >= 0; k-- {
				f.push(items[k])
			}
		case op.UNPACK_EX:
			// oparg low byte = before, high byte = after
			before := int(oparg & 0xff)
			after := int(oparg >> 8)
			seq := f.pop()
			var items []object.Object
			items, err = iterate(i, seq)
			if err != nil {
				goto handleErr
			}
			if len(items) < before+after {
				return nil, object.Errorf(i.valueErr,
					"not enough values to unpack")
			}
			mid := items[before : len(items)-after]
			postItems := items[len(items)-after:]
			// Push in reverse so they pop in order.
			for k := len(postItems) - 1; k >= 0; k-- {
				f.push(postItems[k])
			}
			midList := make([]object.Object, len(mid))
			copy(midList, mid)
			f.push(&object.List{V: midList})
			for k := before - 1; k >= 0; k-- {
				f.push(items[k])
			}

		// --- call & return ---
		case op.CALL, op.CALL_PY_EXACT_ARGS, op.CALL_PY_GENERAL,
			op.CALL_BOUND_METHOD_EXACT_ARGS, op.CALL_BOUND_METHOD_GENERAL,
			op.CALL_BUILTIN_CLASS, op.CALL_BUILTIN_FAST,
			op.CALL_BUILTIN_FAST_WITH_KEYWORDS, op.CALL_BUILTIN_O,
			op.CALL_ISINSTANCE, op.CALL_LEN, op.CALL_LIST_APPEND,
			op.CALL_METHOD_DESCRIPTOR_FAST, op.CALL_METHOD_DESCRIPTOR_FAST_WITH_KEYWORDS,
			op.CALL_METHOD_DESCRIPTOR_NOARGS, op.CALL_METHOD_DESCRIPTOR_O,
			op.CALL_NON_PY_GENERAL, op.CALL_STR_1, op.CALL_TUPLE_1, op.CALL_TYPE_1:
			n := int(oparg)
			base := f.SP - n - 2
			callable := f.Stack[base]
			selfOrNull := f.Stack[base+1]
			args := f.Stack[base+2 : f.SP]
			var call []object.Object
			if selfOrNull != nil {
				call = append([]object.Object{selfOrNull}, args...)
			} else {
				call = append([]object.Object{}, args...)
			}
			f.SP = base
			var r object.Object
			r, err = i.callObject(callable, call, nil)
			if err != nil {
				goto handleErr
			}
			f.push(r)
		case op.CALL_KW, op.CALL_KW_PY, op.CALL_KW_BOUND_METHOD, op.CALL_KW_NON_PY:
			n := int(oparg)
			kwnames := f.pop().(*object.Tuple)
			base := f.SP - n - 2
			callable := f.Stack[base]
			selfOrNull := f.Stack[base+1]
			allArgs := f.Stack[base+2 : f.SP]
			posCount := len(allArgs) - len(kwnames.V)
			pos := allArgs[:posCount]
			kwVals := allArgs[posCount:]
			var call []object.Object
			if selfOrNull != nil {
				call = append([]object.Object{selfOrNull}, pos...)
			} else {
				call = append([]object.Object{}, pos...)
			}
			kw := object.NewDict()
			for k, name := range kwnames.V {
				kw.SetStr(name.(*object.Str).V, kwVals[k])
			}
			f.SP = base
			var r object.Object
			r, err = i.callObject(callable, call, kw)
			if err != nil {
				goto handleErr
			}
			f.push(r)
		case op.CALL_FUNCTION_EX:
			// 3.14 layout: [callable, NULL, args, kwargs_or_NULL]
			_ = oparg
			top := f.pop()
			var kw *object.Dict
			if d, ok := top.(*object.Dict); ok {
				kw = d
			}
			argsObj := f.pop()
			_ = f.pop() // NULL slot beneath callable
			callable := f.pop()
			var args []object.Object
			switch a := argsObj.(type) {
			case *object.Tuple:
				args = a.V
			case *object.List:
				args = a.V
			default:
				var list []object.Object
				list, err = iterate(i, argsObj)
				if err != nil {
					goto handleErr
				}
				args = list
			}
			var r object.Object
			r, err = i.callObject(callable, args, kw)
			if err != nil {
				goto handleErr
			}
			f.push(r)
		case op.CALL_INTRINSIC_1:
			v := f.pop()
			result, err = i.intrinsic1(int(oparg), v)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.CALL_INTRINSIC_2:
			b := f.pop()
			a := f.pop()
			result, err = i.intrinsic2(int(oparg), a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.RETURN_VALUE:
			return f.pop(), nil
		case op.RETURN_GENERATOR:
			// Called-as-generator path is intercepted in callFunction which
			// builds the Generator object directly. When the generator is
			// later resumed, RETURN_GENERATOR is re-executed at IP=0 — treat
			// it as a no-op: the POP_TOP that follows pops the `sent` value
			// the driver pushed before dispatching.
		case op.YIELD_VALUE:
			f.Yielded = f.pop()
			return nil, errYielded
		case op.SEND:
			v := f.pop()
			recv := f.top()
			var yielded object.Object
			var sendErr error
			switch r := recv.(type) {
			case *object.Generator:
				yielded, sendErr = i.resumeGenerator(r, v)
			case *object.Iter:
				if _, ok := v.(*object.NoneType); !ok {
					sendErr = object.Errorf(i.typeErr, "can't send non-None value to a non-generator iterator")
					break
				}
				val, ok, ierr := r.Next()
				if ierr != nil {
					sendErr = ierr
				} else if !ok {
					sendErr = object.Errorf(i.stopIter, "")
				} else {
					yielded = val
				}
			default:
				sendErr = object.Errorf(i.typeErr, "SEND: expected generator/iterator, got %s", object.TypeName(recv))
			}
			if sendErr != nil {
				if exc, ok := sendErr.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					// StopIteration: replace v slot (now top) with the value,
					// leaving receiver in place below, then jump by oparg.
					var stopVal object.Object = object.None
					if exc.Args != nil && len(exc.Args.V) > 0 {
						stopVal = exc.Args.V[0]
					}
					f.push(stopVal)
					f.IP = startIP + 2*(2+int(oparg))
					break
				}
				err = sendErr
				goto handleErr
			}
			f.push(yielded)
		case op.END_SEND:
			// Stack: [..., receiver, value] -> [..., value]
			val := f.pop()
			f.pop() // receiver
			f.push(val)
		case op.GET_YIELD_FROM_ITER:
			v := f.top()
			if _, ok := v.(*object.Generator); ok {
				break
			}
			if _, ok := v.(*object.Iter); ok {
				break
			}
			it, gerr := i.getIter(v)
			if gerr != nil {
				err = gerr
				goto handleErr
			}
			f.setTop(it)
		case op.CLEANUP_THROW:
			// TOS is the exception raised by generator.throw(); propagate.
			exc := f.top()
			if e, ok := exc.(*object.Exception); ok {
				err = e
			} else {
				err = object.Errorf(i.typeErr, "CLEANUP_THROW on non-exception")
			}
			goto handleErr

		// --- functions / classes ---
		case op.MAKE_FUNCTION:
			codeObj := f.pop().(*object.Code)
			fn := &object.Function{
				Code:     codeObj,
				Globals:  f.Globals,
				Name:     codeObj.Name,
				QualName: codeObj.QualName,
			}
			f.push(fn)
		case op.SET_FUNCTION_ATTRIBUTE:
			fn := f.pop().(*object.Function)
			val := f.pop()
			switch oparg {
			case 0x01:
				fn.Defaults = val.(*object.Tuple)
			case 0x02:
				fn.KwDefaults = val.(*object.Dict)
			case 0x04:
				fn.Annotations = val
			case 0x08:
				fn.Closure = val.(*object.Tuple)
			}
			f.push(fn)
		case op.LOAD_BUILD_CLASS:
			bc, _ := i.Builtins.GetStr("__build_class__")
			f.push(bc)
		case op.LOAD_LOCALS:
			f.push(f.Locals)

		// --- format / f-strings ---
		case op.CONVERT_VALUE:
			v := f.pop()
			var s string
			switch oparg {
			case 1: // str
				s = object.Str_(v)
			case 2: // repr
				s = object.Repr(v)
			case 3: // ascii
				s = object.Repr(v)
			default:
				f.push(v)
				continue
			}
			f.push(&object.Str{V: s})
		case op.FORMAT_SIMPLE:
			v := f.pop()
			if _, ok := v.(*object.Str); ok {
				f.push(v)
			} else {
				f.push(&object.Str{V: object.Str_(v)})
			}
		case op.FORMAT_WITH_SPEC:
			spec := f.pop().(*object.Str)
			v := f.pop()
			var s string
			s, err = formatValue(v, spec.V)
			if err != nil {
				goto handleErr
			}
			f.push(&object.Str{V: s})

		// --- import ---
		case op.IMPORT_NAME:
			name := f.Code.Names[oparg]
			fromlist := f.pop()
			levelObj := f.pop()
			level := 0
			if l, ok := levelObj.(*object.Int); ok {
				level = int(l.V.Int64())
			}
			var fl *object.Tuple
			if t, ok := fromlist.(*object.Tuple); ok {
				fl = t
			}
			mod, ierr := i.importName(name, f.Globals, fl, level)
			if ierr != nil {
				err = ierr
				goto handleErr
			}
			f.push(mod)
		case op.IMPORT_FROM:
			name := f.Code.Names[oparg]
			mod := f.top()
			m, ok := mod.(*object.Module)
			if !ok {
				err = object.Errorf(i.importErr, "IMPORT_FROM: not a module")
				goto handleErr
			}
			if v, ok := m.Dict.GetStr(name); ok {
				f.push(v)
				break
			}
			// Fall back to loading `m.name` as a submodule: `from pkg import sub`
			// where `sub` has not been touched yet.
			if isPackage(m) {
				if sub, lerr := i.loadModule(m.Name + "." + name); lerr == nil {
					f.push(sub)
					break
				}
			}
			err = object.Errorf(i.importErr, "cannot import name '%s' from '%s'", name, m.Name)
			goto handleErr

		// --- async ---
		case op.GET_AWAITABLE:
			// For our purposes, Generator/Coroutine/Iter are already
			// awaitable; pass them through. Anything else must expose
			// __await__() (not implemented).
			v := f.top()
			switch v.(type) {
			case *object.Generator, *object.Iter:
				// already awaitable
			default:
				err = object.Errorf(i.typeErr, "object %s can't be used in 'await' expression", object.TypeName(v))
				goto handleErr
			}

		// --- exceptions ---
		case op.RAISE_VARARGS:
			switch oparg {
			case 0:
				if f.ExcInfo != nil {
					err = f.ExcInfo
				} else {
					err = object.Errorf(i.runtimeErr, "No active exception to re-raise")
				}
			case 1:
				v := f.pop()
				err = i.toException(v)
			case 2:
				cause := f.pop()
				v := f.pop()
				e := i.toException(v)
				if _, isNone := cause.(*object.NoneType); !isNone {
					if cx, ok := cause.(*object.Exception); ok {
						e.(*object.Exception).Cause = cx
					}
				}
				err = e
			}
			goto handleErr
		case op.RERAISE:
			v := f.pop()
			if oparg > 0 {
				f.pop()
			}
			if e, ok := v.(*object.Exception); ok {
				err = e
			} else {
				err = i.toException(v)
			}
			goto handleErr
		case op.PUSH_EXC_INFO:
			cur := f.top()
			if f.ExcInfo != nil {
				f.setTop(f.ExcInfo)
			} else {
				f.setTop(object.None)
			}
			f.push(cur)
		case op.POP_EXCEPT:
			v := f.pop()
			if e, ok := v.(*object.Exception); ok {
				f.ExcInfo = e
			} else {
				f.ExcInfo = nil
			}
		case op.CHECK_EXC_MATCH:
			excType := f.pop()
			tos := f.top().(*object.Exception)
			cls, ok := excType.(*object.Class)
			if !ok {
				return nil, object.Errorf(i.typeErr, "except type must be a class")
			}
			f.push(object.BoolOf(object.IsSubclass(tos.Class, cls)))
		case op.LOAD_SUPER_ATTR, op.LOAD_SUPER_ATTR_ATTR, op.LOAD_SUPER_ATTR_METHOD:
			// oparg bit 0 = method (push self after), bit 1 = two-arg super
			methodBit := oparg&1 != 0
			name := f.Code.Names[oparg>>2]
			self := f.pop()
			startCls, _ := f.pop().(*object.Class)
			_ = f.pop() // super callable
			if startCls == nil {
				return nil, object.Errorf(i.typeErr, "super() expects class as 2nd arg")
			}
			inst, instOk := self.(*object.Instance)
			if !instOk {
				return nil, object.Errorf(i.typeErr, "super() requires an instance")
			}
			val, found := lookupAfter(inst.Class, startCls, name)
			if !found {
				return nil, object.Errorf(i.attrErr, "'super' object has no attribute '%s'", name)
			}
			if methodBit {
				f.push(val)
				f.push(self)
			} else {
				bound, berr := i.bindDescriptor(val, inst, inst.Class)
				if berr != nil {
					err = berr
					goto handleErr
				}
				f.push(bound)
			}
		case op.LOAD_SPECIAL:
			// oparg: 0=__enter__ 1=__exit__ 2=__aenter__ 3=__aexit__
			inst := f.pop()
			var name string
			switch oparg {
			case 0:
				name = "__enter__"
			case 1:
				name = "__exit__"
			case 2:
				name = "__aenter__"
			case 3:
				name = "__aexit__"
			default:
				return nil, object.Errorf(i.runtimeErr, "unknown LOAD_SPECIAL %d", oparg)
			}
			var method object.Object
			if inst2, ok := inst.(*object.Instance); ok {
				if v, ok := classLookup(inst2.Class, name); ok {
					method = v
				}
			}
			if method == nil {
				return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", object.TypeName(inst), name)
			}
			f.push(method)
			f.push(inst)
		case op.WITH_EXCEPT_START:
			// Stack after PUSH_EXC_INFO in a with-handler:
			// [..., exit_func, self_cm, lasti, prev_excinfo, exc]
			exc := f.top()
			self := f.peek(3)
			exitFn := f.peek(4)
			var excCls object.Object = object.None
			if e, ok := exc.(*object.Exception); ok {
				excCls = e.Class
			}
			args := []object.Object{self, excCls, exc, object.None}
			result, err = i.callObject(exitFn, args, nil)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.CHECK_EG_MATCH:
			// Exception groups not supported; treat like CHECK_EXC_MATCH.
			excType := f.pop()
			tos := f.top().(*object.Exception)
			cls, _ := excType.(*object.Class)
			if cls != nil && object.IsSubclass(tos.Class, cls) {
				f.push(tos)
				f.push(object.None)
			} else {
				f.push(object.None)
			}

		default:
			return nil, object.Errorf(i.notImpl,
				"opcode %s (%d) not implemented", op.Name(opcode), opcode)
		}
		continue

	handleErr:
		if err == nil {
			continue
		}
		e, eok := err.(*object.Exception)
		if !eok {
			return nil, err
		}
		handler := findHandler(excTable, startIP)
		if handler == nil {
			return nil, err
		}
		// Restore stack to handler.Depth
		for f.SP > handler.Depth {
			f.pop()
		}
		if handler.Lasti {
			f.push(object.NewInt(int64(startIP)))
		}
		f.push(e)
		f.IP = handler.Target
		continue
	}
}

// goto handleErrValue — helper inlined via direct assignment. Go doesn't allow
// labels as expressions, so we emulate: set err and goto handleErr.
// The compiler rewrites `goto handleErr` — that's not valid Go,
// so provide a helper instead.

// Not actually used as a Go label; see note above. Replaced with manual
// err-assignment + `goto handleErr` in the dispatcher.
