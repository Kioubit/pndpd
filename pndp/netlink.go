package pndp

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// TODO autosense, source ip implementation

type netlinkSocket struct {
	fd  int
	lsa unix.SockaddrNetlink
}

type InterfaceAddressUpdate struct {
	InterfaceIndex int
	Event          AddressUpdateInfo
	NetworkFamily  NetworkFamily
	Flags          byte
	Scope          byte
}

type NetworkFamily int
type AddressUpdateInfo int

const (
	IPv4          NetworkFamily     = 4
	IPv6          NetworkFamily     = 6
	AddressDelete AddressUpdateInfo = 0
	AddressAdd    AddressUpdateInfo = 1
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
		socket.lsa.Groups |= (1 << (g - 1))
	}

	err = unix.Bind(fd, &socket.lsa)
	if err != nil {
		unix.Close(fd)
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
	unix.Close(socket.fd)
	socket.fd = -1
}

func GetInterfaceUpdates(updateChannel chan *InterfaceAddressUpdate, stopChannel chan interface{}) error {
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
			var event AddressUpdateInfo
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

				var networkFamily NetworkFamily
				switch int(ifAddrMsg.Family) {
				case unix.AF_INET:
					networkFamily = IPv4
				case unix.AF_INET6:
					networkFamily = IPv6
				default:
					continue
				}

				update := &InterfaceAddressUpdate{}
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
