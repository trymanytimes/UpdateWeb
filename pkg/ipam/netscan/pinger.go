package netscan

import (
	"math"
	"net"
	"time"
)

const (
	SmallSize      = 100
	DefaultTimeout = 3 * time.Second

	BatchSize    = 1024
	BatchTimeout = 5 * time.Second
)

type Pinger struct {
	id int
}

func NewPinger() *Pinger {
	return &Pinger{
		id: 0,
	}
}

//note: Ping isn't thread safe
func (p *Pinger) Ping(ips []net.IP) (<-chan []bool, <-chan error) {
	resultCh := make(chan []bool)
	errorCh := make(chan error, 1)
	go func() {
		p.pingHelper(ips, resultCh, errorCh)
		close(resultCh)
		close(errorCh)
	}()
	return resultCh, errorCh
}

func (p *Pinger) pingHelper(ips []net.IP, resultCh chan<- []bool, errorCh chan<- error) {
	ipCount := len(ips)
	if ipCount == 0 {
		return
	}

	if ipCount < BatchSize {
		timeout := DefaultTimeout
		if ipCount > SmallSize {
			timeout = BatchTimeout
		}
		task, err := newPingTask(p.allocateID(), ips)
		if err != nil {
			errorCh <- err
			return
		}
		resultCh <- task.Run(timeout)
	} else {
		task, err := newPingTask(p.allocateID(), ips[:BatchSize])
		if err != nil {
			errorCh <- err
			return
		}
		resultCh <- task.Run(BatchTimeout)
		p.pingHelper(ips[BatchSize:], resultCh, errorCh)
	}
}

func (p *Pinger) allocateID() int {
	id := p.id
	p.id += 1
	if p.id == math.MaxInt16 {
		p.id = 0
	}
	return id
}
