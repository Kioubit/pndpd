package main

import (
	"net"
	"syscall"
)

var fd int

func respond(iface string, requests chan *NDRequest, respondType NDPType) {
	fd, _ = syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	syscall.BindToDevice(fd, iface)

	niface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}

	for {
		n := <-requests
		pkt(n.srcIP, n.answeringForIP, niface.HardwareAddr, respondType)
	}
}

func pkt(srcip []byte, tgtip []byte, mac []byte, respondType NDPType) {
	v6 := newIpv6Header(emptyIpv6, srcip)
	NDPa := newNdpPacket(tgtip, mac, respondType)
	v6.addPayload(NDPa)
	response := v6.constructPacket()

	var t [16]byte
	if respondType == NDP_SOL {
		copy(t[:], []byte{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02})
	} else {
		copy(t[:], srcip)
	}

	d := syscall.SockaddrInet6{
		Port: 0,
		Addr: t,
	}
	err := syscall.Sendto(fd, response, 0, &d)
	if err != nil {
		panic(err)
	}

	syscall.Close(fd)
}
