package pndp

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type netlinkSocket struct {
	fd  int
	lsa unix.SockaddrNetlink
}

type interfaceAddressUpdate struct {
	InterfaceIndex int
	Event          addressUpdateInfo
	NetworkFamily  networkFamily
	Flags          byte
	Scope          byte
}

type networkFamily int
type addressUpdateInfo int

const (
	IPv4          networkFamily     = 4
	IPv6          networkFamily     = 6
	AddressDelete addressUpdateInfo = 0
	AddressAdd    addressUpdateInfo = 1
)

func newNetlinkSocket(protocol int, multicastGroups ...uint) (*netlinkSocket, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, protocol)
	if err != nil {
		return nil, err
	}

	socket := &netlinkSocket{}
	socket.fd = fd
	socket.lsa.Family = unix.AF_NETLINK

	for _, g := range multicastGroups {
		socket.lsa.Groups |= 1 << (g - 1)
	}

	err = unix.Bind(fd, &socket.lsa)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return socket, nil
}

func (socket *netlinkSocket) receiveMessage() ([]syscall.NetlinkMessage, *unix.SockaddrNetlink, error) {
	fd := socket.fd
	if fd < 0 {
		return nil, nil, fmt.Errorf("socket is closed")
	}

	var buf [7000]byte
	n, from, err := unix.Recvfrom(fd, buf[:], 0)
	if err != nil {
		return nil, nil, err
	}
	if n < unix.NLMSG_HDRLEN {
		return nil, nil, fmt.Errorf("received a message not meeting the minimum message threshold")
	}
	read := make([]byte, n)
	copy(read, buf[:n])

	fromAddr, ok := from.(*unix.SockaddrNetlink)
	if !ok {
		return nil, nil, fmt.Errorf("unable to convert to SockaddrNetlink")
	}

	nl, err := syscall.ParseNetlinkMessage(read)
	if err != nil {
		return nil, nil, err
	}
	return nl, fromAddr, nil
}

func (socket *netlinkSocket) Close() {
	_ = unix.Close(socket.fd)
	socket.fd = -1
}

func getInterfaceUpdates(updateChannel chan *interfaceAddressUpdate, stopChannel chan interface{}) error {
	// Note: UpdateChannel should be buffered

	socket, err := newNetlinkSocket(unix.NETLINK_ROUTE, unix.RTNLGRP_IPV4_IFADDR, unix.RTNLGRP_IPV6_IFADDR)
	if err != nil {
		return err
	}

	if stopChannel != nil {
		go func() {
			<-stopChannel
			socket.Close()
			close(updateChannel)
		}()
	}
	go func() {
		for {
			messages, from, err := socket.receiveMessage()
			if err != nil {
				//Error receiving
				return
			}
			const kernelPid = 0
			if from.Pid != kernelPid {
				continue
			}
			var event addressUpdateInfo
			for i := range messages {
				switch messages[i].Header.Type {
				case unix.NLMSG_DONE:
					continue
				case unix.NLMSG_ERROR:
					continue
				case unix.RTM_NEWADDR:
					event = AddressAdd
				case unix.RTM_DELADDR:
					event = AddressDelete
				default:
					continue
				}

				t := messages[i].Data[:unix.SizeofIfAddrmsg]
				ifAddrMsgPointer := unsafe.Pointer(&t[0])
				ifAddrMsg := (*unix.IfAddrmsg)(ifAddrMsgPointer)

				var networkFamily networkFamily
				switch int(ifAddrMsg.Family) {
				case unix.AF_INET:
					networkFamily = IPv4
				case unix.AF_INET6:
					networkFamily = IPv6
				default:
					continue
				}

				update := &interfaceAddressUpdate{}
				update.Event = event
				update.InterfaceIndex = int(ifAddrMsg.Index)
				update.Flags = ifAddrMsg.Flags
				update.Scope = ifAddrMsg.Scope
				update.NetworkFamily = networkFamily
				updateChannel <- update
			}
		}
	}()
	return nil
}
