package pndp

import (
	"bytes"
	"log/slog"
	"net"
	"sync"
	"syscall"
)

func respond(iface string, requests chan *ndpRequest, respondType ndpType, ndpQuestionChan chan ndpQuestion, filter []*net.IPNet, autoSense string, stopWG *sync.WaitGroup, stopChan chan struct{}) {
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

	var ndpQuestionsList = make([]ndpQuestion, 0, 40)
	var _, linkLocalSpace, _ = net.ParseCIDR("fe80::/10")

	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, syscall.IPPROTO_RAW)
	if err != nil {
		showFatalError(err.Error())
	}
	defer func(fd int) {
		_ = syscall.Close(fd)
	}(fd)
	slog.Debug("Obtained fd", "fd", fd)

	err = syscall.BindToDevice(fd, iface)
	if err != nil {
		showFatalError(err.Error())
	}
	slog.Debug("Bound to interface", "fd", fd, "interface", iface)

	respondIface, err := net.InterfaceByName(iface)
	if err != nil {
		showFatalError(err.Error())
	}

	for {
		var req *ndpRequest
		if (ndpQuestionChan == nil && respondType == ndpAdv) || (ndpQuestionChan != nil && respondType == ndpSol) {
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
			slog.Debug("Dropping packet asking for a link-local IP")
			continue
		}

		// Auto-sense
		if autoSense != "" {
			filter = getInterfaceInfo(autoiface).networks
		}

		if filter != nil {
			ok := false
			for _, i := range filter {
				if i.Contains(req.answeringForIP) {
					slog.Debug("Responding for whitelisted IP", "ip", ipValue{req.answeringForIP})
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		intInfo := getInterfaceInfo(respondIface)
		var selectedSelfSourceIPGua = intInfo.sourceIP
		var selectedSelfSourceIPUla = intInfo.sourceIPULA
		var selectedSelfSourceIP = selectedSelfSourceIPGua
		if ulaSpace.Contains(req.answeringForIP) {
			selectedSelfSourceIP = selectedSelfSourceIPUla
		}

		if req.sourceIface == iface {
			slog.Debug("Sending packet", "type", respondType, "dest", ipValue{req.dstIP}, "interface", respondIface.Name)
			sendNDPPacket(fd, req.dstIP, req.srcIP, req.answeringForIP, respondIface.HardwareAddr, respondType)
		} else {
			if respondType == ndpAdv {
				if !bytes.Equal(req.dstIP, allNodesMulticastIPv6) { // Skip in case of unsolicited advertisement
					success := false
					req.dstIP, success = getAddressFromQuestionList(req.answeringForIP, &ndpQuestionsList)
					if !success {
						slog.Debug("Nobody has asked for this IP", "ip", ipValue{req.answeringForIP})
						continue
					}
				}
			} else {
				if bytes.Equal(req.srcIP, emptyIpv6) {
					// Duplicate Address detection is in progress
					selectedSelfSourceIP = emptyIpv6
				} else {
					ndpQuestionChan <- ndpQuestion{
						targetIP: req.answeringForIP,
						askedBy:  req.srcIP,
					}
				}
			}
			slog.Debug("Sending packet", "type", respondType, "dest", ipValue{req.dstIP}, "interface", respondIface.Name)

			sendNDPPacket(fd, selectedSelfSourceIP, req.dstIP, req.answeringForIP, respondIface.HardwareAddr, respondType)
		}
	}
}

func sendNDPPacket(fd int, ownIP []byte, dstIP []byte, ndpTargetIP []byte, ndpTargetMac []byte, ndpType ndpType) {
	v6, err := newIpv6Header(ownIP, dstIP)
	if err != nil {
		return
	}
	NDPa, err := newNdpPacket(ndpTargetIP, ndpTargetMac, ndpType)
	if err != nil {
		return
	}
	v6.addPayload(NDPa)
	packet := v6.constructPacket()

	var t [16]byte
	copy(t[:], dstIP)

	err = syscall.Sendto(fd, packet, 0, &syscall.SockaddrInet6{
		Addr: t,
	})
	if err != nil {
		slog.Error("Error sending packet", "error", err)
	}
}

func getAddressFromQuestionList(targetIP []byte, ndpQuestionsList *[]ndpQuestion) ([]byte, bool) {
	for i := range *ndpQuestionsList {
		if bytes.Equal((*ndpQuestionsList)[i].targetIP, targetIP) {
			result := (*ndpQuestionsList)[i].askedBy
			*ndpQuestionsList = removeFromQuestionList(*ndpQuestionsList, i)
			return result, true
		}
	}
	return nil, false
}
func removeFromQuestionList(s []ndpQuestion, i int) []ndpQuestion {
	// Remove while keeping the order
	return append(s[:i], s[i+1:]...)
}

func cleanupQuestionList(s []ndpQuestion) []ndpQuestion {
	toRemove := len(s) - 40
	if toRemove <= 0 {
		return s
	}
	return s[toRemove:]
}
