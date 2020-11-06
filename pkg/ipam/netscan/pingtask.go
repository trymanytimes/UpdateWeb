package netscan

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var (
	IPv4Proto        = "ip4:icmp"
	IPv6Proto        = "ip6:ipv6-icmp"
	ProtocolICMP     = 1
	ProtocolIPv6ICMP = 58
)

const (
	MaxResendCount = 3
	ResendInterval = 1 * time.Second
	ReceiveTimeout = time.Millisecond * 100
)

type PingTask struct {
	wg    sync.WaitGroup
	addrs map[string]*net.IPAddr
	isV4  bool

	ips            []net.IP
	ipPos          map[string]int
	reachableHosts map[string]struct{}

	id       int
	sequence int
	conn     *icmp.PacketConn
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func newPingTask(id int, ips []net.IP) (*PingTask, error) {
	addrs := make(map[string]*net.IPAddr)
	ipPos := make(map[string]int)
	for i, ip := range ips {
		ipstr := ip.String()
		if _, ok := addrs[ipstr]; ok {
			return nil, fmt.Errorf("duplicate address %s", ipstr)
		}
		addrs[ipstr] = &net.IPAddr{
			IP:   ips[i],
			Zone: "",
		}
		ipPos[ipstr] = i
	}

	proto := IPv4Proto
	isV4 := len(ips[0].To4()) == 4
	if !isV4 {
		proto = IPv6Proto
	}
	conn, err := icmp.ListenPacket(proto, "")
	if err != nil {
		return nil, err
	}

	return &PingTask{
		id:             id,
		sequence:       rand.Intn(math.MaxInt16),
		addrs:          addrs,
		ipPos:          ipPos,
		ips:            ips,
		isV4:           isV4,
		conn:           conn,
		reachableHosts: make(map[string]struct{}),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}, nil
}

func (t *PingTask) Run(timeout time.Duration) []bool {
	go func() {
		<-time.After(timeout)
		close(t.stopCh)
	}()
	t.ping()
	reachabilities := make([]bool, len(t.ips))
	for ipstr := range t.reachableHosts {
		reachabilities[t.ipPos[ipstr]] = true
	}
	return reachabilities
}

func (t *PingTask) ping() {
	reachableAddrCh := make(chan []*net.IPAddr)
	t.wg.Add(2)
	go t.recv(reachableAddrCh)
	go t.send(reachableAddrCh)
	t.wg.Wait()
	close(reachableAddrCh)
	t.conn.Close()
}

//ping all addrs, then wait for recv handler to report
//addrs which already get response, then ping the left
//addr, recv handler report result every ResendInterval
//this interval make sure ping won't send too often
func (t *PingTask) send(reachableAddrCh <-chan []*net.IPAddr) {
	var typ icmp.Type = ipv4.ICMPTypeEcho
	if !t.isV4 {
		typ = ipv6.ICMPTypeEchoRequest
	}
	defer t.wg.Done()
	sendCount := 0
	for {
		if sendCount < MaxResendCount {
			for _, addr := range t.addrs {
				if err := t.sendOnePkt(addr, typ); err != nil {
					fmt.Printf("send %s failed:%s\n", addr.String(), err.Error())
				}
			}
			sendCount += 1
		}

		select {
		case <-t.stopCh:
			return
		case addrs := <-reachableAddrCh:
			for _, addr := range addrs {
				ip := addr.IP.String()
				if _, ok := t.addrs[ip]; ok {
					t.reachableHosts[ip] = struct{}{}
					delete(t.addrs, ip)
				}
			}
			if len(t.addrs) == 0 {
				close(t.doneCh)
				return
			}
		}
	}
}

func (t *PingTask) sendOnePkt(addr *net.IPAddr, typ icmp.Type) error {
	bs, err := (&icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{
			ID:  t.id,
			Seq: t.sequence,
			//without appending some bytes after time, some host (like 114.114.114.114)
			//willn't echo, :(
			Data: append(timeToBytes(time.Now()), intToBytes(1000)...),
		},
	}).Marshal(nil)
	if err == nil {
		_, err = t.conn.WriteTo(bs, addr)
	}
	return err
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func (t *PingTask) recv(reachableAddrCh chan<- []*net.IPAddr) {
	proto := ProtocolICMP
	if !t.isV4 {
		proto = ProtocolIPv6ICMP
	}

	resendTimer := time.NewTicker(ResendInterval)
	defer resendTimer.Stop()
	defer t.wg.Done()

	var reachableAddrs []*net.IPAddr
	for {
		select {
		case <-t.stopCh:
			return
		case <-t.doneCh:
			return
		case <-resendTimer.C:
			select {
			case reachableAddrCh <- reachableAddrs:
				reachableAddrs = nil
			default:
			}
		default:
			addr, err := t.receiveOnePkt(proto)
			if err == nil {
				reachableAddrs = append(reachableAddrs, addr)
			}
		}
	}
}

func (t *PingTask) receiveOnePkt(proto int) (*net.IPAddr, error) {
	buf := make([]byte, 512)
	t.conn.SetReadDeadline(time.Now().Add(ReceiveTimeout))
	l, addr, err := t.conn.ReadFrom(buf)
	if err != nil {
		return nil, err
	}
	buf = buf[:l]
	if proto == ProtocolICMP {
		buf, err = ipv4Payload(buf)
		if err != nil {
			return nil, err
		}
	}

	m, err := icmp.ParseMessage(proto, buf)
	if err != nil {
		return nil, err
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		return nil, fmt.Errorf("invalid return type:%v\n", m.Type)
	}

	pkt, ok := m.Body.(*icmp.Echo)
	if !ok {
		return nil, fmt.Errorf("invalid return type")
	}

	if pkt.ID == t.id && pkt.Seq == t.sequence {
		return addr.(*net.IPAddr), nil
	} else {
		return nil, fmt.Errorf("id or seq isn't match")
	}
}

func ipv4Payload(b []byte) ([]byte, error) {
	if len(b) < ipv4.HeaderLen {
		return nil, fmt.Errorf("pkt is too short")
	}

	hdrlen := int(b[0]&0x0f) << 2
	if len(b) <= hdrlen {
		return nil, fmt.Errorf("pkt is too short")
	} else {
		return b[hdrlen:], nil
	}
}

func intToBytes(tracker int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(tracker))
	return b
}
