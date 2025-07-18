package pndp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log/slog"
	"net"
)

var emptyIpv6 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var allNodesMulticastIPv6 = []byte{0xFF, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}

type payload interface {
	constructPacket() ([]byte, int)
}

type ipv6Header struct {
	protocol   byte
	srcIP      []byte
	dstIP      []byte
	payloadLen []byte
	payload    []byte
}

func newIpv6Header(srcIp []byte, dstIp []byte) (*ipv6Header, error) {
	if len(dstIp) != 16 || len(srcIp) != 16 {
		return nil, errors.New("malformed IP")
	}
	return &ipv6Header{dstIP: dstIp, srcIP: srcIp, protocol: 0x3a}, nil
}

func (h *ipv6Header) addPayload(payload payload) {
	bPayload, checksumPos := payload.constructPacket()
	bPayloadLen := make([]byte, 2)
	binary.BigEndian.PutUint16(bPayloadLen, uint16(len(bPayload)))
	h.payloadLen = bPayloadLen

	if checksumPos > 0 {
		bChecksum := make([]byte, 2)
		binary.BigEndian.PutUint16(bChecksum, calculateChecksum(h, bPayload))
		bPayload[checksumPos] = bChecksum[0]
		bPayload[checksumPos+1] = bChecksum[1]
	}

	h.payload = bPayload
}

func (h *ipv6Header) constructPacket() []byte {
	header := []byte{
		0x60,            // v6
		0,               // qos
		0,               // qos
		0,               // qos
		h.payloadLen[0], // payload Length
		h.payloadLen[1], // payload Length
		h.protocol,      // Protocol next header
		0xff,            // Hop limit
	}
	final := append(header, h.srcIP...)
	final = append(final, h.dstIP...)
	final = append(final, h.payload...)
	return final
}

type ndpPayload struct {
	packetType     ndpType
	answeringForIP []byte
	mac            []byte
}

func newNdpPacket(answeringForIP []byte, mac []byte, packetType ndpType) (*ndpPayload, error) {
	if len(answeringForIP) != 16 || len(mac) != 6 {
		return nil, errors.New("malformed IP")
	}
	return &ndpPayload{
		packetType:     packetType,
		answeringForIP: answeringForIP,
		mac:            mac,
	}, nil
}

func (p *ndpPayload) constructPacket() ([]byte, int) {
	var protocol byte
	var flags byte
	var linkType byte
	if p.packetType == ndpSol {
		protocol = 0x87
		flags = 0x0
		linkType = 0x01
	} else {
		protocol = 0x88
		flags = 0xe0 // (Router, Solicited, Override)
		linkType = 0x02
	}
	header := []byte{
		protocol, // Type: ndpType
		0x0,      // Code
		0x0,      // Checksum filled in later
		0x0,      // Checksum filled in later
		flags,    // Flags
		0x0,      // Reserved
		0x0,      // Reserved
		0x0,      // Reserved
	}
	final := append(header, p.answeringForIP...)

	secondHeader := []byte{
		linkType, // Type
		0x01,     // Length: 1 (8 bytes)
	}
	final = append(final, secondHeader...)

	final = append(final, p.mac...)
	return final, 2
}

func calculateChecksum(h *ipv6Header, payload []byte) uint16 {
	if len(payload) == 0 {
		return 0x0000
	}

	buf := checksumAddition(h.srcIP, 0)
	buf = checksumAddition(h.dstIP, buf)
	buf = checksumAddition([]byte{0x00, h.protocol}, buf)
	buf = checksumAddition(h.payloadLen, buf)
	buf = checksumAddition(payload, buf)
	return uint16(buf) ^ 0xFFFF
}

func checksumAddition(b []byte, buf uint16) uint16 {
	var sum = uint32(buf)
	cv := len(b) - 1
	for i := 0; i < cv; i += 2 {
		sum += uint32(uint16(b[i])<<8 | uint16(b[i+1]))
	}
	if cv&1 == 0 {
		sum += uint32(uint16(b[cv])<<8 | uint16(0x00))
	}

	for sum>>16 > 0x0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return uint16(sum)
}

func checkPacketChecksum(v6 *ipv6Header, payload []byte) bool {
	packetsum := make([]byte, 2)
	copy(packetsum, payload[2:4])

	bPayloadLen := make([]byte, 2)
	binary.BigEndian.PutUint16(bPayloadLen, uint16(len(payload)))
	v6.payloadLen = bPayloadLen

	payload[2] = 0x0
	payload[3] = 0x0

	bChecksum := make([]byte, 2)
	binary.BigEndian.PutUint16(bChecksum, calculateChecksum(v6, payload))
	if bytes.Equal(packetsum, bChecksum) {
		return true
	} else {
		slog.Debug("Received packet checksum validation failed", "payload", hexValue{payload},
			"v6SrcIP", ipValue{v6.srcIP},
			"v6DstIP", ipValue{v6.dstIP},
		)
		return false
	}
}

func isIpv6(n *net.IPNet) bool {
	return n.IP.To4() == nil
}
