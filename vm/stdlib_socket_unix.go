//go:build unix

package vm

import (
	"net"
	"os"
	"syscall"

	"github.com/tamnd/goipy/object"
)

func registerSocketPlatformConstants(m *object.Module, setInt func(string, int)) {
	setInt("IPPROTO_ICMP", int(syscall.IPPROTO_ICMP))
	setInt("IPPROTO_RAW", int(syscall.IPPROTO_RAW))
	setInt("SO_ERROR", int(syscall.SO_ERROR))
	setInt("SO_TYPE", int(syscall.SO_TYPE))
	setInt("MSG_OOB", int(syscall.MSG_OOB))
	setInt("MSG_DONTWAIT", int(syscall.MSG_DONTWAIT))
	setInt("MSG_WAITALL", int(syscall.MSG_WAITALL))
	setInt("MSG_PEEK", int(syscall.MSG_PEEK))
	setInt("IP_HDRINCL", int(syscall.IP_HDRINCL))
	setInt("IP_MULTICAST_TTL", int(syscall.IP_MULTICAST_TTL))
	setInt("IP_MULTICAST_LOOP", int(syscall.IP_MULTICAST_LOOP))
	setInt("IP_ADD_MEMBERSHIP", int(syscall.IP_ADD_MEMBERSHIP))
}

func socketpairImpl(i *Interp, sockCls *object.Class, socketErrCls *object.Class, family, socktype int) (object.Object, error) {
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
}
