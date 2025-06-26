package pndp

import (
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
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

func setPromisc(fd int, iface string, enable bool) {
	iFace, err := net.InterfaceByName(iface)
	if err != nil {
		showFatalError(err.Error())
		return
	}

	mReq := unix.PacketMreq{
		Ifindex: int32(iFace.Index),
		Type:    unix.PACKET_MR_PROMISC,
	}

	var opt int
	if enable {
		opt = unix.PACKET_ADD_MEMBERSHIP
	} else {
		opt = unix.PACKET_DROP_MEMBERSHIP
	}

	err = unix.SetsockoptPacketMreq(fd, unix.SOL_PACKET, opt, &mReq)
	if err != nil {
		showFatalError(err.Error())
	}
}

func selectSourceIP(iface *net.Interface) (gua []byte, ula []byte) {
	gua = emptyIpv6
	ula = emptyIpv6
	interfaceAddresses, err := iface.Addrs()
	if err != nil {
		return gua, ula
	}

	var haveUla = false
	var haveGua = false
	for _, l := range interfaceAddresses {
		if haveGua && haveUla {
			break
		}

		ipNet, ok := l.(*net.IPNet)
		if !ok {
			continue
		}
		testIP := ipNet.IP

		if isIpv6(ipNet) {
			if testIP.IsGlobalUnicast() {
				if !ulaSpace.Contains(testIP) {
					haveGua = true
					gua = testIP
				} else {
					haveUla = true
					ula = testIP
				}
			} else if testIP.IsLinkLocalUnicast() {
				if !haveUla {
					ula = testIP
				}
				if !haveGua {
					gua = testIP
				}
			}
		}
	}
	return gua, ula
}

func getInterfaceNetworkList(iface *net.Interface) []*net.IPNet {
	filter := make([]*net.IPNet, 0)
	autoifaceaddrs, err := iface.Addrs()
	if err != nil {
		return filter
	}
	for _, l := range autoifaceaddrs {
		ipNet, ok := l.(*net.IPNet)
		if !ok {
			continue
		}

		if isIpv6(ipNet) {
			filter = append(filter, ipNet)
		}
	}
	return filter
}
