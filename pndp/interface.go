package pndp

import (
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
	"syscall"
	"unsafe"
)

// bpfFilter represents a classic BPF filter program that can be applied to a socket
type bpfFilter []bpf.Instruction

// ApplyTo applies the current filter onto the provided file descriptor
func (filter bpfFilter) ApplyTo(fd int) (err error) {
	var assembled []bpf.RawInstruction
	if assembled, err = bpf.Assemble(filter); err != nil {
		return err
	}

	var program = unix.SockFprog{
		Len:    uint16(len(assembled)),
		Filter: (*unix.SockFilter)(unsafe.Pointer(&assembled[0])),
	}
	var b = (*[unix.SizeofSockFprog]byte)(unsafe.Pointer(&program))[:unix.SizeofSockFprog]

	if _, _, errno := syscall.Syscall6(syscall.SYS_SETSOCKOPT,
		uintptr(fd), uintptr(syscall.SOL_SOCKET), uintptr(syscall.SO_ATTACH_FILTER),
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0); errno != 0 {
		return errno
	}

	return nil
}

type iflags struct {
	name  [syscall.IFNAMSIZ]byte
	flags uint16
}

func setAllMulti(fd int, iface string, enable bool) {
	var ifl iflags
	copy(ifl.name[:], []byte(iface))
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		panic(ep)
	}

	if enable {
		ifl.flags |= uint16(syscall.IFF_ALLMULTI)
	} else {
		ifl.flags &^= uint16(syscall.IFF_ALLMULTI)
	}

	_, _, ep = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		panic(ep)
	}
}
