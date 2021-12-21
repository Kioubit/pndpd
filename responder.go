package main

import (
	"bytes"
	"fmt"
	"net"
	"syscall"
)

var globalFd int

func respond(iface string, requests chan *NDRequest, respondType NDPType) {
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(globalFd)
	globalFd = fd
	err = syscall.BindToDevice(fd, iface)
	if err != nil {
		panic(err)
	}

	niface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}

	var result = emptyIpv6
	ifaceaddrs, err := niface.Addrs()

	for _, n := range ifaceaddrs {
		tip, _, err := net.ParseCIDR(n.String())
		if err != nil {
			break
		}
		if IsIPv6(tip.String()) {
			if tip.IsGlobalUnicast() {
				result = tip
				_, tnet, _ := net.ParseCIDR("fc00::/7")
				if !tnet.Contains(tip) {
					break
				}
			}
		}
	}

	for {
		n := <-requests
		if n.sourceIface == iface {
			pkt(result, n.srcIP, n.answeringForIP, niface.HardwareAddr, respondType)
		} else {
			if !bytes.Equal(n.mac, n.receivedIfaceMac) {
				pkt(n.srcIP, n.dstIP, n.answeringForIP, niface.HardwareAddr, respondType)
			}
		}
	}
}

func pkt(ownIP []byte, dstIP []byte, tgtip []byte, mac []byte, respondType NDPType) {
	v6, err := newIpv6Header(ownIP, dstIP)
	if err != nil {
		return
	}
	NDPa, err := newNdpPacket(tgtip, mac, respondType)
	if err != nil {
		return
	}
	v6.addPayload(NDPa)
	response := v6.constructPacket()

	var t [16]byte
	copy(t[:], dstIP)

	d := syscall.SockaddrInet6{
		Port: 0,
		Addr: t,
	}
	fmt.Println("Sending packet of type", respondType, "to")
	fmt.Printf("% X\n", t)

	err = syscall.Sendto(globalFd, response, 0, &d)
	fmt.Println(err.Error())
}
