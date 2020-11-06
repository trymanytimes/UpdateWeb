package util

import (
	"fmt"
	"net"
)

type NetPrefix struct {
	prefix  uint64
	maskLen int
	total   int
}

func FromIPNet(prefix *net.IPNet, maskLen int) (np NetPrefix, err error) {
	prefixMaskLen, total := prefix.Mask.Size()
	np.total = total
	if total == 128 {
		if prefixMaskLen > 64 {
			err = fmt.Errorf("ipv6 prefix is too long")
			return
		}

		if maskLen > 64 {
			err = fmt.Errorf("mask of ipv6 address should equal or smaller than 64")
			return
		}

		np.prefix = bytesToIpv6Prefix(prefix.IP.To16()[:8])
		np.maskLen = maskLen
		return
	} else {
		if maskLen > 24 {
			err = fmt.Errorf("mask of ipv4 address should equal or smaller than 24")
			return
		}
		np.prefix = uint64(bytesToIpv4Prefix(prefix.IP.To4()[:]))
		np.maskLen = maskLen
	}
	return
}

func bytesToIpv6Prefix(bs []byte) uint64 {
	return (uint64(bs[0]) << 56) |
		(uint64(bs[1]) << 48) |
		(uint64(bs[2]) << 40) |
		(uint64(bs[3]) << 32) |
		(uint64(bs[4]) << 24) |
		(uint64(bs[5]) << 16) |
		(uint64(bs[6]) << 8) |
		uint64(bs[7])
}

func bytesToIpv4Prefix(bs []byte) uint32 {
	return (uint32(bs[0]) << 24) |
		(uint32(bs[1]) << 16) |
		(uint32(bs[2]) << 8) |
		uint32(bs[3])
}

func (np *NetPrefix) SetSegment(pos int, width int, val int) error {
	l := 64
	if np.total != 128 {
		l = 32
	}
	if pos+width > l {
		return fmt.Errorf("segment position is out of index")
	}

	seg := uint64(val) << (l - pos - width)
	np.prefix |= seg
	return nil
}

func (np NetPrefix) ToIPNet() *net.IPNet {
	prefix := np.prefix
	if np.total == 128 {
		bs := []byte{
			byte(prefix >> 56),
			byte(prefix >> 48 & 0xff),
			byte(prefix >> 40 & 0xff),
			byte(prefix >> 32 & 0xff),
			byte(prefix >> 24 & 0xff),
			byte(prefix >> 16 & 0xff),
			byte(prefix >> 8 & 0xff),
			byte(prefix & 0xff),
			0, 0, 0, 0, 0, 0, 0, 0,
		}
		return &net.IPNet{
			IP:   net.IP(bs),
			Mask: net.CIDRMask(np.maskLen, 128),
		}
	} else {
		bs := []byte{
			byte(prefix >> 24),
			byte(prefix >> 16 & 0xff),
			byte(prefix >> 8 & 0xff),
			byte(prefix & 0xff),
		}
		return &net.IPNet{
			IP:   net.IP(bs),
			Mask: net.CIDRMask(np.maskLen, 32),
		}
	}
}

func (np NetPrefix) Dump() {
	if np.total == 128 {
		s := fmt.Sprintf("%064b", np.prefix)
		for i := 0; i < 8; i++ {
			fmt.Printf("%s,", s[(i*8):(i+1)*8])
		}
		fmt.Printf("\n")
	} else {
		s := fmt.Sprintf("%064b", np.prefix)[32:]
		for i := 0; i < 4; i++ {
			fmt.Printf("%s,", s[(i*8):(i+1)*8])
		}
		fmt.Printf("\n")
	}
}
