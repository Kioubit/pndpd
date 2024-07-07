package pndp

import (
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
)

func EnableDebugLog() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
}

type hexValue struct {
	arg []byte
}

func (v hexValue) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("%X", v.arg))
}

type ipValue struct {
	arg []byte
}

func (v ipValue) LogValue() slog.Value {
	if len(v.arg) != 16 {
		return slog.StringValue(fmt.Sprintf("%X", v.arg))
	}
	return slog.StringValue(netip.AddrFrom16([16]byte(v.arg)).String())
}

type macValue struct {
	arg []byte
}

func (v macValue) LogValue() slog.Value {
	if len(v.arg) != 6 {
		return slog.StringValue(fmt.Sprintf("%X", v.arg))
	}
	return slog.StringValue(fmt.Sprintf("%X:%X:%X:%X:%X:%X", v.arg[0], v.arg[1], v.arg[2], v.arg[3], v.arg[4], v.arg[5]))
}

// Htons Convert a uint16 to host byte order (big endian)
func htons(v uint16) int      { return int(htons16(v)) }
func htons16(v uint16) uint16 { return (v << 8) | (v >> 8) }

var _, ulaSpace, _ = net.ParseCIDR("fc00::/7")
