package pndp

import (
	"fmt"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
	"net"
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

// Htons Convert a uint16 to host byte order (big endian)
func htons(v uint16) int {
	return int((v << 8) | (v >> 8))
}
func htons16(v uint16) uint16 { return v<<8 | v>>8 }

func listen(iface string, responder chan *ndpRequest, requestType ndpType) {
	niface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}
	tiface := &syscall.SockaddrLinklayer{
		Protocol: htons16(syscall.ETH_P_IPV6),
		Ifindex:  niface.Index,
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, htons(syscall.ETH_P_IPV6))
	if err != nil {
		fmt.Println(err.Error())
	}
	go func() {
		<-stop
		syscall.Close(fd)
		stopWg.Done() // syscall.read does not release when the file descriptor is closed
	}()
	fmt.Println("Obtained fd ", fd)

	if len([]byte(iface)) > syscall.IFNAMSIZ {
		panic("Interface size larger then maximum allowed by the kernel")
	}

	err = syscall.Bind(fd, tiface)
	if err != nil {
		panic(err.Error())
	}

	var protocolNo uint32
	if requestType == ndp_SOL {
		//Neighbor Solicitation
		protocolNo = 0x87
	} else {
		//Neighbor Advertisement
		protocolNo = 0x88
	}

	var f bpfFilter = []bpf.Instruction{
		// Load "EtherType" field from the ethernet header.
		bpf.LoadAbsolute{Off: 12, Size: 2},
		// Jump to the drop packet instruction if EtherType is not IPv6.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x86dd, SkipTrue: 4},
		// Load "Next Header" field from IPV6 header.
		bpf.LoadAbsolute{Off: 20, Size: 1},
		// Jump to the drop packet instruction if Next Header is not ICMPv6.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x3a, SkipTrue: 2},
		// Load "Type" field from ICMPv6 header.
		bpf.LoadAbsolute{Off: 54, Size: 1},
		// Jump to the drop packet instruction if Type is not Neighbor Solicitation / Advertisement.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: protocolNo, SkipTrue: 1},
		// Verdict is "send up to 4k of the packet to userspace."buf
		bpf.RetConstant{Val: 4096},
		// Verdict is "ignore packet."
		bpf.RetConstant{Val: 0},
	}

	err = f.ApplyTo(fd)
	if err != nil {
		panic(err.Error())
	}

	for {
		buf := make([]byte, 4096)
		numRead, err := syscall.Read(fd, buf)
		if err != nil {
			panic(err)
		}
		if GlobalDebug {
			fmt.Println("Source IP:")
			fmt.Printf("% X\n", buf[:numRead][22:38])
			fmt.Println("Destination IP:")
			fmt.Printf("% X\n", buf[:numRead][38:54])
			fmt.Println("Requested IP:")
			fmt.Printf("% X\n", buf[:numRead][62:78])
			fmt.Println("Source MAC")
			fmt.Printf("% X\n", buf[:numRead][80:86])
			fmt.Println()
		}
		responder <- &ndpRequest{
			requestType:      requestType,
			srcIP:            buf[:numRead][22:38],
			dstIP:            buf[:numRead][38:54],
			answeringForIP:   buf[:numRead][62:78],
			mac:              buf[:numRead][80:86],
			receivedIfaceMac: niface.HardwareAddr,
			sourceIface:      iface,
		}
	}
}
