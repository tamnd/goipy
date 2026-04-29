package vm

import (
	"math"
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

		// Advance IP past opcode + immediate arg + any inline caches in one
		// shot. op.Cache[opcode] is 0 for most instructions, so the branch
		// that used to guard the addition was near-useless branch-predictor
		// churn. Just add unconditionally.
		startIP := f.IP
		f.IP += 2 + 2*int(op.Cache[opcode])

		// Per-frame line trace: fires on source-line transitions only.
		// Hot-loop guard — most frames have no tracer attached.
		if f.LocalTrace != nil && !i.inTrace {
			line := f.Code.LineForOffset(startIP)
			if line != 0 && line != f.LastLine {
				f.LastLine = line
				i.fireLineEvent(f)
			}
		}

		var result object.Object
		var err error

		// Pending exception injection for generator.throw().
		if f.PendingThrow != nil {
			err = f.PendingThrow
			f.PendingThrow = nil
			goto handleErr
		}

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
			if !ok || c == nil {
				return nil, object.Errorf(i.nameErr, "free variable referenced before assignment")
			}
			v, set := c.Load()
			if !set {
				return nil, object.Errorf(i.nameErr, "free variable referenced before assignment")
			}
			f.push(v)
		case op.STORE_DEREF:
			c, ok := f.Fast[oparg].(*object.Cell)
			if !ok || c == nil {
				c = &object.Cell{}
				f.Fast[oparg] = c
			}
			c.Store(f.pop())
		case op.DELETE_DEREF:
			if c, ok := f.Fast[oparg].(*object.Cell); ok && c != nil {
				c.Unset()
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
			if !ok || c == nil {
				return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)
			}
			v, set := c.Load()
			if !set {
				return nil, object.Errorf(i.nameErr, "name '%s' is not defined", name)
			}
			f.push(v)
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
		case op.LOAD_ATTR,
			op.LOAD_ATTR_CLASS, op.LOAD_ATTR_CLASS_WITH_METACLASS_CHECK,
			op.LOAD_ATTR_GETATTRIBUTE_OVERRIDDEN, op.LOAD_ATTR_INSTANCE_VALUE,
			op.LOAD_ATTR_METHOD_LAZY_DICT, op.LOAD_ATTR_METHOD_NO_DICT,
			op.LOAD_ATTR_METHOD_WITH_VALUES, op.LOAD_ATTR_MODULE,
			op.LOAD_ATTR_NONDESCRIPTOR_NO_DICT, op.LOAD_ATTR_NONDESCRIPTOR_WITH_VALUES,
			op.LOAD_ATTR_PROPERTY, op.LOAD_ATTR_SLOT, op.LOAD_ATTR_WITH_HINT:
			pushSelf := oparg&1 != 0
			name := f.Code.Names[oparg>>1]
			obj := f.pop()
			// Inline cache: for instances, specialize on the class identity
			// and the kind of attribute (inst-dict hit vs. unbound method on
			// class). This avoids the full getAttr type-switch + classLookup
			// walk on every dispatch.
			if inst, ok := obj.(*object.Instance); ok {
				// Read a snapshot of the cache entry under lock.
				f.Code.Mu.Lock()
				if f.Code.AttrCache == nil {
					f.Code.AttrCache = make([]object.AttrCacheEntry, len(f.Code.Bytecode))
				}
				snap := f.Code.AttrCache[startIP]
				f.Code.Mu.Unlock()

				var hitVal object.Object
				cacheHit := false
				if snap.Cls == inst.Class && snap.Cls != nil {
					switch snap.Kind {
					case object.AttrCacheInstDict:
						if v, ok := inst.Dict.GetStr(name); ok {
							hitVal = v
							cacheHit = true
						}
					case object.AttrCacheClassMethod:
						if v, ok := inst.Dict.GetStr(name); ok {
							hitVal = v
						} else {
							hitVal = &object.BoundMethod{Self: inst, Fn: snap.Val}
						}
						cacheHit = true
					case object.AttrCacheClassValue:
						if v, ok := inst.Dict.GetStr(name); ok {
							hitVal = v
						} else {
							hitVal = snap.Val
						}
						cacheHit = true
					}
				}
				if cacheHit {
					if pushSelf {
						f.push(hitVal)
						f.push(nil)
					} else {
						f.push(hitVal)
					}
					continue
				}
				// slow path — compute, then fill cache.
				var val object.Object
				val, err = i.getAttr(inst, name)
				if err != nil {
					goto handleErr
				}
				// Populate cache under lock.
				f.Code.Mu.Lock()
				entry := &f.Code.AttrCache[startIP]
				if _, inInst := inst.Dict.GetStr(name); inInst {
					if inst.Class == nil {
						entry.Cls = nil
						entry.Kind = object.AttrCacheInstDict
					} else if raw, ok := classLookup(inst.Class, name); !ok || !isDataDescriptor(raw) {
						entry.Cls = inst.Class
						entry.Kind = object.AttrCacheInstDict
					}
				} else if inst.Class != nil {
					if raw, ok := classLookup(inst.Class, name); ok {
						switch fn := raw.(type) {
						case *object.Function:
							entry.Cls = inst.Class
							entry.Kind = object.AttrCacheClassMethod
							entry.Val = fn
						case *object.BuiltinFunc:
							entry.Cls = inst.Class
							entry.Kind = object.AttrCacheClassMethod
							entry.Val = raw
						default:
							if !isDataDescriptor(raw) {
								switch raw.(type) {
								case *object.Int, *object.Str, *object.Float, *object.Bool,
									*object.Tuple, *object.NoneType:
									entry.Cls = inst.Class
									entry.Kind = object.AttrCacheClassValue
									entry.Val = raw
								}
							}
						}
					}
				}
				f.Code.Mu.Unlock()
				if pushSelf {
					f.push(val)
					f.push(nil)
				} else {
					f.push(val)
				}
				continue
			}
			var val object.Object
			val, err = i.getAttr(obj, name)
			if err != nil {
				goto handleErr
			}
			if pushSelf {
				f.push(val)
				f.push(nil)
			} else {
				f.push(val)
			}
		case op.STORE_ATTR,
			op.STORE_ATTR_INSTANCE_VALUE, op.STORE_ATTR_SLOT, op.STORE_ATTR_WITH_HINT:
			name := f.Code.Names[oparg]
			obj := f.pop()
			val := f.pop()
			if err = i.setAttr(obj, name, val); err != nil {
				goto handleErr
			}
		case op.DELETE_ATTR:
			name := f.Code.Names[oparg]
			obj := f.pop()
			if err = i.delAttr(obj, name); err != nil {
				goto handleErr
			}

		// --- arithmetic ---
		case op.BINARY_OP, op.BINARY_OP_EXTEND:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			// Inline Int+Int / Int-Int / Int*Int / Int%Int fast paths: for the
			// vast majority of integer math the values fit in int64 and the
			// generic dispatch just adds interface call + allocation overhead.
			// Results land back in Stack[sp-1] and SP drops by one in place.
			if ai, ok := a.(*object.Int); ok {
				if bi, ok := b.(*object.Int); ok {
					if ai.IsInt64() && bi.IsInt64() {
						av, bv := ai.Int64(), bi.Int64()
						switch oparg {
						case op.NB_ADD, op.NB_INPLACE_ADD:
							sum := av + bv
							if (av >= 0) == (bv >= 0) && (sum >= 0) != (av >= 0) {
								break
							}
							f.Stack[sp-1] = object.IntFromInt64(sum)
							f.SP = sp
							continue
						case op.NB_SUBTRACT, op.NB_INPLACE_SUBTRACT:
							diff := av - bv
							if (av >= 0) != (bv >= 0) && (diff >= 0) != (av >= 0) {
								break
							}
							f.Stack[sp-1] = object.IntFromInt64(diff)
							f.SP = sp
							continue
						case op.NB_MULTIPLY, op.NB_INPLACE_MULTIPLY:
							if av >= math.MinInt32 && av <= math.MaxInt32 &&
								bv >= math.MinInt32 && bv <= math.MaxInt32 {
								f.Stack[sp-1] = object.IntFromInt64(av * bv)
								f.SP = sp
								continue
							}
						case op.NB_REMAINDER, op.NB_INPLACE_REMAINDER:
							if bv != 0 {
								// Python remainder has the sign of the divisor.
								m := av % bv
								if (m < 0) != (bv < 0) && m != 0 {
									m += bv
								}
								f.Stack[sp-1] = object.IntFromInt64(m)
								f.SP = sp
								continue
							}
						case op.NB_FLOOR_DIVIDE, op.NB_INPLACE_FLOOR_DIVIDE:
							if bv != 0 {
								q := av / bv
								if (av%bv != 0) && ((av < 0) != (bv < 0)) {
									q--
								}
								f.Stack[sp-1] = object.IntFromInt64(q)
								f.SP = sp
								continue
							}
						case op.NB_AND, op.NB_INPLACE_AND:
							f.Stack[sp-1] = object.IntFromInt64(av & bv)
							f.SP = sp
							continue
						case op.NB_OR, op.NB_INPLACE_OR:
							f.Stack[sp-1] = object.IntFromInt64(av | bv)
							f.SP = sp
							continue
						case op.NB_XOR, op.NB_INPLACE_XOR:
							f.Stack[sp-1] = object.IntFromInt64(av ^ bv)
							f.SP = sp
							continue
						}
					}
				}
			}
			// Float+Float / Float+Int likewise: avoid interface thrash.
			if af, ok := a.(*object.Float); ok {
				if r, ok := floatFast(af.V, b, oparg); ok {
					f.Stack[sp-1] = r
					f.SP = sp
					continue
				}
			}
			if bf, ok := b.(*object.Float); ok {
				if ai, ok := a.(*object.Int); ok && ai.IsInt64() {
					if r, ok := floatFast(float64(ai.Int64()), bf, oparg); ok {
						f.Stack[sp-1] = r
						f.SP = sp
						continue
					}
				}
			}
			f.SP = sp - 1
			result, err = i.binaryOp(a, b, oparg)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_ADD_INT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if ai, ok := a.(*object.Int); ok {
				if bi, ok := b.(*object.Int); ok {
					if ai.IsInt64() && bi.IsInt64() {
						av, bv := ai.Int64(), bi.Int64()
						sum := av + bv
						if (av >= 0) == (bv >= 0) && (sum >= 0) != (av >= 0) {
							// overflow — fall through to big-int path
						} else {
							f.Stack[sp-1] = object.IntFromInt64(sum)
							f.SP = sp
							continue
						}
					}
					f.Stack[sp-1] = object.IntFromBig(new(big.Int).Add(&ai.V, &bi.V))
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.add(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_ADD_FLOAT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if af, ok := a.(*object.Float); ok {
				if bf, ok := b.(*object.Float); ok {
					f.Stack[sp-1] = &object.Float{V: af.V + bf.V}
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.add(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_ADD_UNICODE:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if as, ok := a.(*object.Str); ok {
				if bs, ok := b.(*object.Str); ok {
					f.Stack[sp-1] = &object.Str{V: as.V + bs.V}
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.add(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_SUBTRACT_INT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if ai, ok := a.(*object.Int); ok {
				if bi, ok := b.(*object.Int); ok {
					if ai.IsInt64() && bi.IsInt64() {
						av, bv := ai.Int64(), bi.Int64()
						diff := av - bv
						if (av >= 0) != (bv >= 0) && (diff >= 0) != (av >= 0) {
							// overflow
						} else {
							f.Stack[sp-1] = object.IntFromInt64(diff)
							f.SP = sp
							continue
						}
					}
					f.Stack[sp-1] = object.IntFromBig(new(big.Int).Sub(&ai.V, &bi.V))
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.sub(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_SUBTRACT_FLOAT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if af, ok := a.(*object.Float); ok {
				if bf, ok := b.(*object.Float); ok {
					f.Stack[sp-1] = &object.Float{V: af.V - bf.V}
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.sub(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_MULTIPLY_INT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if ai, ok := a.(*object.Int); ok {
				if bi, ok := b.(*object.Int); ok {
					if ai.IsInt64() && bi.IsInt64() {
						av, bv := ai.Int64(), bi.Int64()
						// Safe multiplication for values that fit in int32.
						if av >= math.MinInt32 && av <= math.MaxInt32 &&
							bv >= math.MinInt32 && bv <= math.MaxInt32 {
							f.Stack[sp-1] = object.IntFromInt64(av * bv)
							f.SP = sp
							continue
						}
					}
					f.Stack[sp-1] = object.IntFromBig(new(big.Int).Mul(&ai.V, &bi.V))
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
			result, err = i.mul(a, b)
			if err != nil {
				goto handleErr
			}
			f.push(result)
		case op.BINARY_OP_MULTIPLY_FLOAT:
			sp := f.SP - 1
			b := f.Stack[sp]
			a := f.Stack[sp-1]
			if af, ok := a.(*object.Float); ok {
				if bf, ok := b.(*object.Float); ok {
					f.Stack[sp-1] = &object.Float{V: af.V * bf.V}
					f.SP = sp
					continue
				}
			}
			f.SP = sp - 1
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
			if err = i.setitem(c, k, v); err != nil {
				goto handleErr
			}
		case op.STORE_SLICE:
			stop := f.pop()
			start := f.pop()
			c := f.pop()
			v := f.pop()
			if err = i.setitem(c, &object.Slice{Start: start, Stop: stop, Step: object.None}, v); err != nil {
				goto handleErr
			}
		case op.DELETE_SUBSCR:
			k := f.pop()
			c := f.pop()
			if err = i.delitem(c, k); err != nil {
				goto handleErr
			}

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
			if inst, ok := v.(*object.Instance); ok {
				if r, ok, err := i.callInstanceDunder(inst, "__invert__"); ok {
					if err != nil {
						goto handleErr
					}
					f.push(r)
					break
				}
			}
			bi, ok := toBigInt(v)
			if !ok {
				return nil, object.Errorf(i.typeErr, "bad operand for ~")
			}
			r := new(big.Int).Not(bi)
			f.push(object.IntFromBig(r))
		case op.TO_BOOL, op.TO_BOOL_ALWAYS_TRUE, op.TO_BOOL_BOOL,
			op.TO_BOOL_INT, op.TO_BOOL_LIST, op.TO_BOOL_NONE, op.TO_BOOL_STR:
			f.setTop(object.BoolOf(object.Truthy(f.top())))

		// --- comparisons ---
		case op.COMPARE_OP, op.COMPARE_OP_INT, op.COMPARE_OP_FLOAT, op.COMPARE_OP_STR:
			b := f.pop()
			a := f.pop()
			kind := int(oparg >> 5)
			// Inline int64/float64 compares. CPython's PyCompare_IntInt is
			// the equivalent fast path; avoids the interface-method dispatch
			// through i.compare plus the BoolOf lookup.
			if ai, ok := a.(*object.Int); ok {
				if bi, ok := b.(*object.Int); ok && ai.IsInt64() && bi.IsInt64() {
					av, bv := ai.Int64(), bi.Int64()
					var r bool
					switch kind {
					case 0: // <
						r = av < bv
					case 1: // <=
						r = av <= bv
					case 2: // ==
						r = av == bv
					case 3: // !=
						r = av != bv
					case 4: // >
						r = av > bv
					case 5: // >=
						r = av >= bv
					default:
						goto compareSlow
					}
					f.push(object.BoolOf(r))
					continue
				}
			}
			if af, ok := a.(*object.Float); ok {
				if bf, ok := b.(*object.Float); ok {
					av, bv := af.V, bf.V
					var r bool
					switch kind {
					case 0:
						r = av < bv
					case 1:
						r = av <= bv
					case 2:
						r = av == bv
					case 3:
						r = av != bv
					case 4:
						r = av > bv
					case 5:
						r = av >= bv
					default:
						goto compareSlow
					}
					f.push(object.BoolOf(r))
					continue
				}
			}
		compareSlow:
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
				if err = s.Add(f.Stack[k]); err != nil {
					goto handleErr
				}
			}
			f.SP -= n
			f.push(s)
		case op.BUILD_MAP:
			n := int(oparg)
			d := object.NewDict()
			base := f.SP - 2*n
			for k := 0; k < n; k++ {
				if err = d.Set(f.Stack[base+2*k], f.Stack[base+2*k+1]); err != nil {
					goto handleErr
				}
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

		case op.BUILD_INTERPOLATION:
			// PEP 750: build an Interpolation object for a t-string.
			// oparg encoding:
			//   bit 0: has_format_spec (format_spec string is on stack above expr_str)
			//   bit 1: always 1 (has_expression_str, always present)
			//   bits 3-2: conversion (0=None, 1='s', 2='r', 3='a')
			// Stack: value, expr_str [, format_spec if bit 0 set]
			var fmtSpec string
			if oparg&1 != 0 {
				if s, ok := f.pop().(*object.Str); ok {
					fmtSpec = s.V
				}
			}
			exprStr := ""
			if s, ok := f.pop().(*object.Str); ok {
				exprStr = s.V
			}
			value := f.pop()
			convCode := (oparg >> 2) & 3
			conv := ""
			switch convCode {
			case 1:
				conv = "s"
			case 2:
				conv = "r"
			case 3:
				conv = "a"
			}
			f.push(&object.Interpolation{
				Value:      value,
				Expression: exprStr,
				Conversion: conv,
				FormatSpec: fmtSpec,
			})

		case op.BUILD_TEMPLATE:
			// PEP 750: build a Template from two stack items:
			//   TOS   = tuple of Interpolation objects
			//   TOS-1 = tuple of string parts (len = len(interps)+1)
			interps := f.pop()
			strs := f.pop()
			var interpObjs []*object.Interpolation
			if t, ok := interps.(*object.Tuple); ok {
				for _, o := range t.V {
					if interp, ok2 := o.(*object.Interpolation); ok2 {
						interpObjs = append(interpObjs, interp)
					}
				}
			}
			var strObjs []*object.Str
			if t, ok := strs.(*object.Tuple); ok {
				for _, o := range t.V {
					if s, ok2 := o.(*object.Str); ok2 {
						strObjs = append(strObjs, s)
					}
				}
			}
			f.push(&object.Template{Strings: strObjs, Interpolations: interpObjs})

		case op.LIST_APPEND:
			v := f.pop()
			l := f.peek(int(oparg) - 1).(*object.List)
			l.Append(v)
		case op.LIST_EXTEND:
			it := f.pop()
			l := f.peek(int(oparg) - 1).(*object.List)
			var items []object.Object
			items, err = iterate(i, it)
			if err != nil {
				goto handleErr
			}
			l.Extend(items)
		case op.SET_ADD:
			v := f.pop()
			s := f.peek(int(oparg) - 1).(*object.Set)
			if err = s.Add(v); err != nil {
				goto handleErr
			}
		case op.SET_UPDATE:
			it := f.pop()
			s := f.peek(int(oparg) - 1).(*object.Set)
			var items []object.Object
			items, err = iterate(i, it)
			if err != nil {
				goto handleErr
			}
			for _, x := range items {
				if err = s.Add(x); err != nil {
					goto handleErr
				}
			}
		case op.MAP_ADD:
			v := f.pop()
			k := f.pop()
			d := f.peek(int(oparg) - 1).(*object.Dict)
			if err = d.Set(k, v); err != nil {
				goto handleErr
			}
		case op.DICT_UPDATE, op.DICT_MERGE:
			src := f.pop()
			d := f.peek(int(oparg) - 1).(*object.Dict)
			sd, ok := src.(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "expected dict, got %s", object.TypeName(src))
			}
			ks, vs := sd.Items()
			for k, key := range ks {
				if err = d.Set(key, vs[k]); err != nil {
					goto handleErr
				}
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
		case op.FOR_ITER, op.FOR_ITER_LIST, op.FOR_ITER_TUPLE, op.FOR_ITER_RANGE, op.FOR_ITER_GEN:
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
				err = ierr
				goto handleErr
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
			op.CALL_NON_PY_GENERAL, op.CALL_STR_1, op.CALL_TUPLE_1, op.CALL_TYPE_1,
			op.CALL_ALLOC_AND_ENTER_INIT:
			n := int(oparg)
			base := f.SP - n - 2
			callable := f.Stack[base]
			selfOrNull := f.Stack[base+1]
			args := f.Stack[base+2 : f.SP]
			// Fast path: plain Python function, no self injection needed,
			// arity matches, no *args/**kwargs/kwonly. Skips bindArgs and
			// the temporary call-slice allocation.
			if selfOrNull == nil {
				if fn, ok := callable.(*object.Function); ok && isFastCallable(fn, n) {
					f.SP = base
					r, cerr := i.callFunctionFast(fn, nil, args)
					if cerr != nil {
						err = cerr
						goto handleErr
					}
					f.push(r)
					continue
				}
				if bm, ok := callable.(*object.BoundMethod); ok {
					if fn, ok := bm.Fn.(*object.Function); ok && isFastCallable(fn, n+1) {
						f.SP = base
						r, cerr := i.callFunctionFast(fn, bm.Self, args)
						if cerr != nil {
							err = cerr
							goto handleErr
						}
						f.push(r)
						continue
					}
					// Fast path for BoundMethod{Fn:*BuiltinFunc}: avoid the
					// temporary [self, ...args] slice allocation by reusing the
					// stack slot at base+1 (currently nil selfOrNull) as the
					// self argument in place.
					if bfn, ok := bm.Fn.(*object.BuiltinFunc); ok {
						f.Stack[base+1] = bm.Self
						callArgs := f.Stack[base+1 : f.SP]
						r, cerr := bfn.Call(i, callArgs, nil)
						f.SP = base
						if cerr != nil {
							err = cerr
							goto handleErr
						}
						f.push(r)
						continue
					}
				}
			}
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
			// Fast path: *Function with no *args/**kwargs, optional bound self.
			if fn, ok := callable.(*object.Function); ok {
				nPosTotal := posCount
				if selfOrNull != nil {
					nPosTotal++
				}
				if isFastKwCallable(fn, nPosTotal) {
					var posArgs []object.Object
					if selfOrNull != nil {
						posArgs = f.Stack[base+1 : base+1+nPosTotal]
					} else {
						posArgs = pos
					}
					r, callErr := i.callFunctionFastKw(fn, posArgs, kwnames.V, kwVals)
					f.SP = base
					if callErr != nil {
						err = callErr
						goto handleErr
					}
					f.push(r)
					continue
				}
			}
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
			f.YieldIP = startIP
			f.Yielded = f.pop()
			return nil, errYielded
		case op.SEND, op.SEND_GEN:
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
			// co_consts[0] is the docstring if it's a string.
			if len(codeObj.Consts) > 0 {
				if ds, ok := codeObj.Consts[0].(*object.Str); ok {
					fn.Doc = ds
				}
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
			case 0x10:
				// PEP 649: __annotate__ — lazy annotation function.
				fn.Annotate = val
			}
			f.push(fn)
		case op.LOAD_BUILD_CLASS:
			bc, _ := i.Builtins.GetStr("__build_class__")
			f.push(bc)
		case op.LOAD_LOCALS:
			f.push(f.Locals)

		case op.EXIT_INIT_CHECK:
			// Validate that __init__ returned None.
			v := f.pop()
			if _, ok := v.(*object.NoneType); !ok {
				err = object.Errorf(i.typeErr, "__init__() should return None, not '%.100s'", object.TypeName(v))
				goto handleErr
			}

		case op.SETUP_ANNOTATIONS:
			// Ensure __annotations__ dict exists in the local namespace.
			if _, ok := f.Locals.GetStr("__annotations__"); !ok {
				f.Locals.SetStr("__annotations__", object.NewDict())
			}

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
				break
			}
			if r, ok, ferr := i.instanceFormat(v, ""); ok {
				if ferr != nil {
					err = ferr
					goto handleErr
				}
				f.push(&object.Str{V: r})
				break
			}
			f.push(&object.Str{V: object.Str_(v)})
		case op.FORMAT_WITH_SPEC:
			spec := f.pop().(*object.Str)
			v := f.pop()
			var s string
			if r, ok, ferr := i.instanceFormat(v, spec.V); ok {
				if ferr != nil {
					err = ferr
					goto handleErr
				}
				s = r
			} else {
				s, err = formatValue(v, spec.V)
				if err != nil {
					goto handleErr
				}
			}
			f.push(&object.Str{V: s})

		// --- import ---
		case op.IMPORT_NAME:
			name := f.Code.Names[oparg]
			fromlist := f.pop()
			levelObj := f.pop()
			level := 0
			if l, ok := levelObj.(*object.Int); ok {
				level = int(l.Int64())
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
			// Generators/iterators are already awaitable. User classes
			// expose `__await__`, which must return an iterator; we
			// swap the instance for the iterator on the stack. Pass
			// a generator through unwrapped so SEND preserves its
			// StopIteration(value) — wrapping via getIter would turn
			// that into "exhausted" and lose the awaited return value.
			v := f.top()
			switch x := v.(type) {
			case *object.Generator, *object.Iter:
				// already awaitable
			case *object.Instance:
				r, ok, cerr := i.callInstanceDunder(x, "__await__")
				if cerr != nil {
					err = cerr
					goto handleErr
				}
				if !ok {
					err = object.Errorf(i.typeErr, "object %s can't be used in 'await' expression", object.TypeName(v))
					goto handleErr
				}
				switch r.(type) {
				case *object.Generator, *object.Iter:
					f.setTop(r)
				default:
					it, ierr := i.getIter(r)
					if ierr != nil {
						err = ierr
						goto handleErr
					}
					f.setTop(it)
				}
			default:
				err = object.Errorf(i.typeErr, "object %s can't be used in 'await' expression", object.TypeName(v))
				goto handleErr
			}

		case op.GET_AITER:
			// async for: pop the async iterable, call __aiter__(), push the
			// resulting async iterator.
			o := f.pop()
			fn, gerr := i.getAttr(o, "__aiter__")
			if gerr != nil {
				err = object.Errorf(i.typeErr, "'%.100s' object is not an async iterable", object.TypeName(o))
				goto handleErr
			}
			r, cerr := i.callObject(fn, nil, nil)
			if cerr != nil {
				err = cerr
				goto handleErr
			}
			f.push(r)

		case op.GET_ANEXT:
			// async for: peek at TOS (async iterator stays), call __anext__(),
			// push the coroutine/awaitable returned.
			ait := f.top()
			fn, gerr := i.getAttr(ait, "__anext__")
			if gerr != nil {
				err = object.Errorf(i.typeErr, "'%.100s' object is not an async iterator", object.TypeName(ait))
				goto handleErr
			}
			r, cerr := i.callObject(fn, nil, nil)
			if cerr != nil {
				err = cerr
				goto handleErr
			}
			f.push(r)

		case op.END_ASYNC_FOR:
			// Reached via exception table when StopAsyncIteration propagates
			// from inside the async for coroutine. Stack: [ait, exc].
			exc := f.pop()
			f.pop() // async iterator — discard
			if e, ok := exc.(*object.Exception); ok && object.IsSubclass(e.Class, i.stopAsyncIter) {
				// StopAsyncIteration: loop ended normally; continue execution.
			} else {
				// Other exception: re-raise.
				if e, ok2 := exc.(*object.Exception); ok2 {
					err = e
				} else {
					err = object.Errorf(i.runtimeErr, "END_ASYNC_FOR: expected exception on stack")
				}
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
			// Implicit context: if a new exception is raised while handling
			// another, link the old one into .Ctx so traceback formatting
			// can print "During handling of the above exception...".
			if e, ok := err.(*object.Exception); ok && f.ExcInfo != nil && e.Ctx == nil && e != f.ExcInfo {
				e.Ctx = f.ExcInfo
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
			// Make the newly caught exception the frame's current exc_info
			// so sys.exc_info() reads it during the handler body.
			if e, ok := cur.(*object.Exception); ok {
				f.ExcInfo = e
			}
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
			match := false
			switch t := excType.(type) {
			case *object.Class:
				match = object.IsSubclass(tos.Class, t)
			case *object.Tuple:
				for _, x := range t.V {
					cls, ok := x.(*object.Class)
					if !ok {
						return nil, object.Errorf(i.typeErr, "except type must be a class")
					}
					if object.IsSubclass(tos.Class, cls) {
						match = true
						break
					}
				}
			default:
				return nil, object.Errorf(i.typeErr, "except type must be a class")
			}
			f.push(object.BoolOf(match))
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
				if inst2.Class != nil {
					if v, ok := classLookup(inst2.Class, name); ok {
						method = v
					}
				}
			}
			if method == nil {
				// Fall back to getAttr for non-instance objects (e.g. File,
				// StringIO) that expose __enter__/__exit__ through getAttr.
				if m, gerr := i.getAttr(inst, name); gerr == nil {
					method = m
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
			// PEP 654 except*: split TOS exception by `cls`. Replace TOS
			// with the unmatched remainder (or None if fully matched), and
			// push the matched part (or None if nothing matched).
			excType := f.pop()
			tos := f.pop().(*object.Exception)
			cls, _ := excType.(*object.Class)
			if cls == nil {
				f.push(tos)
				f.push(object.None)
				continue
			}
			// Non-group: behaves like CHECK_EXC_MATCH.
			isGroup := object.IsSubclass(tos.Class, i.baseExcGroup)
			if !isGroup {
				if object.IsSubclass(tos.Class, cls) {
					f.push(object.None)
					f.push(tos)
				} else {
					f.push(tos)
					f.push(object.None)
				}
				continue
			}
			// Group: scan inner exceptions (Args[1]) and split.
			var inners []object.Object
			if tos.Args != nil && len(tos.Args.V) >= 2 {
				if l, ok := tos.Args.V[1].(*object.List); ok {
					inners = l.V
				} else if t, ok := tos.Args.V[1].(*object.Tuple); ok {
					inners = t.V
				}
			}
			var matched, unmatched []object.Object
			for _, inner := range inners {
				if e, ok := inner.(*object.Exception); ok && object.IsSubclass(e.Class, cls) {
					matched = append(matched, inner)
				} else {
					unmatched = append(unmatched, inner)
				}
			}
			mkGroup := func(items []object.Object) object.Object {
				if len(items) == 0 {
					return object.None
				}
				msg := ""
				if tos.Args != nil && len(tos.Args.V) >= 1 {
					if s, ok := tos.Args.V[0].(*object.Str); ok {
						msg = s.V
					}
				}
				return &object.Exception{
					Class: tos.Class,
					Args: &object.Tuple{V: []object.Object{
						&object.Str{V: msg},
						&object.List{V: items},
					}},
				}
			}
			f.push(mkGroup(unmatched))
			f.push(mkGroup(matched))

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
		// Record the frame's position at the point of the raise, for both
		// caught and uncaught paths — CPython's traceback includes every
		// frame the exception passes through, including the catching one.
		f.LastIP = startIP
		extendTraceback(e, f)
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
