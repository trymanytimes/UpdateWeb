package handler

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	WSAlarmPath = "/apis/ws.linkingthing.com/v1/alarm"
)

type AlarmMessage struct {
	Type  resource.AlarmState `json:"type"`
	Count uint64              `json:"count"`
}

func (h *AlarmHandler) RegisterWSHandler(router gin.IRoutes) {
	router.GET(WSAlarmPath, func(c *gin.Context) {
		h.OpenAlarm(c.Request, c.Writer)
	})
}

func (h *AlarmHandler) OpenAlarm(r *http.Request, w http.ResponseWriter) {
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		log.Warnf("event websocket upgrade failed %s", err.Error())
		return
	}
	defer conn.Close()

	listener := h.AddListener()
	defer listener.Stop()
	alarmCh := listener.AlarmChan()
	for {
		count, ok := <-alarmCh
		if ok == false {
			break
		}

		err = conn.WriteJSON(&AlarmMessage{Type: resource.AlarmStateUntreated, Count: count})
		if err != nil {
			if util.IsBrokenPipeErr(err) == false {
				log.Warnf("send alarm failed:%s", err.Error())
			}
			break
		}
	}
}

type AlarmListener struct {
	count   uint64
	alarmCh chan uint64
	stopCh  chan struct{}
}

func (l *AlarmListener) Stop() {
	l.stopCh <- struct{}{}
	<-l.stopCh
	close(l.alarmCh)
}

func (h *AlarmHandler) AddListener() *AlarmListener {
	listener := &AlarmListener{
		alarmCh: make(chan uint64),
		stopCh:  make(chan struct{}),
	}

	go h.publishUntreatedCount(listener)
	return listener
}

func (l *AlarmListener) AlarmChan() <-chan uint64 {
	return l.alarmCh
}

func (h *AlarmHandler) publishUntreatedCount(listener *AlarmListener) {
	for {
		select {
		case <-listener.stopCh:
			listener.stopCh <- struct{}{}
			return
		default:
		}

		count := atomic.LoadUint64(&h.untreatedCount)
		if listener.count == count {
			h.lock.Lock()
			h.cond.Wait()
			h.lock.Unlock()
		} else {
			listener.count = count
			listener.alarmCh <- count
		}
	}
}
