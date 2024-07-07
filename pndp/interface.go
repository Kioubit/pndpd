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

type iflags struct {
	name  [syscall.IFNAMSIZ]byte
	flags uint16
}

func setPromisc(fd int, iface string, enable bool, withInterfaceFlags bool) {

	// -------------------------- Interface flags --------------------------
	if withInterfaceFlags {
		tFD, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_DGRAM, 0)
		if err != nil {
			showFatalError(err.Error())
		}

		var ifl iflags
		copy(ifl.name[:], iface)
		_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(tFD), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
		if ep != 0 {
			showFatalError(ep.Error())
		}

		if enable {
			ifl.flags |= uint16(syscall.IFF_PROMISC)
		} else {
			ifl.flags &^= uint16(syscall.IFF_PROMISC)
		}

		_, _, ep = syscall.Syscall(syscall.SYS_IOCTL, uintptr(tFD), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
		if ep != 0 {
			showFatalError(ep.Error())
		}

		_ = syscall.Close(tFD)
	}
	// ---------------------------------------------------------------------

	// -------------------------- Socket Options ---------------------------
	iFace, err := net.InterfaceByName(iface)
	if err != nil {
		showFatalError(err.Error())
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
	// ---------------------------------------------------------------------
}

func selectSourceIP(iface *net.Interface) []byte {
	var _, ulaSpace, _ = net.ParseCIDR("fc00::/7")
	var result = emptyIpv6
	ifaceaddrs, err := iface.Addrs()
	if err != nil {
		return result
	}

	for _, n := range ifaceaddrs {
		tip, _, err := net.ParseCIDR(n.String())
		if err != nil {
			break
		}
		var haveUla = false
		if isIpv6(tip.String()) {
			if tip.IsGlobalUnicast() {
				haveUla = true
				result = tip

				if !ulaSpace.Contains(tip) {
					break
				}
			} else if tip.IsLinkLocalUnicast() && !haveUla {
				result = tip
			}
		}
	}
	return result
}

func getInterfaceNetworkList(iface *net.Interface) []*net.IPNet {
	filter := make([]*net.IPNet, 0)
	autoifaceaddrs, err := iface.Addrs()
	if err != nil {
		return filter
	}
	for _, l := range autoifaceaddrs {
		testIP, anet, err := net.ParseCIDR(l.String())
		if err != nil {
			break
		}
		if isIpv6(testIP.String()) {
			filter = append(filter, anet)
		}
	}
	return filter
}
