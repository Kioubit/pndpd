package pndp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
	"net"
	"sync"
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

func listen(iface string, responder chan *ndpRequest, requestType ndpType, stopWG *sync.WaitGroup, stopChan chan struct{}) {
	stopWG.Add(1)
	defer stopWG.Done()
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
		<-stopChan
		syscall.Close(fd)
		stopWG.Done() // syscall.read does not release when the file descriptor is closed
	}()
	if GlobalDebug {
		fmt.Println("Obtained fd ", fd)
	}

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
		if numRead < 86 {
			if GlobalDebug {

				fmt.Println("Dropping packet since it does not meet the minimum length requirement")
				fmt.Printf("% X\n", buf[:numRead])
			}
			continue
		}
		if GlobalDebug {
			fmt.Println("Got packet on", iface, "of type", requestType)
			fmt.Printf("% X\n", buf[:numRead])

			fmt.Println("Source MAC ETHER")
			fmt.Printf("% X\n", buf[:numRead][6:12])
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

		if bytes.Equal(buf[:numRead][6:12], niface.HardwareAddr) {
			if GlobalDebug {
				fmt.Println("Dropping packet from ourselves")
			}
			continue
		}

		if !checkPacketChecksum(buf[:numRead][22:38], buf[:numRead][38:54], buf[:numRead][54:numRead]) {
			if GlobalDebug {
				fmt.Println("Dropping packet because of invalid checksum")
			}
			continue
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

func checkPacketChecksum(scrip, dstip, payload []byte) bool {
	v6, err := newIpv6Header(scrip, dstip)
	if err != nil {
		return false
	}

	packetsum := make([]byte, 2)
	copy(packetsum, payload[2:4])

	bPayloadLen := make([]byte, 2)
	binary.BigEndian.PutUint16(bPayloadLen, uint16(len(payload)))
	v6.payloadLen = bPayloadLen

	payload[2] = 0x0
	payload[3] = 0x0

	bChecksum := make([]byte, 2)
	binary.BigEndian.PutUint16(bChecksum, calculateChecksum(v6, payload))
	if bytes.Equal(packetsum, bChecksum) {
		if GlobalDebug {
			fmt.Println("Verified received packet checksum")
		}
		return true
	} else {
		if GlobalDebug {
			fmt.Println("Received packet checksum validation failed")
		}
		return false
	}
}
