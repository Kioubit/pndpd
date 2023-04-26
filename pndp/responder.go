package pndp

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"syscall"
)

func respond(iface string, requests chan *ndpRequest, respondType ndpType, ndpQuestionChan chan *ndpQuestion, filter []*net.IPNet, autoSense string, stopWG *sync.WaitGroup, stopChan chan struct{}) {
	stopWG.Add(1)
	defer stopWG.Done()

	var autoiface *net.Interface
	if autoSense != "" {
		var err error
		autoiface, err = net.InterfaceByName(autoSense)
		if err != nil {
			showFatalError(err.Error())
		}
	}

	var ndpQuestionsList = make([]*ndpQuestion, 0, 40)
	var _, linkLocalSpace, _ = net.ParseCIDR("fe80::/10")

	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		showFatalError(err.Error())
	}
	defer func(fd int) {
		_ = syscall.Close(fd)
	}(fd)
	err = syscall.BindToDevice(fd, iface)
	if err != nil {
		showFatalError(err.Error())
	}

	respondIface, err := net.InterfaceByName(iface)
	if err != nil {
		showFatalError(err.Error())
	}

	var selectedSelfSourceIP = emptyIpv6

	for {
		var req *ndpRequest
		if (ndpQuestionChan == nil && respondType == ndp_ADV) || (ndpQuestionChan != nil && respondType == ndp_SOL) {
			select {
			case <-stopChan:
				return
			case req = <-requests:
			}
		} else {
			// This is if ndpQuestionChan != nil && respondType == ndp_ADV
			select {
			case <-stopChan:
				return
			case q := <-ndpQuestionChan:
				ndpQuestionsList = append(ndpQuestionsList, q)
				ndpQuestionsList = cleanupQuestionList(ndpQuestionsList)
				continue
			case req = <-requests:
			}
		}

		v6Header, err := newIpv6Header(req.srcIP, req.dstIP)
		if err != nil {
			continue
		}
		if !checkPacketChecksum(v6Header, req.payload) {
			continue
		}

		if linkLocalSpace.Contains(req.answeringForIP) {
			if GlobalDebug {
				fmt.Println("Dropping packet asking for a link-local IP")
			}
			continue
		}

		selectedSelfSourceIP = getInterfaceInfo(respondIface).sourceIP
		// Auto-sense
		if autoSense != "" {
			filter = getInterfaceInfo(autoiface).networks
		}

		if filter != nil {
			ok := false
			for _, i := range filter {
				if i.Contains(req.answeringForIP) {
					if GlobalDebug {
						fmt.Println("Responded for whitelisted IP", req.answeringForIP)
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

		if req.sourceIface == iface {
			pkt(fd, selectedSelfSourceIP, req.srcIP, req.answeringForIP, respondIface.HardwareAddr, respondType)
		} else {
			if respondType == ndp_ADV {
				if !bytes.Equal(req.dstIP, allNodesMulticastIPv6) { // Skip in case of unsolicited advertisement
					success := false
					req.dstIP, success = getAddressFromQuestionListRetry(req.answeringForIP, ndpQuestionChan, ndpQuestionsList)
					if !success {
						if GlobalDebug {
							fmt.Println("Nobody has asked for this IP")
						}
						continue
					}
				}
			} else {
				if bytes.Equal(req.srcIP, emptyIpv6) {
					// Duplicate Address detection is in progress
					selectedSelfSourceIP = emptyIpv6
				} else {
					ndpQuestionChan <- &ndpQuestion{
						targetIP: req.answeringForIP,
						askedBy:  req.srcIP,
					}
				}
			}
			pkt(fd, selectedSelfSourceIP, req.dstIP, req.answeringForIP, respondIface.HardwareAddr, respondType)
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

	gotBuffered := false
	select {
	case q := <-ndpQuestionChan:
		ndpQuestionsList = append(ndpQuestionsList, q)
		gotBuffered = true
	default:
	}

	if gotBuffered {
		result, success = getAddressFromQuestionList(targetIP, ndpQuestionsList)
	}

	return nil, false
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
	/*
		More efficient but order has some importance as otherwise newer entries might get removed via cleanupQuestionList()
		s[i] = s[len(s)-1]
		return s[:len(s)-1]
	*/
	return append(s[:i], s[i+1:]...)
}

func cleanupQuestionList(s []*ndpQuestion) []*ndpQuestion {
	toRemove := len(s) - 40
	if toRemove <= 0 {
		return s
	}
	return s[toRemove:]
}
