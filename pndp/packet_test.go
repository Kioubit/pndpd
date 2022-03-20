package pndp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	//fd00::251d:bbbb:bbbb:bbbb → ff02::1:ff00:99 ICMPv6 Neighbor Solicitation for fd00::99 from ad:ad:ad:ad:ad:ad

	type testCase struct {
		payloadHexString string
		want             []byte
	}

	cases := []testCase{
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

func TestCheckPacketChecksum(t *testing.T) {
	type testCase struct {
		payloadHexString string
		want             bool
	}

	cases := []testCase{
		{"87 00 1D 12 00 00 00 00 FD 00 00 00 00 00 00 00 00 00 00 00 00 00 00 99 01 01 AD AD AD AD AD AD", //32 (Even)
			true},
		{"87 00 1D 12 00 00 00 00 FD 00 00 00 00 00 00 00 00 00 00 00 00 00 00 99 01 01 AD AD AD AD AD", //31 (Not even)
			false},
	}

	testSrcIP := []byte{0xFD, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x25, 0x1D, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}
	testDstIP := []byte{0xFF, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xFF, 0x00, 0x00, 0x99}
	testHeader, _ := newIpv6Header(testSrcIP, testDstIP)

	for _, tc := range cases {
		payloadBytes, err := hex.DecodeString(strings.Join(strings.Fields(tc.payloadHexString), ""))
		if err != nil {
			t.Errorf(err.Error())
		}
		got := checkPacketChecksum(testHeader, []byte(payloadBytes))
		if tc.want != got {
			t.Errorf("Excpected valid: '%t', but got valid: '%t' with payload '%x'", tc.want, got, payloadBytes)
		}

	}
}

func TestIsIpv6(t *testing.T) {
	type testCase struct {
		ip   string
		want bool
	}
	cases := []testCase{
		{"0.0.0.0", false},
		{"fd", false},
		{"fd::", true},
		{"fd00::", true},
	}

	for _, tc := range cases {
		if isIpv6(tc.ip) != tc.want {
			t.Errorf("Expected '%t', but got '%t' for %s", tc.want, !tc.want, tc.ip)
		}
	}

}
