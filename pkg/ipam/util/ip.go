package util

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ParseIPv4Mask(s string) (net.IPMask, error) {
	segs, err := StringArrayToBytes(strings.Split(s, "."), 4)
	if err != nil {
		return nil, err
	}
	return net.IPv4Mask(segs[0], segs[1], segs[2], segs[3]), nil
}

func ParseIPv4Address(ss []string) (net.IP, error) {
	segs, err := StringArrayToBytes(ss, 4)
	if err != nil {
		return nil, err
	} else {
		return net.IP(segs), nil
	}
}

func ParseIPv6Address(ss []string) (net.IP, error) {
	segs, err := StringArrayToBytes(ss, 16)
	if err != nil {
		return nil, err
	} else {
		return net.IP(segs), nil
	}
}

func StringArrayToBytes(ss []string, byteCount int) ([]byte, error) {
	if len(ss) != byteCount {
		return nil, fmt.Errorf("%v should have %d segments", ss, byteCount)
	}

	segs := make([]byte, 0, byteCount)
	for _, s := range ss {
		seg, err := strconv.Atoi(s)
		if err != nil || seg < 0 || seg > 255 {
			return nil, fmt.Errorf("invlid segment %s", s)
		}
		segs = append(segs, byte(seg))
	}
	return segs, nil
}
