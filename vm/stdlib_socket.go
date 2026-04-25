package vm

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tamnd/goipy/object"
)

// sockStateRegistry maps Python socket instances to their Go state so that
// other modules (e.g., ssl, select) can access the underlying net.Conn.
var sockStateRegistry sync.Map // *object.Instance -> *sockState

// connFd returns the raw OS file descriptor from a net.Conn or net.Listener
// via SyscallConn().Control(). Does not dup the fd or change blocking mode.
func connFd(c interface{}) int {
	type rawConner interface {
		SyscallConn() (syscall.RawConn, error)
	}
	if sc, ok := c.(rawConner); ok {
		if rc, err := sc.SyscallConn(); err == nil {
			fd := -1
			_ = rc.Control(func(f uintptr) { fd = int(f) })
			return fd
		}
	}
	return -1
}

// sockStateOf returns the sockState for a Python socket instance, or nil.
func sockStateOf(obj object.Object) *sockState {
	if inst, ok := obj.(*object.Instance); ok {
		if v, ok2 := sockStateRegistry.Load(inst); ok2 {
			return v.(*sockState)
		}
	}
	return nil
}

// sockState holds the Go-level state for a Python socket instance.
type sockState struct {
	mu       sync.RWMutex
	family   int
	socktype int
	proto    int
	listener net.Listener // TCP server: set at bind()
	conn     net.Conn     // TCP connected/accepted socket
	udpConn  *net.UDPConn // UDP socket set at bind()
	bindAddr string       // address passed to bind()
	bound    bool
	timeout  float64             // -1 = None (blocking), >=0 = seconds
	closed   bool
	opts     map[int]map[int]int // level -> optname -> value
}

func newSockState(family, socktype, proto int) *sockState {
	return &sockState{
		family:   family,
		socktype: socktype,
		proto:    proto,
		timeout:  -1,
		opts:     make(map[int]map[int]int),
	}
}

func (st *sockState) setOpt(level, name, val int) {
	if st.opts[level] == nil {
		st.opts[level] = make(map[int]int)
	}
	st.opts[level][name] = val
}

func (st *sockState) getOpt(level, name int) int {
	if m, ok := st.opts[level]; ok {
		return m[name]
	}
	return 0
}

func (st *sockState) deadline() time.Time {
	if st.timeout > 0 {
		return time.Now().Add(time.Duration(st.timeout * float64(time.Second)))
	}
	return time.Time{}
}

func (st *sockState) applyDeadline() {
	dl := st.deadline()
	if !dl.IsZero() {
		if st.conn != nil {
			st.conn.SetDeadline(dl) //nolint
		}
		if st.udpConn != nil {
			st.udpConn.SetDeadline(dl) //nolint
		}
	}
}

// parseAddr converts a Python address tuple/string to a Go "host:port" string.
func parseAddr(addrObj object.Object) (string, error) {
	switch a := addrObj.(type) {
	case *object.Tuple:
		if len(a.V) >= 2 {
			host := ""
			if s, ok := a.V[0].(*object.Str); ok {
				host = s.V
			}
			port, _ := toInt64(a.V[1])
			return fmt.Sprintf("%s:%d", host, port), nil
		}
	case *object.Str:
		return a.V, nil
	}
	return "", fmt.Errorf("invalid address")
}

// netAddrToTuple converts a Go net.Addr to a Python (host, port) tuple.
func netAddrToTuple(addr net.Addr) object.Object {
	if addr == nil {
		return &object.Tuple{V: []object.Object{&object.Str{V: ""}, object.NewInt(0)}}
	}
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return &object.Tuple{V: []object.Object{&object.Str{V: addr.String()}, object.NewInt(0)}}
	}
	port, _ := strconv.Atoi(portStr)
	return &object.Tuple{V: []object.Object{&object.Str{V: host}, object.NewInt(int64(port))}}
}

// makeSocketInst wraps an existing sockState into a Python socket instance.
func (i *Interp) makeSocketInst(sockCls *object.Class, st *sockState, socketErrCls *object.Class) *object.Instance {
	inst := &object.Instance{Class: sockCls, Dict: object.NewDict()}
	sockStateRegistry.Store(inst, st)
	inst.Dict.SetStr("family", object.NewInt(int64(st.family)))
	inst.Dict.SetStr("type", object.NewInt(int64(st.socktype)))
	inst.Dict.SetStr("proto", object.NewInt(int64(st.proto)))

	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			st.closed = true
			if st.listener != nil {
				st.listener.Close() //nolint
				st.listener = nil
			}
			if st.conn != nil {
				st.conn.Close() //nolint
				st.conn = nil
			}
			if st.udpConn != nil {
				st.udpConn.Close() //nolint
				st.udpConn = nil
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("bind", &object.BuiltinFunc{Name: "bind",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "bind() requires an address")
			}
			addr, err := parseAddr(a[0])
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			st.bindAddr = addr
			if st.socktype == syscall.SOCK_DGRAM {
				udpAddr, uerr := net.ResolveUDPAddr("udp", addr)
				if uerr != nil {
					return nil, object.Errorf(socketErrCls, "%v", uerr)
				}
				uc, uerr := net.ListenUDP("udp", udpAddr)
				if uerr != nil {
					return nil, object.Errorf(socketErrCls, "%v", uerr)
				}
				st.udpConn = uc
			} else {
				ln, lerr := net.Listen("tcp", addr)
				if lerr != nil {
					return nil, object.Errorf(socketErrCls, "%v", lerr)
				}
				st.listener = ln
			}
			st.bound = true
			return object.None, nil
		}})

	inst.Dict.SetStr("listen", &object.BuiltinFunc{Name: "listen",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Listener already created in bind(); this is a no-op.
			return object.None, nil
		}})

	inst.Dict.SetStr("accept", &object.BuiltinFunc{Name: "accept",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.RLock()
			ln := st.listener
			st.mu.RUnlock()
			if ln == nil {
				return nil, object.Errorf(socketErrCls, "socket not listening")
			}
			conn, err := ln.Accept()
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			newSt := newSockState(st.family, st.socktype, st.proto)
			newSt.conn = conn
			newInst := i.makeSocketInst(sockCls, newSt, socketErrCls)
			addr := netAddrToTuple(conn.RemoteAddr())
			return &object.Tuple{V: []object.Object{newInst, addr}}, nil
		}})

	connect := func(addrObj object.Object) error {
		addr, err := parseAddr(addrObj)
		if err != nil {
			return object.Errorf(socketErrCls, "%v", err)
		}
		var conn net.Conn
		if st.timeout > 0 {
			conn, err = net.DialTimeout("tcp", addr, time.Duration(st.timeout*float64(time.Second)))
		} else {
			conn, err = net.Dial("tcp", addr)
		}
		if err != nil {
			return object.Errorf(socketErrCls, "%v", err)
		}
		st.mu.Lock()
		st.conn = conn
		st.mu.Unlock()
		return nil
	}

	inst.Dict.SetStr("connect", &object.BuiltinFunc{Name: "connect",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "connect() requires an address")
			}
			if err := connect(a[0]); err != nil {
				return nil, err
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("connect_ex", &object.BuiltinFunc{Name: "connect_ex",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "connect_ex() requires an address")
			}
			if err := connect(a[0]); err != nil {
				return object.NewInt(int64(syscall.ECONNREFUSED)), nil
			}
			return object.NewInt(0), nil
		}})

	inst.Dict.SetStr("send", &object.BuiltinFunc{Name: "send",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "send() requires data")
			}
			data, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "send() argument must be bytes")
			}
			st.mu.RLock()
			conn := st.conn
			st.mu.RUnlock()
			if conn == nil {
				return nil, object.Errorf(socketErrCls, "socket not connected")
			}
			st.applyDeadline()
			n, err := conn.Write(data.V)
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			return object.NewInt(int64(n)), nil
		}})

	inst.Dict.SetStr("sendall", &object.BuiltinFunc{Name: "sendall",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "sendall() requires data")
			}
			data, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "sendall() argument must be bytes")
			}
			st.mu.RLock()
			conn := st.conn
			st.mu.RUnlock()
			if conn == nil {
				return nil, object.Errorf(socketErrCls, "socket not connected")
			}
			st.applyDeadline()
			buf := data.V
			for len(buf) > 0 {
				n, err := conn.Write(buf)
				if err != nil {
					return nil, object.Errorf(socketErrCls, "%v", err)
				}
				buf = buf[n:]
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("recv", &object.BuiltinFunc{Name: "recv",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "recv() requires bufsize")
			}
			bufsize, ok := toInt64(a[0])
			if !ok || bufsize <= 0 {
				bufsize = 4096
			}
			st.mu.RLock()
			conn := st.conn
			st.mu.RUnlock()
			if conn == nil {
				return nil, object.Errorf(socketErrCls, "socket not connected")
			}
			st.applyDeadline()
			buf := make([]byte, bufsize)
			n, err := conn.Read(buf)
			if err != nil {
				if n == 0 {
					return &object.Bytes{V: []byte{}}, nil
				}
			}
			return &object.Bytes{V: buf[:n]}, nil
		}})

	inst.Dict.SetStr("sendto", &object.BuiltinFunc{Name: "sendto",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "sendto() requires data and address")
			}
			data, ok := a[0].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "sendto() data must be bytes")
			}
			addrStr, err := parseAddr(a[1])
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			st.mu.RLock()
			uc := st.udpConn
			st.mu.RUnlock()
			if uc != nil {
				udpAddr, uerr := net.ResolveUDPAddr("udp", addrStr)
				if uerr != nil {
					return nil, object.Errorf(socketErrCls, "%v", uerr)
				}
				n, uerr := uc.WriteToUDP(data.V, udpAddr)
				if uerr != nil {
					return nil, object.Errorf(socketErrCls, "%v", uerr)
				}
				return object.NewInt(int64(n)), nil
			}
			// Unbound UDP: use net.Dial
			conn, derr := net.Dial("udp", addrStr)
			if derr != nil {
				return nil, object.Errorf(socketErrCls, "%v", derr)
			}
			defer conn.Close()
			n, derr := conn.Write(data.V)
			if derr != nil {
				return nil, object.Errorf(socketErrCls, "%v", derr)
			}
			return object.NewInt(int64(n)), nil
		}})

	inst.Dict.SetStr("recvfrom", &object.BuiltinFunc{Name: "recvfrom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "recvfrom() requires bufsize")
			}
			bufsize, ok := toInt64(a[0])
			if !ok || bufsize <= 0 {
				bufsize = 4096
			}
			st.mu.RLock()
			uc := st.udpConn
			conn := st.conn
			st.mu.RUnlock()
			buf := make([]byte, bufsize)
			if uc != nil {
				st.applyDeadline()
				n, addr, err := uc.ReadFromUDP(buf)
				if err != nil {
					return nil, object.Errorf(socketErrCls, "%v", err)
				}
				addrTuple := &object.Tuple{V: []object.Object{
					&object.Str{V: addr.IP.String()},
					object.NewInt(int64(addr.Port)),
				}}
				return &object.Tuple{V: []object.Object{&object.Bytes{V: buf[:n]}, addrTuple}}, nil
			}
			if conn != nil {
				st.applyDeadline()
				n, err := conn.Read(buf)
				if err != nil && n == 0 {
					return nil, object.Errorf(socketErrCls, "%v", err)
				}
				addr := netAddrToTuple(conn.RemoteAddr())
				return &object.Tuple{V: []object.Object{&object.Bytes{V: buf[:n]}, addr}}, nil
			}
			return nil, object.Errorf(socketErrCls, "socket not bound")
		}})

	inst.Dict.SetStr("getsockname", &object.BuiltinFunc{Name: "getsockname",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.RLock()
			defer st.mu.RUnlock()
			if st.listener != nil {
				return netAddrToTuple(st.listener.Addr()), nil
			}
			if st.conn != nil {
				return netAddrToTuple(st.conn.LocalAddr()), nil
			}
			if st.udpConn != nil {
				return netAddrToTuple(st.udpConn.LocalAddr()), nil
			}
			return &object.Tuple{V: []object.Object{&object.Str{V: ""}, object.NewInt(0)}}, nil
		}})

	inst.Dict.SetStr("getpeername", &object.BuiltinFunc{Name: "getpeername",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.RLock()
			conn := st.conn
			st.mu.RUnlock()
			if conn == nil {
				return nil, object.Errorf(socketErrCls, "socket not connected")
			}
			return netAddrToTuple(conn.RemoteAddr()), nil
		}})

	inst.Dict.SetStr("setsockopt", &object.BuiltinFunc{Name: "setsockopt",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "setsockopt() requires level, optname, value")
			}
			level, _ := toInt64(a[0])
			name, _ := toInt64(a[1])
			val, _ := toInt64(a[2])
			st.setOpt(int(level), int(name), int(val))
			return object.None, nil
		}})

	inst.Dict.SetStr("getsockopt", &object.BuiltinFunc{Name: "getsockopt",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "getsockopt() requires level, optname")
			}
			level, _ := toInt64(a[0])
			name, _ := toInt64(a[1])
			// If buflen arg given, return bytes
			if len(a) >= 3 {
				buflen, _ := toInt64(a[2])
				buf := make([]byte, buflen)
				val := st.getOpt(int(level), int(name))
				if buflen >= 4 {
					binary.LittleEndian.PutUint32(buf, uint32(val))
				}
				return &object.Bytes{V: buf}, nil
			}
			return object.NewInt(int64(st.getOpt(int(level), int(name)))), nil
		}})

	inst.Dict.SetStr("setblocking", &object.BuiltinFunc{Name: "setblocking",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 && object.Truthy(a[0]) {
				st.timeout = -1 // blocking
			} else {
				st.timeout = 0 // non-blocking
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("getblocking", &object.BuiltinFunc{Name: "getblocking",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// blocking if timeout is None (-1) or >0 (timeout mode counts as blocking)
			return object.BoolOf(st.timeout != 0), nil
		}})

	inst.Dict.SetStr("settimeout", &object.BuiltinFunc{Name: "settimeout",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 || a[0] == object.None {
				st.timeout = -1
			} else if f, ok := toFloat64(a[0]); ok {
				st.timeout = f
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("gettimeout", &object.BuiltinFunc{Name: "gettimeout",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if st.timeout < 0 {
				return object.None, nil
			}
			return &object.Float{V: st.timeout}, nil
		}})

	inst.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.conn != nil {
				st.conn.Close() //nolint
				st.conn = nil
			}
			return object.None, nil
		}})

	inst.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.RLock()
			conn, ln := st.conn, st.listener
			st.mu.RUnlock()
			if conn != nil {
				if fd := connFd(conn); fd >= 0 {
					return object.NewInt(int64(fd)), nil
				}
			}
			if ln != nil {
				if fd := connFd(ln); fd >= 0 {
					return object.NewInt(int64(fd)), nil
				}
			}
			return object.NewInt(-1), nil
		}})

	inst.Dict.SetStr("makefile", &object.BuiltinFunc{Name: "makefile",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("detach", &object.BuiltinFunc{Name: "detach",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(-1), nil
		}})

	inst.Dict.SetStr("dup", &object.BuiltinFunc{Name: "dup",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			st.closed = true
			if st.listener != nil {
				st.listener.Close() //nolint
			}
			if st.conn != nil {
				st.conn.Close() //nolint
			}
			if st.udpConn != nil {
				st.udpConn.Close() //nolint
			}
			return object.False, nil
		}})

	return inst
}

// buildSocket constructs the socket module.
func (i *Interp) buildSocket() *object.Module {
	m := &object.Module{Name: "socket", Dict: object.NewDict()}

	// ── exceptions ────────────────────────────────────────────────────────

	// socket.error = OSError
	socketErrCls := i.osErr
	m.Dict.SetStr("error", socketErrCls)

	herrorCls := &object.Class{
		Name:  "herror",
		Dict:  object.NewDict(),
		Bases: []*object.Class{socketErrCls},
	}
	m.Dict.SetStr("herror", herrorCls)

	gaierrorCls := &object.Class{
		Name:  "gaierror",
		Dict:  object.NewDict(),
		Bases: []*object.Class{socketErrCls},
	}
	m.Dict.SetStr("gaierror", gaierrorCls)

	timeoutCls := &object.Class{
		Name:  "timeout",
		Dict:  object.NewDict(),
		Bases: []*object.Class{socketErrCls},
	}
	m.Dict.SetStr("timeout", timeoutCls)

	// ── socket class ──────────────────────────────────────────────────────

	sockCls := &object.Class{Name: "socket", Dict: object.NewDict()}

	// ── constants ─────────────────────────────────────────────────────────

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// Address families
	setInt("AF_INET", int(syscall.AF_INET))
	setInt("AF_INET6", int(syscall.AF_INET6))
	setInt("AF_UNIX", int(syscall.AF_UNIX))
	setInt("AF_UNSPEC", int(syscall.AF_UNSPEC))

	// Socket types
	setInt("SOCK_STREAM", int(syscall.SOCK_STREAM))
	setInt("SOCK_DGRAM", int(syscall.SOCK_DGRAM))
	setInt("SOCK_RAW", int(syscall.SOCK_RAW))
	setInt("SOCK_SEQPACKET", int(syscall.SOCK_SEQPACKET))

	// Protocols
	setInt("IPPROTO_TCP", int(syscall.IPPROTO_TCP))
	setInt("IPPROTO_UDP", int(syscall.IPPROTO_UDP))
	setInt("IPPROTO_IP", int(syscall.IPPROTO_IP))
	setInt("IPPROTO_IPV6", int(syscall.IPPROTO_IPV6))
	setInt("IPPROTO_ICMP", int(syscall.IPPROTO_ICMP))
	setInt("IPPROTO_RAW", int(syscall.IPPROTO_RAW))

	// Socket-level options
	setInt("SOL_SOCKET", int(syscall.SOL_SOCKET))
	setInt("SO_REUSEADDR", int(syscall.SO_REUSEADDR))
	setInt("SO_KEEPALIVE", int(syscall.SO_KEEPALIVE))
	setInt("SO_BROADCAST", int(syscall.SO_BROADCAST))
	setInt("SO_SNDBUF", int(syscall.SO_SNDBUF))
	setInt("SO_RCVBUF", int(syscall.SO_RCVBUF))
	setInt("SO_ERROR", int(syscall.SO_ERROR))
	setInt("SO_TYPE", int(syscall.SO_TYPE))
	setInt("SO_LINGER", int(syscall.SO_LINGER))

	// TCP options
	setInt("IPPROTO_TCP", int(syscall.IPPROTO_TCP))
	setInt("TCP_NODELAY", int(syscall.TCP_NODELAY))

	// IP options
	setInt("IP_TTL", int(syscall.IP_TTL))
	setInt("IP_HDRINCL", int(syscall.IP_HDRINCL))
	setInt("IP_MULTICAST_TTL", int(syscall.IP_MULTICAST_TTL))
	setInt("IP_MULTICAST_LOOP", int(syscall.IP_MULTICAST_LOOP))
	setInt("IP_ADD_MEMBERSHIP", int(syscall.IP_ADD_MEMBERSHIP))

	// Shutdown flags
	m.Dict.SetStr("SHUT_RD", object.NewInt(0))
	m.Dict.SetStr("SHUT_WR", object.NewInt(1))
	m.Dict.SetStr("SHUT_RDWR", object.NewInt(2))

	// Address constants
	m.Dict.SetStr("INADDR_ANY", object.NewInt(0))
	m.Dict.SetStr("INADDR_LOOPBACK", object.NewInt(0x7f000001))
	m.Dict.SetStr("INADDR_BROADCAST", object.NewInt(int64(0xffffffff)))
	m.Dict.SetStr("INADDR_NONE", object.NewInt(int64(0xffffffff)))

	// getaddrinfo / getnameinfo flags (POSIX standard values)
	setInt("AI_PASSIVE", 0x00000001)
	setInt("AI_CANONNAME", 0x00000002)
	setInt("AI_NUMERICHOST", 0x00000004)
	setInt("AI_NUMERICSERV", 0x00000008)
	setInt("NI_NUMERICHOST", 0x00000002)
	setInt("NI_NAMEREQD", 0x00000004)
	setInt("NI_NUMERICSERV", 0x00000008)

	// Message flags
	setInt("MSG_OOB", int(syscall.MSG_OOB))
	setInt("MSG_DONTWAIT", int(syscall.MSG_DONTWAIT))
	setInt("MSG_WAITALL", int(syscall.MSG_WAITALL))
	setInt("MSG_PEEK", int(syscall.MSG_PEEK))

	// ── default timeout ───────────────────────────────────────────────────

	var defaultTimeout float64 = -1 // -1 = None

	m.Dict.SetStr("setdefaulttimeout", &object.BuiltinFunc{Name: "setdefaulttimeout",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 || a[0] == object.None {
				defaultTimeout = -1
			} else if f, ok := toFloat64(a[0]); ok {
				defaultTimeout = f
			}
			return object.None, nil
		}})

	m.Dict.SetStr("getdefaulttimeout", &object.BuiltinFunc{Name: "getdefaulttimeout",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if defaultTimeout < 0 {
				return object.None, nil
			}
			return &object.Float{V: defaultTimeout}, nil
		}})

	// ── socket constructor ────────────────────────────────────────────────

	m.Dict.SetStr("socket", &object.BuiltinFunc{Name: "socket",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			family := int(syscall.AF_INET)
			socktype := int(syscall.SOCK_STREAM)
			proto := 0

			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					family = int(n)
				}
			}
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					socktype = int(n)
				}
			}
			if len(a) > 2 {
				if n, ok := toInt64(a[2]); ok {
					proto = int(n)
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("family"); ok {
					if n, ok2 := toInt64(v); ok2 {
						family = int(n)
					}
				}
				if v, ok := kw.GetStr("type"); ok {
					if n, ok2 := toInt64(v); ok2 {
						socktype = int(n)
					}
				}
				if v, ok := kw.GetStr("proto"); ok {
					if n, ok2 := toInt64(v); ok2 {
						proto = int(n)
					}
				}
			}

			st := newSockState(family, socktype, proto)
			if defaultTimeout >= 0 {
				st.timeout = defaultTimeout
			}
			return i.makeSocketInst(sockCls, st, socketErrCls), nil
		}})

	// ── socketpair ────────────────────────────────────────────────────────

	m.Dict.SetStr("socketpair", &object.BuiltinFunc{Name: "socketpair",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			family := int(syscall.AF_UNIX)
			socktype := int(syscall.SOCK_STREAM)
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					family = int(n)
				}
			}
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					socktype = int(n)
				}
			}
			// Use real OS socket pair so the kernel buffer allows non-blocking sends.
			fds, err := syscall.Socketpair(family, socktype, 0)
			if err != nil {
				return nil, object.Errorf(socketErrCls, "socketpair: %v", err)
			}
			f1 := os.NewFile(uintptr(fds[0]), "socketpair[0]")
			c1, err := net.FileConn(f1)
			f1.Close() //nolint
			if err != nil {
				return nil, object.Errorf(socketErrCls, "socketpair: %v", err)
			}
			f2 := os.NewFile(uintptr(fds[1]), "socketpair[1]")
			c2, err := net.FileConn(f2)
			f2.Close() //nolint
			if err != nil {
				c1.Close() //nolint
				return nil, object.Errorf(socketErrCls, "socketpair: %v", err)
			}
			st1 := newSockState(family, socktype, 0)
			st1.conn = c1
			st2 := newSockState(family, socktype, 0)
			st2.conn = c2
			inst1 := i.makeSocketInst(sockCls, st1, socketErrCls)
			inst2 := i.makeSocketInst(sockCls, st2, socketErrCls)
			return &object.Tuple{V: []object.Object{inst1, inst2}}, nil
		}})

	// ── create_connection ─────────────────────────────────────────────────

	m.Dict.SetStr("create_connection", &object.BuiltinFunc{Name: "create_connection",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "create_connection() requires address")
			}
			addrStr, err := parseAddr(a[0])
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			timeout := time.Duration(0)
			if len(a) > 1 && a[1] != object.None {
				if f, ok := toFloat64(a[1]); ok && f > 0 {
					timeout = time.Duration(f * float64(time.Second))
				}
			}
			var conn net.Conn
			if timeout > 0 {
				conn, err = net.DialTimeout("tcp", addrStr, timeout)
			} else {
				conn, err = net.Dial("tcp", addrStr)
			}
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			st := newSockState(int(syscall.AF_INET), int(syscall.SOCK_STREAM), int(syscall.IPPROTO_TCP))
			st.conn = conn
			return i.makeSocketInst(sockCls, st, socketErrCls), nil
		}})

	// ── create_server ─────────────────────────────────────────────────────

	m.Dict.SetStr("create_server", &object.BuiltinFunc{Name: "create_server",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "create_server() requires address")
			}
			addrStr, err := parseAddr(a[0])
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			ln, err := net.Listen("tcp", addrStr)
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			st := newSockState(int(syscall.AF_INET), int(syscall.SOCK_STREAM), int(syscall.IPPROTO_TCP))
			st.listener = ln
			st.bound = true
			return i.makeSocketInst(sockCls, st, socketErrCls), nil
		}})

	// ── getaddrinfo ───────────────────────────────────────────────────────

	m.Dict.SetStr("getaddrinfo", &object.BuiltinFunc{Name: "getaddrinfo",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "getaddrinfo() requires host and port")
			}
			host := ""
			if s, ok := a[0].(*object.Str); ok {
				host = s.V
			}
			port, _ := toInt64(a[1])

			// Resolve addresses.
			var addrs []net.IPAddr
			if net.ParseIP(host) != nil {
				ip := net.ParseIP(host)
				addrs = []net.IPAddr{{IP: ip}}
			} else {
				var err error
				addrs, err = net.DefaultResolver.LookupIPAddr(nil, host)
				if err != nil {
					return nil, object.Errorf(gaierrorCls, "%v", err)
				}
			}

			var results []object.Object
			for _, addr := range addrs {
				var family int
				var addrTuple object.Object
				if addr.IP.To4() != nil {
					family = int(syscall.AF_INET)
					addrTuple = &object.Tuple{V: []object.Object{
						&object.Str{V: addr.IP.String()},
						object.NewInt(port),
					}}
				} else {
					family = int(syscall.AF_INET6)
					addrTuple = &object.Tuple{V: []object.Object{
						&object.Str{V: addr.IP.String()},
						object.NewInt(port),
						object.NewInt(0),
						object.NewInt(0),
					}}
				}
				// (family, type, proto, canonname, sockaddr)
				entry := &object.Tuple{V: []object.Object{
					object.NewInt(int64(family)),
					object.NewInt(int64(syscall.SOCK_STREAM)),
					object.NewInt(int64(syscall.IPPROTO_TCP)),
					&object.Str{V: ""},
					addrTuple,
				}}
				results = append(results, entry)
			}
			return &object.List{V: results}, nil
		}})

	// ── gethostname ───────────────────────────────────────────────────────

	m.Dict.SetStr("gethostname", &object.BuiltinFunc{Name: "gethostname",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			h, err := os.Hostname()
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			return &object.Str{V: h}, nil
		}})

	// ── gethostbyname ─────────────────────────────────────────────────────

	m.Dict.SetStr("gethostbyname", &object.BuiltinFunc{Name: "gethostbyname",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "gethostbyname() requires host")
			}
			host := ""
			if s, ok := a[0].(*object.Str); ok {
				host = s.V
			}
			if ip := net.ParseIP(host); ip != nil {
				if ip4 := ip.To4(); ip4 != nil {
					return &object.Str{V: ip4.String()}, nil
				}
				return &object.Str{V: ip.String()}, nil
			}
			addrs, err := net.LookupHost(host)
			if err != nil {
				return nil, object.Errorf(gaierrorCls, "%v", err)
			}
			if len(addrs) == 0 {
				return nil, object.Errorf(gaierrorCls, "no address for %s", host)
			}
			return &object.Str{V: addrs[0]}, nil
		}})

	// ── gethostbyaddr ─────────────────────────────────────────────────────

	m.Dict.SetStr("gethostbyaddr", &object.BuiltinFunc{Name: "gethostbyaddr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "gethostbyaddr() requires IP")
			}
			ip := ""
			if s, ok := a[0].(*object.Str); ok {
				ip = s.V
			}
			names, err := net.LookupAddr(ip)
			hostname := ip
			if err == nil && len(names) > 0 {
				hostname = strings.TrimSuffix(names[0], ".")
			}
			addrList := &object.List{V: []object.Object{&object.Str{V: ip}}}
			return &object.Tuple{V: []object.Object{
				&object.Str{V: hostname},
				&object.List{V: nil},
				addrList,
			}}, nil
		}})

	// ── getnameinfo ───────────────────────────────────────────────────────

	m.Dict.SetStr("getnameinfo", &object.BuiltinFunc{Name: "getnameinfo",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "getnameinfo() requires sockaddr")
			}
			host := ""
			port := int64(0)
			if t, ok := a[0].(*object.Tuple); ok && len(t.V) >= 2 {
				if s, ok2 := t.V[0].(*object.Str); ok2 {
					host = s.V
				}
				port, _ = toInt64(t.V[1])
			}
			names, err := net.LookupAddr(host)
			hostname := host
			if err == nil && len(names) > 0 {
				hostname = strings.TrimSuffix(names[0], ".")
			}
			return &object.Tuple{V: []object.Object{
				&object.Str{V: hostname},
				&object.Str{V: strconv.FormatInt(port, 10)},
			}}, nil
		}})

	// ── getprotobyname ────────────────────────────────────────────────────

	protoTable := map[string]int{
		"ip": 0, "icmp": 1, "igmp": 2, "tcp": 6, "udp": 17,
		"ipv6": 41, "ospf": 89, "sctp": 132,
	}

	m.Dict.SetStr("getprotobyname", &object.BuiltinFunc{Name: "getprotobyname",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getprotobyname() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = strings.ToLower(s.V)
			}
			if n, ok := protoTable[name]; ok {
				return object.NewInt(int64(n)), nil
			}
			return nil, object.Errorf(socketErrCls, "unknown protocol name: %s", name)
		}})

	// ── getservbyname / getservbyport ─────────────────────────────────────

	serviceByName := map[string]int{
		"ftp": 21, "ssh": 22, "telnet": 23, "smtp": 25, "dns": 53,
		"http": 80, "pop3": 110, "imap": 143, "https": 443, "smtps": 465,
		"ldap": 389, "ldaps": 636, "imaps": 993, "pop3s": 995,
		"ntp": 123, "snmp": 161, "bgp": 179, "mysql": 3306,
		"postgresql": 5432, "redis": 6379,
	}
	serviceByPort := make(map[int]string, len(serviceByName))
	for k, v := range serviceByName {
		serviceByPort[v] = k
	}

	m.Dict.SetStr("getservbyname", &object.BuiltinFunc{Name: "getservbyname",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getservbyname() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = strings.ToLower(s.V)
			}
			if n, ok := serviceByName[name]; ok {
				return object.NewInt(int64(n)), nil
			}
			return nil, object.Errorf(socketErrCls, "service/proto not found")
		}})

	m.Dict.SetStr("getservbyport", &object.BuiltinFunc{Name: "getservbyport",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getservbyport() requires port")
			}
			port, _ := toInt64(a[0])
			if s, ok := serviceByPort[int(port)]; ok {
				return &object.Str{V: s}, nil
			}
			return nil, object.Errorf(socketErrCls, "port/proto not found")
		}})

	// ── inet_aton / inet_ntoa ─────────────────────────────────────────────

	m.Dict.SetStr("inet_aton", &object.BuiltinFunc{Name: "inet_aton",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "inet_aton() requires ip string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "inet_aton() requires string")
			}
			ip := net.ParseIP(s.V)
			if ip == nil {
				return nil, object.Errorf(socketErrCls, "illegal IP address string passed to inet_aton")
			}
			ip4 := ip.To4()
			if ip4 == nil {
				return nil, object.Errorf(socketErrCls, "illegal IP address string passed to inet_aton")
			}
			return &object.Bytes{V: []byte(ip4)}, nil
		}})

	m.Dict.SetStr("inet_ntoa", &object.BuiltinFunc{Name: "inet_ntoa",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "inet_ntoa() requires packed bytes")
			}
			b, ok := a[0].(*object.Bytes)
			if !ok || len(b.V) != 4 {
				return nil, object.Errorf(socketErrCls, "inet_ntoa() argument must be 4-byte bytes")
			}
			ip := net.IP(b.V)
			return &object.Str{V: ip.String()}, nil
		}})

	// ── inet_pton / inet_ntop ─────────────────────────────────────────────

	m.Dict.SetStr("inet_pton", &object.BuiltinFunc{Name: "inet_pton",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "inet_pton() requires family and ip")
			}
			family, _ := toInt64(a[0])
			s, ok := a[1].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "inet_pton() ip must be string")
			}
			ip := net.ParseIP(s.V)
			if ip == nil {
				return nil, object.Errorf(socketErrCls, "illegal IP address string")
			}
			if int(family) == int(syscall.AF_INET) {
				ip4 := ip.To4()
				if ip4 == nil {
					return nil, object.Errorf(socketErrCls, "illegal IP address string for AF_INET")
				}
				return &object.Bytes{V: []byte(ip4)}, nil
			}
			// AF_INET6
			ip6 := ip.To16()
			if ip6 == nil {
				return nil, object.Errorf(socketErrCls, "illegal IP address string for AF_INET6")
			}
			return &object.Bytes{V: []byte(ip6)}, nil
		}})

	m.Dict.SetStr("inet_ntop", &object.BuiltinFunc{Name: "inet_ntop",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "inet_ntop() requires family and packed")
			}
			family, _ := toInt64(a[0])
			b, ok := a[1].(*object.Bytes)
			if !ok {
				return nil, object.Errorf(i.typeErr, "inet_ntop() packed must be bytes")
			}
			ip := net.IP(b.V)
			if int(family) == int(syscall.AF_INET) {
				if len(b.V) != 4 {
					return nil, object.Errorf(socketErrCls, "inet_ntop() AF_INET requires 4 bytes")
				}
				return &object.Str{V: ip.String()}, nil
			}
			// AF_INET6
			if len(b.V) != 16 {
				return nil, object.Errorf(socketErrCls, "inet_ntop() AF_INET6 requires 16 bytes")
			}
			return &object.Str{V: ip.String()}, nil
		}})

	// ── byte order ────────────────────────────────────────────────────────

	// Detect host byte order: NativeEndian.Uint16 on {1,0} returns 1 on LE, 256 on BE.
	hostIsLE := binary.NativeEndian.Uint16([]byte{1, 0}) == 1

	swap16 := func(v uint16) uint16 {
		return (v >> 8) | (v << 8)
	}
	swap32 := func(v uint32) uint32 {
		return (v>>24) | ((v>>8)&0xff00) | ((v<<8)&0xff0000) | (v<<24)
	}

	m.Dict.SetStr("htons", &object.BuiltinFunc{Name: "htons",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "htons() requires integer")
			}
			n, _ := toInt64(a[0])
			v := uint16(n)
			if hostIsLE {
				v = swap16(v)
			}
			return object.NewInt(int64(v)), nil
		}})

	m.Dict.SetStr("ntohs", &object.BuiltinFunc{Name: "ntohs",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "ntohs() requires integer")
			}
			n, _ := toInt64(a[0])
			v := uint16(n)
			if hostIsLE {
				v = swap16(v)
			}
			return object.NewInt(int64(v)), nil
		}})

	m.Dict.SetStr("htonl", &object.BuiltinFunc{Name: "htonl",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "htonl() requires integer")
			}
			n, _ := toInt64(a[0])
			v := uint32(n)
			if hostIsLE {
				v = swap32(v)
			}
			return object.NewInt(int64(v)), nil
		}})

	m.Dict.SetStr("ntohl", &object.BuiltinFunc{Name: "ntohl",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "ntohl() requires integer")
			}
			n, _ := toInt64(a[0])
			v := uint32(n)
			if hostIsLE {
				v = swap32(v)
			}
			return object.NewInt(int64(v)), nil
		}})

	// ── interface helpers ─────────────────────────────────────────────────

	m.Dict.SetStr("if_nameindex", &object.BuiltinFunc{Name: "if_nameindex",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ifaces, err := net.Interfaces()
			if err != nil {
				return &object.List{V: nil}, nil
			}
			var items []object.Object
			for _, iface := range ifaces {
				items = append(items, &object.Tuple{V: []object.Object{
					object.NewInt(int64(iface.Index)),
					&object.Str{V: iface.Name},
				}})
			}
			return &object.List{V: items}, nil
		}})

	m.Dict.SetStr("if_nametoindex", &object.BuiltinFunc{Name: "if_nametoindex",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "if_nametoindex() requires name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			iface, err := net.InterfaceByName(name)
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			return object.NewInt(int64(iface.Index)), nil
		}})

	m.Dict.SetStr("if_indextoname", &object.BuiltinFunc{Name: "if_indextoname",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "if_indextoname() requires index")
			}
			idx, _ := toInt64(a[0])
			iface, err := net.InterfaceByIndex(int(idx))
			if err != nil {
				return nil, object.Errorf(socketErrCls, "%v", err)
			}
			return &object.Str{V: iface.Name}, nil
		}})

	// ── CMSG helpers (stubs) ──────────────────────────────────────────────

	m.Dict.SetStr("CMSG_LEN", &object.BuiltinFunc{Name: "CMSG_LEN",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			if len(a) > 0 {
				n, _ = toInt64(a[0])
			}
			return object.NewInt(n), nil
		}})

	m.Dict.SetStr("CMSG_SPACE", &object.BuiltinFunc{Name: "CMSG_SPACE",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			if len(a) > 0 {
				n, _ = toInt64(a[0])
			}
			return object.NewInt(n), nil
		}})

	// Expose the socket class itself (so isinstance works)
	m.Dict.SetStr("SocketType", sockCls)

	return m
}
