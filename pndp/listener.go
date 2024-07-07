package pndp

import (
	"bytes"
	"errors"
	"golang.org/x/net/bpf"
	"log/slog"
	"net"
	"os"
	"sync"
	"syscall"
)

func listen(iface string, responder chan *ndpRequest, requestType ndpType, stopWG *sync.WaitGroup, stopChan chan struct{}) {
	stopWG.Add(1)
	defer stopWG.Done()

	niface, err := net.InterfaceByName(iface)
	if err != nil {
		showFatalError(err.Error())
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, htons(syscall.ETH_P_IPV6))
	if err != nil {
		showFatalError("Failed setting up listener on interface", iface)
	}

	slog.Debug("Obtained fd", "fd", fd)
	err = syscall.Bind(fd, &syscall.SockaddrLinklayer{
		Protocol: htons16(syscall.ETH_P_IPV6),
		Ifindex:  niface.Index,
	})
	if err != nil {
		showFatalError(err.Error())
	}
	slog.Debug("Bound to interface", "fd", fd, "interface", iface)

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
		showFatalError(err.Error())
	}

	err = syscall.SetNonblock(fd, true)
	if err != nil {
		slog.Warn("Failed setting nonblock", "fd", fd)
	}

	fdN := os.NewFile(uintptr(fd), "")
	go func() {
		<-stopChan
		_ = fdN.Close()
	}()

	for {
		buf := make([]byte, 86)
		numRead, err := fdN.Read(buf)
		if err != nil {
			if errors.Is(err, os.ErrClosed) {
				return
			}
			showFatalError(err.Error())
		}

		pLogger := slog.Default().With("packet", hexValue{buf[:numRead]})

		if numRead < 78 {
			pLogger.Debug("Dropping packet since it does not meet the minimum length requirement")
			continue
		}

		if bytes.Equal(buf[6:12], niface.HardwareAddr) {
			pLogger.Debug("Dropping packet from ourselves")
			continue
		}

		if requestType == ndp_ADV {
			if buf[58] == 0x0 {
				pLogger.Debug("Dropping advertisement packet without any NDP flags set")
				continue
			}
		}

		pLogger.Debug("Got packet", "interface", iface, "type", requestType,
			"source MAC", macValue{buf[6:12]},
			"source IP", ipValue{buf[22:38]},
			"destination IP", ipValue{buf[38:54]},
			"requested IP", ipValue{buf[62:78]},
		)

		responder <- &ndpRequest{
			requestType:    requestType,
			srcIP:          buf[22:38],
			dstIP:          buf[38:54],
			answeringForIP: buf[62:78],
			payload:        buf[54:numRead],
			sourceIface:    iface,
		}
	}
}
