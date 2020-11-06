package util

import (
	"net"
	"strings"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
)

func TestParseV4Mask(t *testing.T) {
	cases := []struct {
		maskInIp string
		maskLen  int
	}{
		{"255.0.0.0", 8},
		{"255.255.0.0", 16},
		{"255.255.255.0", 24},
		{"255.240.0.0", 12},
		{"255.255.128.0", 17},
	}

	for _, c := range cases {
		mask, _ := ParseIPv4Mask(c.maskInIp)
		ut.Equal(t, mask, net.CIDRMask(c.maskLen, 32))
	}
}

func TestParseAddress(t *testing.T) {
	for _, addr := range []string{
		"1.1.1.1",
		"192.168.0.0",
		"172.16.0.0",
	} {
		ip, _ := ParseIPv4Address(strings.Split(addr, "."))
		ut.Equal(t, net.ParseIP(addr).To4(), ip)
	}

	for _, c := range []struct {
		ipInByte string
		ipInHex  string
	}{
		{"252.128.0.0.0.0.0.0.174.31.107.255.254.221.0.1", "fc80::ae1f:6bff:fedd:1"},
		{"253.0.0.16.0.0.0.0.0.0.0.0.0.0.2.84", "fd00:10::254"},
	} {
		ip, _ := ParseIPv6Address(strings.Split(c.ipInByte, "."))
		ut.Equal(t, net.ParseIP(c.ipInHex).To16(), ip)
	}
}
