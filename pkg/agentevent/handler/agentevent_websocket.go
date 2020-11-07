package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zdnscloud/cement/log"

	"github.com/trymanytimes/UpdateWeb/pkg/agentevent/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

var (
	WSFeedbackPath = "/apis/ws.linkingthing.com/v1/agentevent"
)

type AgentEventListener struct {
	eventCh chan interface{}
	offset  uint64
	stopCh  chan struct{}
}

func (h *AgentEventHandler) RegisterWSHandler(router gin.IRoutes) {
	router.GET(WSFeedbackPath, func(c *gin.Context) {
		h.OpenAgentEvent(c.Request, c.Writer)
	})
}

func (h *AgentEventHandler) OpenAgentEvent(r *http.Request, w http.ResponseWriter) {
	conn, err := websocket.Upgrade(w, r, nil, 0, 0)
	if err != nil {
		log.Warnf("OpenAgentEvent websocket upgrade failed %s", err.Error())
		return
	}
	defer conn.Close()

	listener := h.AddListener()
	broadcastCh := listener.EventNotifyChan()
	defer listener.stop()

	for {
		data, ok := <-broadcastCh
		if !ok {
			break
		}

		if err = conn.WriteJSON(data); err != nil {
			if util.IsBrokenPipeErr(err) == false {
				log.Warnf("send agentEvent websocket failed:%s", err.Error())
			}
			break
		}
	}
}

func (h *AgentEventHandler) AddListener() *AgentEventListener {
	listener := &AgentEventListener{
		eventCh: make(chan interface{}),
		stopCh:  make(chan struct{}),
	}

	go h.publicAgentEvent(listener)
	return listener
}

func (listener *AgentEventListener) stop() {
	listener.stopCh <- struct{}{}
	<-listener.stopCh
	close(listener.eventCh)
}

func (listener *AgentEventListener) EventNotifyChan() <-chan interface{} {
	return listener.eventCh
}

func (h *AgentEventHandler) publicAgentEvent(listener *AgentEventListener) {
	for {
		select {
		case <-listener.stopCh:
			listener.stopCh <- struct{}{}
			return
		default:
		}

		events := h.getEvents(listener.offset)
		eventsLen := len(events)
		if eventsLen == 0 {
			h.lock.Lock()
			h.cond.Wait()
			h.lock.Unlock()
			continue
		}

		listener.offset += uint64(eventsLen)
		for i := eventsLen - 1; i >= 0; i-- {
			select {
			case <-listener.stopCh:
				listener.stopCh <- struct{}{}
				return
			case listener.eventCh <- events[i]:
			}
		}
	}
}

func (h *AgentEventHandler) getEvents(offset uint64) []*resource.AgentEvent {
	h.lock.RLock()
	defer h.lock.RUnlock()

	var events []*resource.AgentEvent
	for e := h.eventList.Back(); e != nil; e = e.Prev() {
		event := e.Value.(*Event)
		if event.index > offset {
			events = append(events, event.AgentEvent)
		} else {
			break
		}
	}

	return events
}
