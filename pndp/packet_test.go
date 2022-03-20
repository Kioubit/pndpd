package pndp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

type checksumTestCase struct {
	payloadHexString string
	want             []byte
}

func TestCalculateChecksum(t *testing.T) {
	//fd00::251d:bbbb:bbbb:bbbb → ff02::1:ff00:99 ICMPv6 Neighbor Solicitation for fd00::99 from ad:ad:ad:ad:ad:ad

	cases := []checksumTestCase{
		{"87 00 1D 12 00 00 00 00 FD 00 00 00 00 00 00 00 00 00 00 00 00 00 00 99 01 01 AD AD AD AD AD AD", //32 (Even)
			[]byte{0x1D, 0x12}},
		{"87 00 1D 12 00 00 00 00 FD 00 00 00 00 00 00 00 00 00 00 00 00 00 00 99 01 01 AD AD AD AD AD", //31 (Not even)
			[]byte{0x1D, 0xC0}},
	}

	testSrcIP := []byte{0xFD, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x25, 0x1D, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}
	testDstIP := []byte{0xFF, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xFF, 0x00, 0x00, 0x99}
	testHeader, _ := newIpv6Header(testSrcIP, testDstIP)

	for _, tc := range cases {
		payloadBytes, err := hex.DecodeString(strings.Join(strings.Fields(tc.payloadHexString), ""))
		if err != nil {
			t.Errorf(err.Error())
		}

		bPayloadLen := make([]byte, 2)
		binary.BigEndian.PutUint16(bPayloadLen, uint16(len(payloadBytes)))
		testHeader.payloadLen = bPayloadLen

		// Clear existing checksum as it should be zero for calculation
		payloadBytes[2] = 0x0
		payloadBytes[3] = 0x0

		got := calculateChecksum(testHeader, payloadBytes)

		bChecksum := make([]byte, 2)
		binary.BigEndian.PutUint16(bChecksum, got)
		if !bytes.Equal(tc.want, bChecksum) {
			t.Errorf("Expected '%x', but got '%x'", tc.want, bChecksum)
		}
	}
}
