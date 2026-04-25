//go:build windows

package vm

import (
	"net"

	"github.com/tamnd/goipy/object"
)

func registerSocketPlatformConstants(m *object.Module, setInt func(string, int)) {
	setInt("IPPROTO_ICMP", 1)
	setInt("IPPROTO_RAW", 255)
	setInt("SO_ERROR", 0x1007)
	setInt("SO_TYPE", 0x1008)
	setInt("MSG_OOB", 0x1)
	setInt("MSG_DONTWAIT", 0x40)
	setInt("MSG_WAITALL", 0x8)
	setInt("MSG_PEEK", 0x2)
	setInt("IP_HDRINCL", 2)
	setInt("IP_MULTICAST_TTL", 10)
	setInt("IP_MULTICAST_LOOP", 11)
	setInt("IP_ADD_MEMBERSHIP", 12)
}

// socketpairImpl creates a connected socket pair via a loopback TCP connection.
func socketpairImpl(i *Interp, sockCls *object.Class, socketErrCls *object.Class, family, socktype int) (object.Object, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, object.Errorf(socketErrCls, "socketpair: %v", err)
	}
	defer ln.Close()

	c1, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		return nil, object.Errorf(socketErrCls, "socketpair: %v", err)
	}

	c2, err := ln.Accept()
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
}
