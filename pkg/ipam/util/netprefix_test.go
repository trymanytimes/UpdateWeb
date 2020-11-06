package util

import (
	"net"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
)

func TestNetPrefixSetSegment(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("2001:503:ba3e::/48")
	finalPrefix := []string{
		"2001:503:ba3e::/56",
		"2001:503:ba3e:4000::/56",
		"2001:503:ba3e:8000::/56",
		"2001:503:ba3e:c000::/56",
	}

	for i := 0; i < 4; i++ {
		prefix, err := FromIPNet(ipnet, 56)
		ut.Assert(t, err == nil, "")
		prefix.SetSegment(48, 2, i)
		ipnet := prefix.ToIPNet()
		ut.Equal(t, finalPrefix[i], ipnet.String())
	}

	_, ipnet, _ = net.ParseCIDR("1.0.0.0/8")
	finalPrefix = []string{
		"1.0.0.0/16",
		"1.64.0.0/16",
		"1.128.0.0/16",
		"1.192.0.0/16",
	}
	for i := 0; i < 4; i++ {
		prefix, err := FromIPNet(ipnet, 16)
		ut.Assert(t, err == nil, "")
		prefix.SetSegment(8, 2, i)
		ipnet := prefix.ToIPNet()
		ut.Equal(t, finalPrefix[i], ipnet.String())
	}
}
