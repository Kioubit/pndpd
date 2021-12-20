package main

import (
	"syscall"
)

var fd int

func respond(iface string, requests chan *NDRequest) {
	fd, _ = syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	syscall.BindToDevice(fd, iface)

	for {
		n := <-requests
		pkt(n.srcIP, n.answeringForIP, n.mac)
	}
}

func pkt(srcip []byte, tgtip []byte, mac []byte) {
	v6 := newIpv6Header(emptyIpv6, srcip)
	NDPa := newNdpPacket(tgtip, mac)
	v6.addPayload(NDPa)
	response := v6.constructPacket()

	var t [16]byte
	copy(t[:], srcip)
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
