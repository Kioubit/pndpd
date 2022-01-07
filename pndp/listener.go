package pndp

import (
	"bytes"
	"fmt"
	"golang.org/x/net/bpf"
	"net"
	"sync"
	"syscall"
)

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
		setPromisc(fd, iface, false, false)
		_ = syscall.Close(fd)
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

	setPromisc(fd, iface, true, false)

	var protocolNo uint32
	if requestType == ndp_SOL {
		//Neighbor Solicitation
		protocolNo = 0x87
	} else {
		//Neighbor Advertisement
		protocolNo = 0x88
	}
	var f bpfFilter
	f = []bpf.Instruction{
		// Load "EtherType" field from the ethernet header.
		bpf.LoadAbsolute{Off: 12, Size: 2},
		// Jump to the drop packet instruction if EtherType is not IPv6.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x86dd, SkipTrue: 5},
		// Load "Next Header" field from IPV6 header.
		bpf.LoadAbsolute{Off: 20, Size: 1},
		// Jump to the drop packet instruction if Next Header is not ICMPv6.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: 0x3a, SkipTrue: 3},
		// Load "Type" field from ICMPv6 header.
		bpf.LoadAbsolute{Off: 54, Size: 1},
		// Jump to the drop packet instruction if Type is not Neighbor Solicitation / Advertisement.
		bpf.JumpIf{Cond: bpf.JumpNotEqual, Val: protocolNo, SkipTrue: 1},
		// Verdict is: send up to 86 bytes of the packet to userspace.
		bpf.RetConstant{Val: 86},
		// Verdict is: "ignore packet."
		bpf.RetConstant{Val: 0},
	}

	err = f.ApplyTo(fd)
	if err != nil {
		panic(err.Error())
	}

	for {
		buf := make([]byte, 86)
		numRead, err := syscall.Read(fd, buf)
		if err != nil {
			panic(err)
		}
		if numRead < 78 {
			if GlobalDebug {
				fmt.Println("Dropping packet since it does not meet the minimum length requirement")
				fmt.Printf("% X\n", buf[:numRead])
			}
			continue
		}
		if GlobalDebug {
			fmt.Println("Got packet on", iface, "of type", requestType)
			fmt.Printf("% X\n", buf[:numRead])

			fmt.Println("Source mac on ethernet layer:")
			fmt.Printf("% X\n", buf[6:12])
			fmt.Println("Source IP:")
			fmt.Printf("% X\n", buf[22:38])
			fmt.Println("Destination IP:")
			fmt.Printf("% X\n", buf[38:54])
			fmt.Println("Requested IP:")
			fmt.Printf("% X\n", buf[62:78])
			if requestType == ndp_ADV {
				fmt.Println("NDP Flags")
				fmt.Printf("% X\n", buf[58])
			}
			fmt.Println()
		}

		if bytes.Equal(buf[6:12], niface.HardwareAddr) {
			if GlobalDebug {
				fmt.Println("Dropping packet from ourselves")
			}
			continue
		}

		if requestType == ndp_ADV {
			if buf[58] == 0x0 {
				if GlobalDebug {
					fmt.Println("Dropping Advertisement packet without any NDP flags set")
				}
				continue
			}
		}

		responder <- &ndpRequest{
			requestType:    requestType,
			srcIP:          buf[22:38],
			dstIP:          buf[38:54],
			answeringForIP: buf[62:78],
			payload:        buf[54:],
			sourceIface:    iface,
		}
	}
}
