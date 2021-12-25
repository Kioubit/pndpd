package pndp

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"syscall"
)

func respond(iface string, requests chan *ndpRequest, respondType ndpType, ndpQuestionChan chan *ndpQuestion, filter []*net.IPNet, autoSense string, stopWG *sync.WaitGroup, stopChan chan struct{}) {
	var ndpQuestionsList = make([]*ndpQuestion, 0, 40)
	stopWG.Add(1)
	defer stopWG.Done()
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		panic(err)
	}
	defer func(fd int) {
		_ = syscall.Close(fd)
	}(fd)
	err = syscall.BindToDevice(fd, iface)
	if err != nil {
		panic(err)
	}

	nIface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}

	var result = emptyIpv6
	ifaceaddrs, err := nIface.Addrs()

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
				_, tnet, _ := net.ParseCIDR("fc00::/7")
				if !tnet.Contains(tip) {
					break
				}
			} else if tip.IsLinkLocalUnicast() && !haveUla {
				result = tip
			}
		}
	}

	for {
		var n *ndpRequest
		if (ndpQuestionChan == nil && respondType == ndp_ADV) || (ndpQuestionChan != nil && respondType == ndp_SOL) {
			select {
			case <-stopChan:
				return
			case n = <-requests:
			}
		} else {
			// THis is if ndpQuestionChan != nil && respondType == ndp_ADV
			select {
			case <-stopChan:
				return
			case q := <-ndpQuestionChan:
				ndpQuestionsList = append(ndpQuestionsList, q)
				ndpQuestionsList = cleanupQuestionList(ndpQuestionsList)
				continue
			case n = <-requests:
			}
		}

		var _, LinkLocalSpace, _ = net.ParseCIDR("fe80::/10")
		if LinkLocalSpace.Contains(n.answeringForIP) {
			if GlobalDebug {
				fmt.Println("Dropping packet asking for a link-local IP")
			}
			continue
		}

		if n.requestType == ndp_ADV {
			if (n.rawPacket[78] != 0x02) || (n.rawPacket[79] != 0x01) {
				if GlobalDebug {
					fmt.Println("Dropping Advertisement packet without target Source address set")
				}
				continue
			}
			if n.rawPacket[58] == 0x0 {
				if GlobalDebug {
					fmt.Println("Dropping Advertisement packet without any NDP flags set")
				}
				continue
			}
		} else {
			if (n.rawPacket[78] != 0x01) || (n.rawPacket[79] != 0x01) {
				if GlobalDebug {
					fmt.Println("Dropping Solicitation packet without Source address set")
				}
				continue
			}
		}

		if !checkPacketChecksum(n.srcIP, n.dstIP, n.rawPacket[54:]) {
			if GlobalDebug {
				fmt.Println("Dropping packet because of invalid checksum")
			}
			continue
		}

		if autoSense != "" {
			autoiface, err := net.InterfaceByName(autoSense)
			if err != nil {
				panic(err)
			}
			autoifaceaddrs, err := autoiface.Addrs()

			for _, l := range autoifaceaddrs {
				_, anet, err := net.ParseCIDR(l.String())
				if err != nil {
					break
				}
				if isIpv6(anet.String()) {
					filter = append(filter, anet)
				}
			}
		}

		if filter != nil {
			ok := false
			for _, i := range filter {
				if i.Contains(n.answeringForIP) {
					if GlobalDebug {
						fmt.Println("Responded for whitelisted IP", n.answeringForIP)
					}
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		if GlobalDebug {
			fmt.Println("Getting ready to send packet of type", respondType, "out on interface", iface)
		}

		if n.sourceIface == iface {
			pkt(fd, result, n.srcIP, n.answeringForIP, nIface.HardwareAddr, respondType)
		} else {
			if respondType == ndp_ADV {
				success := false
				n.dstIP, success = getAddressFromQuestionListRetry(n.answeringForIP, ndpQuestionChan, ndpQuestionsList)
				if !success {
					if GlobalDebug {
						fmt.Println("Nobody has asked for this IP")
					}
					continue
				}
			} else {
				ndpQuestionChan <- &ndpQuestion{
					targetIP: n.answeringForIP,
					askedBy:  n.srcIP,
				}
			}
			pkt(fd, result, n.dstIP, n.answeringForIP, nIface.HardwareAddr, respondType)
		}
	}
}

func pkt(fd int, ownIP []byte, dstIP []byte, tgtip []byte, mac []byte, respondType ndpType) {
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
	if GlobalDebug {
		fmt.Println("Sending packet of type", respondType, "to")
		fmt.Printf("% X\n", t)
	}
	err = syscall.Sendto(fd, response, 0, &d)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func getAddressFromQuestionListRetry(targetIP []byte, ndpQuestionChan chan *ndpQuestion, ndpQuestionsList []*ndpQuestion) ([]byte, bool) {
	success := false
	var result []byte
	result, success = getAddressFromQuestionList(targetIP, ndpQuestionsList)
	if success {
		return result, true
	}
forloop:
	for {
		select {
		case q := <-ndpQuestionChan:
			ndpQuestionsList = append(ndpQuestionsList, q)
		default:
			break forloop
		}
	}

	result, success = getAddressFromQuestionList(targetIP, ndpQuestionsList)
	return result, success
}

func getAddressFromQuestionList(targetIP []byte, ndpQuestionsList []*ndpQuestion) ([]byte, bool) {
	for i := range ndpQuestionsList {
		if bytes.Equal((*ndpQuestionsList[i]).targetIP, targetIP) {
			result := (*ndpQuestionsList[i]).askedBy
			ndpQuestionsList = removeFromQuestionList(ndpQuestionsList, i)
			return result, true
		}
	}
	return nil, false
}
func removeFromQuestionList(s []*ndpQuestion, i int) []*ndpQuestion {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func cleanupQuestionList(s []*ndpQuestion) []*ndpQuestion {
	for len(s) >= 40 {
		s = removeFromQuestionList(s, 0)
	}
	return s
}
