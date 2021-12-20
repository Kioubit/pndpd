package main

import (
	"encoding/binary"
)

var emptyIpv6 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type Payload interface {
	constructPacket() ([]byte, int)
}

type IPv6Header struct {
	protocol   byte
	srcIP      []byte
	dstIP      []byte
	payloadLen []byte
	payload    []byte
}

func newIpv6Header(srcIp []byte, dstIp []byte) *IPv6Header {
	return &IPv6Header{dstIP: dstIp, srcIP: srcIp, protocol: 0x3a}
}

func (h *IPv6Header) addPayload(payload Payload) {
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

func (h *IPv6Header) constructPacket() []byte {
	header := []byte{
		0x60,            // v6
		0,               // qos
		0,               // qos
		0,               // qos
		h.payloadLen[0], // Payload Length
		h.payloadLen[1], // Payload Length
		h.protocol,      // Protocol next header
		0xff,            // Hop limit
	}
	final := append(header, h.srcIP...)
	final = append(final, h.dstIP...)
	final = append(final, h.payload...)
	return final
}

type NDPAdvPayload struct {
	answeringForIP []byte
	mac            []byte
}

func newNdpPacket(answeringForIP []byte, mac []byte) *NDPAdvPayload {
	return &NDPAdvPayload{
		answeringForIP: answeringForIP,
		mac:            mac,
	}
}

func (p *NDPAdvPayload) constructPacket() ([]byte, int) {
	header := []byte{
		0x88, // Type: Neighbor Advertisement
		0x0,  // Code
		0x0,  // Checksum filled in later
		0x0,  // Checksum filled in later
		0x60, // Flags (Solicited,Override)
		0x0,  // Reserved
		0x0,  // Reserved
		0x0,  // Reserved
	}
	if len(p.answeringForIP) != 16 {
		panic("malformed IP")
	} //TODO check IP lengths on assign everywhere
	final := append(header, p.answeringForIP...)

	secondHeader := []byte{
		0x02, // Type: Target link-layer address (2)
		0x01, // Length: 1 (8 bytes)
	}
	final = append(final, secondHeader...)

	final = append(final, p.mac...)
	return final, 2
}

func calculateChecksum(h *IPv6Header, payload []byte) uint16 {
	sumPseudoHeader := checksumAddition(h.srcIP) + checksumAddition(h.dstIP) + checksumAddition([]byte{0x00, h.protocol}) + checksumAddition(h.payloadLen)
	sumPayload := checksumAddition(payload)
	sumTotal := sumPayload + sumPseudoHeader
	for sumTotal>>16 > 0x0 {
		sumTotal = (sumTotal & 0xffff) + (sumTotal >> 16)
	}
	return uint16(sumTotal) ^ 0xFFFF

}
func checksumAddition(b []byte) uint32 {
	var sum uint32 = 0
	for i := 0; i < len(b); i++ {
		if i%2 == 0 {
			sum += uint32(uint16(b[i])<<8 | uint16(b[i+1]))
		}
	}
	return sum
}
