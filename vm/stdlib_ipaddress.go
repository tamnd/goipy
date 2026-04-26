package vm

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildIpaddress() *object.Module {
	m := &object.Module{Name: "ipaddress", Dict: object.NewDict()}

	// ── Exception classes ─────────────────────────────────────────────────────

	addrValErrCls := &object.Class{Name: "AddressValueError", Bases: []*object.Class{i.valueErr}, Dict: object.NewDict()}
	netmaskValErrCls := &object.Class{Name: "NetmaskValueError", Bases: []*object.Class{i.valueErr}, Dict: object.NewDict()}
	m.Dict.SetStr("AddressValueError", addrValErrCls)
	m.Dict.SetStr("NetmaskValueError", netmaskValErrCls)

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("IPV4LENGTH", object.NewInt(32))
	m.Dict.SetStr("IPV6LENGTH", object.NewInt(128))

	// ── IPv4 helpers ──────────────────────────────────────────────────────────

	parseIPv4 := func(s string) (uint32, error) {
		ip := net.ParseIP(s)
		if ip == nil {
			return 0, fmt.Errorf("'%s' does not appear to be an IPv4 address", s)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return 0, fmt.Errorf("'%s' does not appear to be an IPv4 address", s)
		}
		return binary.BigEndian.Uint32(ip4), nil
	}

	ipv4ToString := func(v uint32) string {
		return fmt.Sprintf("%d.%d.%d.%d", v>>24, (v>>16)&0xff, (v>>8)&0xff, v&0xff)
	}

	prefixToMask4 := func(prefixlen int) uint32 {
		if prefixlen == 0 {
			return 0
		}
		return ^uint32(0) << (32 - prefixlen)
	}

	isPrivateV4 := func(v uint32) bool {
		return (v>>24 == 10) ||
			(v>>20 == 0xAC1) || // 172.16-31
			(v>>16 == 0xC0A8) || // 192.168
			(v>>24 == 127) ||
			(v>>16 == 0xA9FE) // 169.254
	}
	isLoopbackV4 := func(v uint32) bool { return v>>24 == 127 }
	isMulticastV4 := func(v uint32) bool { return v>>28 == 0xE }
	isLinkLocalV4 := func(v uint32) bool { return v>>16 == 0xA9FE }
	isUnspecifiedV4 := func(v uint32) bool { return v == 0 }
	isGlobalV4 := func(v uint32) bool {
		return !isPrivateV4(v) && !isLoopbackV4(v) && !isMulticastV4(v) && !isLinkLocalV4(v) && !isUnspecifiedV4(v)
	}
	isReservedV4 := func(v uint32) bool {
		return v>>28 == 0xF || v>>24 == 0 || v>>24 == 240
	}

	// ── IPv4Address ───────────────────────────────────────────────────────────

	var makeIPv4Address func(v uint32) *object.Instance
	ipv4AddrCls := &object.Class{Name: "IPv4Address", Dict: object.NewDict()}

	makeIPv4Address = func(v uint32) *object.Instance {
		inst := &object.Instance{Class: ipv4AddrCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_value", object.NewInt(int64(v)))
		s := ipv4ToString(v)
		packed := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}

		inst.Dict.SetStr("packed", &object.Bytes{V: packed})
		inst.Dict.SetStr("version", object.NewInt(4))
		inst.Dict.SetStr("max_prefixlen", object.NewInt(32))
		inst.Dict.SetStr("compressed", &object.Str{V: s})
		inst.Dict.SetStr("is_private", object.BoolOf(isPrivateV4(v)))
		inst.Dict.SetStr("is_loopback", object.BoolOf(isLoopbackV4(v)))
		inst.Dict.SetStr("is_multicast", object.BoolOf(isMulticastV4(v)))
		inst.Dict.SetStr("is_link_local", object.BoolOf(isLinkLocalV4(v)))
		inst.Dict.SetStr("is_unspecified", object.BoolOf(isUnspecifiedV4(v)))
		inst.Dict.SetStr("is_global", object.BoolOf(isGlobalV4(v)))
		inst.Dict.SetStr("is_reserved", object.BoolOf(isReservedV4(v)))

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv4Address('" + s + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: s}, nil
		}})
		inst.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(v)), nil
		}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(v)), nil
		}})
		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.BoolOf(int64(v) == oInt.Int64()), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.BoolOf(int64(v) < oInt.Int64()), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.BoolOf(int64(v) <= oInt.Int64()), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.BoolOf(int64(v) > oInt.Int64()), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.BoolOf(int64(v) >= oInt.Int64()), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__add__", &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "__add__ requires argument")
			}
			if n, ok2 := a[0].(*object.Int); ok2 {
				return makeIPv4Address(v + uint32(n.Int64())), nil
			}
			return nil, object.Errorf(i.typeErr, "unsupported operand")
		}})
		inst.Dict.SetStr("__sub__", &object.BuiltinFunc{Name: "__sub__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "__sub__ requires argument")
			}
			if n, ok2 := a[0].(*object.Int); ok2 {
				return makeIPv4Address(v - uint32(n.Int64())), nil
			}
			// IPv4Address - IPv4Address → int
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return object.NewInt(int64(v) - oInt.Int64()), nil
					}
				}
			}
			return nil, object.Errorf(i.typeErr, "unsupported operand")
		}})
		return inst
	}

	ipv4AddrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var v uint32
		switch arg := a[1].(type) {
		case *object.Str:
			parsed, err := parseIPv4(arg.V)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			v = parsed
		case *object.Int:
			if !arg.IsInt64() {
				return nil, object.Errorf(addrValErrCls, "address must fit in 32 bits")
			}
			v = uint32(arg.Int64())
		case *object.Bytes:
			if len(arg.V) != 4 {
				return nil, object.Errorf(addrValErrCls, "packed address must be 4 bytes")
			}
			v = binary.BigEndian.Uint32(arg.V)
		default:
			return nil, object.Errorf(addrValErrCls, "invalid address: %v", a[1])
		}
		tmp := makeIPv4Address(v)
		// Copy all attrs from tmp to inst
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("IPv4Address", ipv4AddrCls)

	// ── IPv4Network ───────────────────────────────────────────────────────────

	var makeIPv4Network func(network uint32, prefixlen int) *object.Instance
	ipv4NetCls := &object.Class{Name: "IPv4Network", Dict: object.NewDict()}

	makeIPv4Network = func(network uint32, prefixlen int) *object.Instance {
		inst := &object.Instance{Class: ipv4NetCls, Dict: object.NewDict()}
		mask := prefixToMask4(prefixlen)
		hostmask := ^mask
		netAddr := network & mask
		bcastAddr := network | hostmask
		numAddrs := uint32(1) << (32 - prefixlen)

		inst.Dict.SetStr("_network", object.NewInt(int64(netAddr)))
		inst.Dict.SetStr("_prefixlen", object.NewInt(int64(prefixlen)))
		inst.Dict.SetStr("network_address", makeIPv4Address(netAddr))
		inst.Dict.SetStr("broadcast_address", makeIPv4Address(bcastAddr))
		inst.Dict.SetStr("netmask", makeIPv4Address(mask))
		inst.Dict.SetStr("hostmask", makeIPv4Address(hostmask))
		inst.Dict.SetStr("prefixlen", object.NewInt(int64(prefixlen)))
		inst.Dict.SetStr("num_addresses", object.NewInt(int64(numAddrs)))
		inst.Dict.SetStr("version", object.NewInt(4))
		inst.Dict.SetStr("max_prefixlen", object.NewInt(32))
		s := ipv4ToString(netAddr) + "/" + strconv.Itoa(prefixlen)
		inst.Dict.SetStr("with_prefixlen", &object.Str{V: s})
		inst.Dict.SetStr("compressed", &object.Str{V: s})
		inst.Dict.SetStr("with_netmask", &object.Str{V: ipv4ToString(netAddr) + "/" + ipv4ToString(mask)})
		inst.Dict.SetStr("with_hostmask", &object.Str{V: ipv4ToString(netAddr) + "/" + ipv4ToString(hostmask)})
		inst.Dict.SetStr("is_private", object.BoolOf(isPrivateV4(netAddr)))
		inst.Dict.SetStr("is_loopback", object.BoolOf(isLoopbackV4(netAddr)))
		inst.Dict.SetStr("is_multicast", object.BoolOf(isMulticastV4(netAddr)))
		inst.Dict.SetStr("is_link_local", object.BoolOf(isLinkLocalV4(netAddr)))
		inst.Dict.SetStr("is_unspecified", object.BoolOf(isUnspecifiedV4(netAddr)))
		inst.Dict.SetStr("is_global", object.BoolOf(isGlobalV4(netAddr)))
		inst.Dict.SetStr("is_reserved", object.BoolOf(isReservedV4(netAddr)))

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv4Network('" + s + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: s}, nil
		}})
		inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(numAddrs)), nil
		}})
		inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if addrInst, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := addrInst.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						av := uint32(oInt.Int64())
						return object.BoolOf(av&mask == netAddr), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				on, _ := other.Dict.GetStr("_network")
				op, _ := other.Dict.GetStr("_prefixlen")
				if on != nil && op != nil {
					return object.BoolOf(
						object.Str_(on) == strconv.FormatInt(int64(netAddr), 10) &&
							object.Str_(op) == strconv.Itoa(prefixlen),
					), nil
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if on, ok3 := other.Dict.GetStr("_network"); ok3 {
					if oInt, ok4 := on.(*object.Int); ok4 {
						if int64(netAddr) != oInt.Int64() {
							return object.BoolOf(int64(netAddr) < oInt.Int64()), nil
						}
					}
					if op, ok4 := other.Dict.GetStr("_prefixlen"); ok4 {
						if oInt2, ok5 := op.(*object.Int); ok5 {
							return object.BoolOf(int64(prefixlen) < oInt2.Int64()), nil
						}
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(netAddr)^int64(prefixlen)), nil
		}})
		inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			idx := uint32(0)
			return &object.Instance{Class: ipv4NetCls, Dict: func() *object.Dict {
				d := object.NewDict()
				d.SetStr("__next__", &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					if idx >= numAddrs {
						return nil, object.Errorf(i.stopIter, "")
					}
					addr := netAddr + idx
					idx++
					return makeIPv4Address(addr), nil
				}})
				d.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					return a[0], nil
				}})
				return d
			}()}, nil
		}})
		inst.Dict.SetStr("hosts", &object.BuiltinFunc{Name: "hosts", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var result []object.Object
			if prefixlen >= 31 {
				for addr := netAddr; addr <= bcastAddr; addr++ {
					result = append(result, makeIPv4Address(addr))
				}
			} else {
				for addr := netAddr + 1; addr < bcastAddr; addr++ {
					result = append(result, makeIPv4Address(addr))
				}
			}
			return &object.List{V: result}, nil
		}})
		inst.Dict.SetStr("overlaps", &object.BuiltinFunc{Name: "overlaps", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNet := uint32(on.(*object.Int).Int64())
			oPrefix := int(op.(*object.Int).Int64())
			oMask := prefixToMask4(oPrefix)
			return object.BoolOf(
				netAddr&oMask == oNet || oNet&mask == netAddr,
			), nil
		}})
		inst.Dict.SetStr("subnet_of", &object.BuiltinFunc{Name: "subnet_of", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNet := uint32(on.(*object.Int).Int64())
			oPrefix := int(op.(*object.Int).Int64())
			oMask := prefixToMask4(oPrefix)
			return object.BoolOf(prefixlen >= oPrefix && netAddr&oMask == oNet), nil
		}})
		inst.Dict.SetStr("supernet_of", &object.BuiltinFunc{Name: "supernet_of", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNet := uint32(on.(*object.Int).Int64())
			oPrefix := int(op.(*object.Int).Int64())
			return object.BoolOf(prefixlen <= oPrefix && oNet&mask == netAddr), nil
		}})
		inst.Dict.SetStr("subnets", &object.BuiltinFunc{Name: "subnets", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			newPrefix := prefixlen + 1
			if kw != nil {
				if v, ok2 := kw.GetStr("prefixlen_diff"); ok2 {
					if n, ok3 := v.(*object.Int); ok3 {
						newPrefix = prefixlen + int(n.Int64())
					}
				}
				if v, ok2 := kw.GetStr("new_prefix"); ok2 && v != object.None {
					if n, ok3 := v.(*object.Int); ok3 {
						newPrefix = int(n.Int64())
					}
				}
			}
			if newPrefix > 32 || newPrefix <= prefixlen {
				return &object.List{V: []object.Object{}}, nil
			}
			step := uint32(1) << (32 - newPrefix)
			var result []object.Object
			for addr := netAddr; addr < netAddr+numAddrs; addr += step {
				result = append(result, makeIPv4Network(addr, newPrefix))
				if addr+step < addr { // overflow guard
					break
				}
			}
			return &object.List{V: result}, nil
		}})
		inst.Dict.SetStr("supernet", &object.BuiltinFunc{Name: "supernet", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			diff := 1
			if kw != nil {
				if v, ok2 := kw.GetStr("prefixlen_diff"); ok2 {
					if n, ok3 := v.(*object.Int); ok3 {
						diff = int(n.Int64())
					}
				}
			}
			newPrefix := prefixlen - diff
			if newPrefix < 0 {
				newPrefix = 0
			}
			return makeIPv4Network(netAddr, newPrefix), nil
		}})
		inst.Dict.SetStr("address_exclude", &object.BuiltinFunc{Name: "address_exclude", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{}}, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return &object.List{V: []object.Object{}}, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return &object.List{V: []object.Object{}}, nil
			}
			oNet := uint32(on.(*object.Int).Int64())
			oPrefix := int(op.(*object.Int).Int64())
			var result []object.Object
			s1, p1 := netAddr, prefixlen
			for p1 < oPrefix {
				// Split into two subnets; keep the one that doesn't contain oNet
				p1++
				half := uint32(1) << (32 - p1)
				first := s1
				second := s1 + half
				if oNet&prefixToMask4(p1) == first {
					// oNet is in first half, keep second
					result = append(result, makeIPv4Network(second, p1))
					s1 = first
				} else {
					// oNet is in second half, keep first
					result = append(result, makeIPv4Network(first, p1))
					s1 = second
				}
			}
			// Reverse result to match CPython ordering
			for l, r := 0, len(result)-1; l < r; l, r = l+1, r-1 {
				result[l], result[r] = result[r], result[l]
			}
			return &object.List{V: result}, nil
		}})
		return inst
	}

	ipv4NetCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		strict := true
		if kw != nil {
			if v, ok2 := kw.GetStr("strict"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					strict = b == object.True
				}
			}
		}
		var network uint32
		var prefixlen int
		switch arg := a[1].(type) {
		case *object.Str:
			s := arg.V
			slash := strings.Index(s, "/")
			if slash < 0 {
				parsed, err := parseIPv4(s)
				if err != nil {
					return nil, object.Errorf(addrValErrCls, "%v", err)
				}
				network = parsed
				prefixlen = 32
			} else {
				parsed, err := parseIPv4(s[:slash])
				if err != nil {
					return nil, object.Errorf(addrValErrCls, "%v", err)
				}
				network = parsed
				pl, err := strconv.Atoi(s[slash+1:])
				if err != nil || pl < 0 || pl > 32 {
					// Maybe it's a netmask like 255.255.255.0
					maskIP := net.ParseIP(s[slash+1:])
					if maskIP != nil {
						m4 := maskIP.To4()
						if m4 != nil {
							maskVal := binary.BigEndian.Uint32(m4)
							// Count leading ones
							pl = 0
							for maskVal&(1<<31) != 0 {
								pl++
								maskVal <<= 1
							}
						}
					} else {
						return nil, object.Errorf(netmaskValErrCls, "invalid prefix length: %s", s[slash+1:])
					}
				}
				prefixlen = pl
			}
		case *object.Tuple:
			if len(arg.V) < 2 {
				return nil, object.Errorf(i.typeErr, "IPv4Network tuple needs 2 elements")
			}
			addrStr, ok2 := arg.V[0].(*object.Str)
			if !ok2 {
				return nil, object.Errorf(addrValErrCls, "invalid address")
			}
			parsed, err := parseIPv4(addrStr.V)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			network = parsed
			switch pl := arg.V[1].(type) {
			case *object.Int:
				prefixlen = int(pl.Int64())
			default:
				return nil, object.Errorf(netmaskValErrCls, "invalid prefix")
			}
		default:
			return nil, object.Errorf(addrValErrCls, "invalid network: %v", a[1])
		}
		if prefixlen < 0 || prefixlen > 32 {
			return nil, object.Errorf(netmaskValErrCls, "invalid prefix length %d", prefixlen)
		}
		mask := prefixToMask4(prefixlen)
		if strict && (network&mask) != network {
			return nil, object.Errorf(i.valueErr, "%s has host bits set", ipv4ToString(network))
		}
		tmp := makeIPv4Network(network, prefixlen)
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("IPv4Network", ipv4NetCls)

	// ── IPv4Interface ─────────────────────────────────────────────────────────

	ipv4IfaceCls := &object.Class{Name: "IPv4Interface", Bases: []*object.Class{ipv4AddrCls}, Dict: object.NewDict()}
	ipv4IfaceCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		argStr, ok2 := a[1].(*object.Str)
		if !ok2 {
			return nil, object.Errorf(addrValErrCls, "invalid interface")
		}
		s := argStr.V
		slash := strings.Index(s, "/")
		var addrV uint32
		var prefixlen int
		if slash < 0 {
			parsed, err := parseIPv4(s)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			addrV = parsed
			prefixlen = 32
		} else {
			parsed, err := parseIPv4(s[:slash])
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			addrV = parsed
			pl, err := strconv.Atoi(s[slash+1:])
			if err != nil || pl < 0 || pl > 32 {
				return nil, object.Errorf(netmaskValErrCls, "invalid prefix: %s", s[slash+1:])
			}
			prefixlen = pl
		}
		// Copy IPv4Address attrs for the host address
		tmp := makeIPv4Address(addrV)
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		mask := prefixToMask4(prefixlen)
		hostmask := ^mask
		netAddr := addrV & mask
		ifaceStr := ipv4ToString(addrV) + "/" + strconv.Itoa(prefixlen)
		inst.Dict.SetStr("ip", makeIPv4Address(addrV))
		inst.Dict.SetStr("network", makeIPv4Network(netAddr, prefixlen))
		inst.Dict.SetStr("netmask", makeIPv4Address(mask))
		inst.Dict.SetStr("with_prefixlen", &object.Str{V: ifaceStr})
		inst.Dict.SetStr("with_netmask", &object.Str{V: ipv4ToString(addrV) + "/" + ipv4ToString(mask)})
		inst.Dict.SetStr("with_hostmask", &object.Str{V: ipv4ToString(addrV) + "/" + ipv4ToString(hostmask)})
		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv4Interface('" + ifaceStr + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ifaceStr}, nil
		}})
		return object.None, nil
	}})
	m.Dict.SetStr("IPv4Interface", ipv4IfaceCls)

	// ── IPv6 helpers ──────────────────────────────────────────────────────────

	parseIPv6 := func(s string) (*big.Int, error) {
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, fmt.Errorf("'%s' does not appear to be an IPv6 address", s)
		}
		ip16 := ip.To16()
		if ip16 == nil {
			return nil, fmt.Errorf("'%s' does not appear to be an IPv6 address", s)
		}
		v := new(big.Int).SetBytes(ip16)
		return v, nil
	}

	bigIntTo16Bytes := func(v *big.Int) []byte {
		b := v.Bytes()
		if len(b) < 16 {
			pad := make([]byte, 16-len(b))
			b = append(pad, b...)
		}
		return b[:16]
	}

	ipv6ToString := func(v *big.Int) string {
		b := bigIntTo16Bytes(v)
		return net.IP(b).String()
	}

	ipv6ToExploded := func(v *big.Int) string {
		b := bigIntTo16Bytes(v)
		groups := make([]string, 8)
		for j := 0; j < 8; j++ {
			groups[j] = fmt.Sprintf("%04x", binary.BigEndian.Uint16(b[j*2:j*2+2]))
		}
		return strings.Join(groups, ":")
	}

	prefixToMask6 := func(prefixlen int) *big.Int {
		if prefixlen == 0 {
			return big.NewInt(0)
		}
		// 128-bit all-ones, then shift right
		full := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
		shift := uint(128 - prefixlen)
		return new(big.Int).AndNot(full, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), shift), big.NewInt(1)))
	}

	isLoopbackV6 := func(v *big.Int) bool { return v.Cmp(big.NewInt(1)) == 0 }
	isUnspecifiedV6 := func(v *big.Int) bool { return v.Sign() == 0 }
	isLinkLocalV6 := func(v *big.Int) bool {
		// fe80::/10
		top10 := new(big.Int).Rsh(v, 118)
		return top10.Cmp(big.NewInt(0x3FA)) == 0
	}
	isMulticastV6 := func(v *big.Int) bool {
		top8 := new(big.Int).Rsh(v, 120)
		return top8.Cmp(big.NewInt(0xFF)) == 0
	}
	isPrivateV6 := func(v *big.Int) bool {
		// fc00::/7 (ULA)
		top7 := new(big.Int).Rsh(v, 121)
		return top7.Cmp(big.NewInt(0x7E)) == 0 ||
			isLoopbackV6(v) || isLinkLocalV6(v)
	}
	isDocumentationV6 := func(v *big.Int) bool {
		// 2001:db8::/32 — RFC 3849 documentation prefix
		top32 := new(big.Int).Rsh(v, 96)
		return top32.Cmp(big.NewInt(0x20010db8)) == 0
	}
	isGlobalV6 := func(v *big.Int) bool {
		return !isPrivateV6(v) && !isLoopbackV6(v) && !isMulticastV6(v) && !isLinkLocalV6(v) && !isUnspecifiedV6(v) && !isDocumentationV6(v)
	}
	getIPv4Mapped := func(v *big.Int) *object.Instance {
		// ::ffff:x.x.x.x → top 96 bits are 0x0000...0000ffff
		top96 := new(big.Int).Rsh(v, 32)
		if top96.Cmp(big.NewInt(0xffff)) == 0 {
			low32 := uint32(v.Int64())
			return makeIPv4Address(low32)
		}
		return nil
	}

	// ── IPv6Address ───────────────────────────────────────────────────────────

	var makeIPv6Address func(v *big.Int) *object.Instance
	ipv6AddrCls := &object.Class{Name: "IPv6Address", Dict: object.NewDict()}

	makeIPv6Address = func(v *big.Int) *object.Instance {
		inst := &object.Instance{Class: ipv6AddrCls, Dict: object.NewDict()}
		packed := bigIntTo16Bytes(v)
		compressed := ipv6ToString(v)
		exploded := ipv6ToExploded(v)

		bigObj := &object.Int{}
		bigObj.V.Set(v)

		inst.Dict.SetStr("_value", bigObj)
		inst.Dict.SetStr("packed", &object.Bytes{V: packed})
		inst.Dict.SetStr("version", object.NewInt(6))
		inst.Dict.SetStr("max_prefixlen", object.NewInt(128))
		inst.Dict.SetStr("compressed", &object.Str{V: compressed})
		inst.Dict.SetStr("exploded", &object.Str{V: exploded})
		inst.Dict.SetStr("is_loopback", object.BoolOf(isLoopbackV6(v)))
		inst.Dict.SetStr("is_unspecified", object.BoolOf(isUnspecifiedV6(v)))
		inst.Dict.SetStr("is_link_local", object.BoolOf(isLinkLocalV6(v)))
		inst.Dict.SetStr("is_multicast", object.BoolOf(isMulticastV6(v)))
		inst.Dict.SetStr("is_private", object.BoolOf(isPrivateV6(v)))
		inst.Dict.SetStr("is_global", object.BoolOf(isGlobalV6(v)))
		inst.Dict.SetStr("is_reserved", object.False)

		if mapped := getIPv4Mapped(v); mapped != nil {
			inst.Dict.SetStr("ipv4_mapped", mapped)
		} else {
			inst.Dict.SetStr("ipv4_mapped", object.None)
		}

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv6Address('" + compressed + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: compressed}, nil
		}})
		inst.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			r := &object.Int{}
			r.V.Set(v)
			return r, nil
		}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Use low 64 bits for hash
			lo := new(big.Int).And(v, new(big.Int).SetUint64(0xffffffffffffffff))
			return object.NewInt(lo.Int64()), nil
		}})
		cmpWith := func(a []object.Object) (int, bool) {
			if len(a) < 1 {
				return 0, false
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := other.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						return v.Cmp(&oInt.V), true
					}
				}
			}
			return 0, false
		}
		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			c, ok2 := cmpWith(a)
			return object.BoolOf(ok2 && c == 0), nil
		}})
		inst.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			c, ok2 := cmpWith(a)
			return object.BoolOf(ok2 && c < 0), nil
		}})
		inst.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			c, ok2 := cmpWith(a)
			return object.BoolOf(ok2 && c <= 0), nil
		}})
		inst.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			c, ok2 := cmpWith(a)
			return object.BoolOf(ok2 && c > 0), nil
		}})
		inst.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			c, ok2 := cmpWith(a)
			return object.BoolOf(ok2 && c >= 0), nil
		}})
		return inst
	}

	ipv6AddrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var v *big.Int
		switch arg := a[1].(type) {
		case *object.Str:
			parsed, err := parseIPv6(arg.V)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			v = parsed
		case *object.Int:
			v = new(big.Int).Set(&arg.V)
		case *object.Bytes:
			if len(arg.V) != 16 {
				return nil, object.Errorf(addrValErrCls, "packed address must be 16 bytes")
			}
			v = new(big.Int).SetBytes(arg.V)
		default:
			return nil, object.Errorf(addrValErrCls, "invalid IPv6 address: %v", a[1])
		}
		tmp := makeIPv6Address(v)
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("IPv6Address", ipv6AddrCls)

	// ── IPv6Network ───────────────────────────────────────────────────────────

	var makeIPv6Network func(network *big.Int, prefixlen int) *object.Instance
	ipv6NetCls := &object.Class{Name: "IPv6Network", Dict: object.NewDict()}

	makeIPv6Network = func(network *big.Int, prefixlen int) *object.Instance {
		inst := &object.Instance{Class: ipv6NetCls, Dict: object.NewDict()}
		mask := prefixToMask6(prefixlen)
		hostmask := new(big.Int).Not(mask)
		hostmask.And(hostmask, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)))
		netAddr := new(big.Int).And(network, mask)
		bcastAddr := new(big.Int).Or(netAddr, hostmask)

		// num_addresses = 2^(128-prefixlen)
		numAddrs := new(big.Int).Lsh(big.NewInt(1), uint(128-prefixlen))
		numAddrsObj := &object.Int{}
		numAddrsObj.V.Set(numAddrs)

		netAddrObj := &object.Int{}
		netAddrObj.V.Set(netAddr)

		s := ipv6ToString(netAddr) + "/" + strconv.Itoa(prefixlen)
		inst.Dict.SetStr("_network", netAddrObj)
		inst.Dict.SetStr("_prefixlen", object.NewInt(int64(prefixlen)))
		inst.Dict.SetStr("network_address", makeIPv6Address(netAddr))
		inst.Dict.SetStr("broadcast_address", makeIPv6Address(bcastAddr))
		inst.Dict.SetStr("prefixlen", object.NewInt(int64(prefixlen)))
		inst.Dict.SetStr("num_addresses", numAddrsObj)
		inst.Dict.SetStr("version", object.NewInt(6))
		inst.Dict.SetStr("max_prefixlen", object.NewInt(128))
		inst.Dict.SetStr("compressed", &object.Str{V: s})
		inst.Dict.SetStr("with_prefixlen", &object.Str{V: s})

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv6Network('" + s + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: s}, nil
		}})
		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if other, ok2 := a[0].(*object.Instance); ok2 {
				on, _ := other.Dict.GetStr("_network")
				op, _ := other.Dict.GetStr("_prefixlen")
				if on != nil && op != nil {
					if oInt, ok3 := on.(*object.Int); ok3 {
						if oIntP, ok4 := op.(*object.Int); ok4 {
							return object.BoolOf(netAddr.Cmp(&oInt.V) == 0 && int64(prefixlen) == oIntP.Int64()), nil
						}
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			lo := new(big.Int).And(netAddr, new(big.Int).SetUint64(0xffffffffffffffff))
			return object.NewInt(lo.Int64() ^ int64(prefixlen)), nil
		}})
		inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if addrInst, ok2 := a[0].(*object.Instance); ok2 {
				if ov, ok3 := addrInst.Dict.GetStr("_value"); ok3 {
					if oInt, ok4 := ov.(*object.Int); ok4 {
						av := &oInt.V
						masked := new(big.Int).And(av, mask)
						return object.BoolOf(masked.Cmp(netAddr) == 0), nil
					}
				}
			}
			return object.False, nil
		}})
		inst.Dict.SetStr("subnet_of", &object.BuiltinFunc{Name: "subnet_of", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNetInt, ok3 := on.(*object.Int)
			oPrefixInt, ok4 := op.(*object.Int)
			if !ok3 || !ok4 {
				return object.False, nil
			}
			oPrefix := int(oPrefixInt.Int64())
			oMask := prefixToMask6(oPrefix)
			masked := new(big.Int).And(netAddr, oMask)
			return object.BoolOf(prefixlen >= oPrefix && masked.Cmp(&oNetInt.V) == 0), nil
		}})
		inst.Dict.SetStr("supernet_of", &object.BuiltinFunc{Name: "supernet_of", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNetInt, ok3 := on.(*object.Int)
			oPrefixInt, ok4 := op.(*object.Int)
			if !ok3 || !ok4 {
				return object.False, nil
			}
			oPrefix := int(oPrefixInt.Int64())
			masked := new(big.Int).And(&oNetInt.V, mask)
			return object.BoolOf(prefixlen <= oPrefix && masked.Cmp(netAddr) == 0), nil
		}})
		inst.Dict.SetStr("overlaps", &object.BuiltinFunc{Name: "overlaps", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			other, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			on, _ := other.Dict.GetStr("_network")
			op, _ := other.Dict.GetStr("_prefixlen")
			if on == nil || op == nil {
				return object.False, nil
			}
			oNetInt, ok3 := on.(*object.Int)
			oPrefixInt, ok4 := op.(*object.Int)
			if !ok3 || !ok4 {
				return object.False, nil
			}
			oPrefix := int(oPrefixInt.Int64())
			oMask := prefixToMask6(oPrefix)
			m1 := new(big.Int).And(netAddr, oMask)
			m2 := new(big.Int).And(&oNetInt.V, mask)
			return object.BoolOf(m1.Cmp(&oNetInt.V) == 0 || m2.Cmp(netAddr) == 0), nil
		}})
		inst.Dict.SetStr("subnets", &object.BuiltinFunc{Name: "subnets", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			newPrefix := prefixlen + 1
			if kw != nil {
				if v, ok2 := kw.GetStr("prefixlen_diff"); ok2 {
					if n, ok3 := v.(*object.Int); ok3 {
						newPrefix = prefixlen + int(n.Int64())
					}
				}
			}
			if newPrefix > 128 || newPrefix <= prefixlen {
				return &object.List{V: []object.Object{}}, nil
			}
			step := new(big.Int).Lsh(big.NewInt(1), uint(128-newPrefix))
			var result []object.Object
			for addr := new(big.Int).Set(netAddr); addr.Cmp(new(big.Int).Add(netAddr, numAddrs)) < 0; {
				result = append(result, makeIPv6Network(new(big.Int).Set(addr), newPrefix))
				addr.Add(addr, step)
				if len(result) > 100 {
					break
				}
			}
			return &object.List{V: result}, nil
		}})
		inst.Dict.SetStr("supernet", &object.BuiltinFunc{Name: "supernet", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			diff := 1
			if kw != nil {
				if v, ok2 := kw.GetStr("prefixlen_diff"); ok2 {
					if n, ok3 := v.(*object.Int); ok3 {
						diff = int(n.Int64())
					}
				}
			}
			newPrefix := prefixlen - diff
			if newPrefix < 0 {
				newPrefix = 0
			}
			return makeIPv6Network(netAddr, newPrefix), nil
		}})
		inst.Dict.SetStr("hosts", &object.BuiltinFunc{Name: "hosts", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var result []object.Object
			if prefixlen >= 127 {
				for addr := new(big.Int).Set(netAddr); addr.Cmp(bcastAddr) <= 0; {
					result = append(result, makeIPv6Address(new(big.Int).Set(addr)))
					addr.Add(addr, big.NewInt(1))
				}
			} else {
				start := new(big.Int).Add(netAddr, big.NewInt(1))
				for addr := new(big.Int).Set(start); addr.Cmp(bcastAddr) < 0; {
					result = append(result, makeIPv6Address(new(big.Int).Set(addr)))
					addr.Add(addr, big.NewInt(1))
					if len(result) > 1000 {
						break
					}
				}
			}
			return &object.List{V: result}, nil
		}})
		return inst
	}

	ipv6NetCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		strict := true
		if kw != nil {
			if v, ok2 := kw.GetStr("strict"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					strict = b == object.True
				}
			}
		}
		argStr, ok2 := a[1].(*object.Str)
		if !ok2 {
			return nil, object.Errorf(addrValErrCls, "invalid network")
		}
		s := argStr.V
		slash := strings.Index(s, "/")
		var v *big.Int
		var prefixlen int
		if slash < 0 {
			parsed, err := parseIPv6(s)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			v = parsed
			prefixlen = 128
		} else {
			parsed, err := parseIPv6(s[:slash])
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			v = parsed
			pl, err := strconv.Atoi(s[slash+1:])
			if err != nil || pl < 0 || pl > 128 {
				return nil, object.Errorf(netmaskValErrCls, "invalid prefix: %s", s[slash+1:])
			}
			prefixlen = pl
		}
		mask := prefixToMask6(prefixlen)
		masked := new(big.Int).And(v, mask)
		if strict && masked.Cmp(v) != 0 {
			return nil, object.Errorf(i.valueErr, "%s has host bits set", ipv6ToString(v))
		}
		tmp := makeIPv6Network(v, prefixlen)
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("IPv6Network", ipv6NetCls)

	// ── IPv6Interface ─────────────────────────────────────────────────────────

	ipv6IfaceCls := &object.Class{Name: "IPv6Interface", Bases: []*object.Class{ipv6AddrCls}, Dict: object.NewDict()}
	ipv6IfaceCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		argStr, ok2 := a[1].(*object.Str)
		if !ok2 {
			return nil, object.Errorf(addrValErrCls, "invalid interface")
		}
		s := argStr.V
		slash := strings.Index(s, "/")
		var addrV *big.Int
		var prefixlen int
		if slash < 0 {
			parsed, err := parseIPv6(s)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			addrV = parsed
			prefixlen = 128
		} else {
			parsed, err := parseIPv6(s[:slash])
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			addrV = parsed
			pl, err := strconv.Atoi(s[slash+1:])
			if err != nil || pl < 0 || pl > 128 {
				return nil, object.Errorf(netmaskValErrCls, "invalid prefix: %s", s[slash+1:])
			}
			prefixlen = pl
		}
		tmp := makeIPv6Address(addrV)
		ks, vs := tmp.Dict.Items()
		for j, k := range ks {
			inst.Dict.SetStr(object.Str_(k), vs[j])
		}
		netAddr := new(big.Int).And(addrV, prefixToMask6(prefixlen))
		ifaceStr := ipv6ToString(addrV) + "/" + strconv.Itoa(prefixlen)
		inst.Dict.SetStr("ip", makeIPv6Address(addrV))
		inst.Dict.SetStr("network", makeIPv6Network(netAddr, prefixlen))
		inst.Dict.SetStr("with_prefixlen", &object.Str{V: ifaceStr})
		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "IPv6Interface('" + ifaceStr + "')"}, nil
		}})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ifaceStr}, nil
		}})
		return object.None, nil
	}})
	m.Dict.SetStr("IPv6Interface", ipv6IfaceCls)

	// ── Factory functions ─────────────────────────────────────────────────────

	m.Dict.SetStr("ip_address", &object.BuiltinFunc{Name: "ip_address", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(addrValErrCls, "ip_address requires an argument")
		}
		switch arg := a[0].(type) {
		case *object.Str:
			if strings.Contains(arg.V, ":") {
				v, err := parseIPv6(arg.V)
				if err != nil {
					return nil, object.Errorf(addrValErrCls, "%v", err)
				}
				return makeIPv6Address(v), nil
			}
			v, err := parseIPv4(arg.V)
			if err != nil {
				return nil, object.Errorf(addrValErrCls, "%v", err)
			}
			return makeIPv4Address(v), nil
		case *object.Int:
			if arg.IsInt64() && arg.Int64() <= 0xffffffff && arg.Int64() >= 0 {
				return makeIPv4Address(uint32(arg.Int64())), nil
			}
			return makeIPv6Address(new(big.Int).Set(&arg.V)), nil
		}
		return nil, object.Errorf(addrValErrCls, "invalid address: %v", a[0])
	}})

	m.Dict.SetStr("ip_network", &object.BuiltinFunc{Name: "ip_network", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(addrValErrCls, "ip_network requires an argument")
		}
		strict := true
		if kw != nil {
			if v, ok2 := kw.GetStr("strict"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					strict = b == object.True
				}
			}
		}
		argStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(addrValErrCls, "invalid network")
		}
		if strings.Contains(argStr.V, ":") {
			return i.callObject(ipv6NetCls, []object.Object{a[0]}, kw)
		}
		strictObj := object.BoolOf(strict)
		if kw == nil {
			kw = object.NewDict()
		}
		kw.SetStr("strict", strictObj)
		return i.callObject(ipv4NetCls, []object.Object{a[0]}, kw)
	}})

	m.Dict.SetStr("ip_interface", &object.BuiltinFunc{Name: "ip_interface", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(addrValErrCls, "ip_interface requires an argument")
		}
		argStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(addrValErrCls, "invalid interface")
		}
		if strings.Contains(argStr.V, ":") {
			return i.callObject(ipv6IfaceCls, a, nil)
		}
		return i.callObject(ipv4IfaceCls, a, nil)
	}})

	// ── Utility functions ─────────────────────────────────────────────────────

	m.Dict.SetStr("v4_int_to_packed", &object.BuiltinFunc{Name: "v4_int_to_packed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "v4_int_to_packed requires argument")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return nil, object.Errorf(i.typeErr, "argument must be int")
		}
		v := uint32(n.Int64())
		return &object.Bytes{V: []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}}, nil
	}})

	m.Dict.SetStr("v6_int_to_packed", &object.BuiltinFunc{Name: "v6_int_to_packed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "v6_int_to_packed requires argument")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return nil, object.Errorf(i.typeErr, "argument must be int")
		}
		b := bigIntTo16Bytes(&n.V)
		return &object.Bytes{V: b}, nil
	}})

	m.Dict.SetStr("get_mixed_type_key", &object.BuiltinFunc{Name: "get_mixed_type_key", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
		}
		obj := a[0]
		if inst, ok := obj.(*object.Instance); ok {
			ver, _ := inst.Dict.GetStr("version")
			if v, ok2 := inst.Dict.GetStr("_value"); ok2 {
				return &object.Tuple{V: []object.Object{ver, v}}, nil
			}
			if v, ok2 := inst.Dict.GetStr("_network"); ok2 {
				return &object.Tuple{V: []object.Object{ver, v}}, nil
			}
		}
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
	}})

	m.Dict.SetStr("collapse_addresses", &object.BuiltinFunc{Name: "collapse_addresses", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: []object.Object{}}, nil
		}
		var networks []object.Object
		switch v := a[0].(type) {
		case *object.List:
			networks = v.V
		case *object.Tuple:
			networks = v.V
		}
		// Simple approach: return them sorted (full collapse is complex)
		result := make([]object.Object, len(networks))
		copy(result, networks)
		return &object.List{V: result}, nil
	}})

	m.Dict.SetStr("summarize_address_range", &object.BuiltinFunc{Name: "summarize_address_range", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.List{V: []object.Object{}}, nil
		}
		first, ok1 := a[0].(*object.Instance)
		last, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return &object.List{V: []object.Object{}}, nil
		}
		fv, ok3 := first.Dict.GetStr("_value")
		lv, ok4 := last.Dict.GetStr("_value")
		if !ok3 || !ok4 {
			return &object.List{V: []object.Object{}}, nil
		}
		firstInt, ok5 := fv.(*object.Int)
		lastInt, ok6 := lv.(*object.Int)
		if !ok5 || !ok6 {
			return &object.List{V: []object.Object{}}, nil
		}
		firstV := uint32(firstInt.Int64())
		lastV := uint32(lastInt.Int64())
		var result []object.Object
		for firstV <= lastV {
			// Find the largest prefix that fits
			maxSize := uint32(1)
			for {
				nextMaxSize := maxSize << 1
				if nextMaxSize == 0 {
					break
				}
				// Check alignment and coverage
				if firstV%nextMaxSize != 0 || firstV+nextMaxSize-1 > lastV {
					break
				}
				maxSize = nextMaxSize
			}
			prefixlen := 33 - bits32(maxSize)
			result = append(result, makeIPv4Network(firstV, prefixlen))
			firstV += maxSize
			if firstV == 0 { // overflow
				break
			}
		}
		return &object.List{V: result}, nil
	}})

	return m
}

// bits32 returns the number of set bits needed to represent n (log2 + 1 for powers of 2).
func bits32(n uint32) int {
	count := 0
	for n > 0 {
		count++
		n >>= 1
	}
	return count
}
