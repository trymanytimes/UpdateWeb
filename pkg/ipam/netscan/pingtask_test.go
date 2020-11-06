package netscan

import (
	"net"
	"testing"
	"time"

	ut "github.com/zdnscloud/cement/unittest"
)

func testPinger(t *testing.T) {
	cases := []struct {
		ips         []string
		isReachable []bool
	}{
		{[]string{"114.114.114.114",
			"8.8.8.8",
			"119.29.29.29",
			"223.5.5.5",
			"223.5.5.6",
			"101.226.4.6",
			"2.2.3.3",
			"2.2.2.3",
			"2.2.3.4",
			"2.2.4.3",
			"2.2.3.5",
			"2.2.5.3"},
			[]bool{true, true, true, true, true, true, false, false, false, false, false, false},
		},
		{[]string{"::1", "fe80::42:aeff:fe14:db1e"},
			[]bool{true, true},
		},
	}

	for i, c := range cases {
		ips := make([]net.IP, len(c.ips))
		for j, ip := range c.ips {
			ips[j] = net.ParseIP(ip)
		}
		task, err := newPingTask(i, ips)
		ut.Assert(t, err == nil, "get err:%v", err)
		results := task.Run(3 * time.Second)
		for i, reachable := range c.isReachable {
			ut.Assert(t, reachable == results[i], "ip %s should %v", c.ips[i], reachable)
		}
	}
}
