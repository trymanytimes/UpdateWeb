package handler

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"

	agentkafkaproducer "github.com/linkingthing/ddi-agent/pkg/kafkaproducer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/agentevent/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
)

var (
	TableAgentEvent = restdb.ResourceDBType(&resource.AgentEvent{})
	deleteAgentSql  = `delete from gr_agent_event where create_time < $1 or create_time < $2`
)

const (
	maxMessageLen int = 1000
)

type AgentEventHandler struct {
	lock        sync.RWMutex
	cond        *sync.Cond
	eventList   *list.List
	eventOffset uint64
}

type Event struct {
	index uint64
	*resource.AgentEvent
}

func NewAgentEventHandler() (*AgentEventHandler, error) {
	h := &AgentEventHandler{eventList: list.New()}
	h.cond = sync.NewCond(&h.lock)

	if err := h.loadEventList(); err != nil {
		return nil, err
	}

	go h.runTicker()
	go h.runKafkaConsumer()
	return h, nil
}

func (h *AgentEventHandler) runKafkaConsumer() {
	kafkaReader := kg.NewReader(kg.ReaderConfig{
		Brokers:        config.GetConfig().Kafka.Addr,
		Topic:          agentkafkaproducer.AgentEventTopic,
		GroupID:        config.GetConfig().Kafka.GroupIdAgentEvent,
		MinBytes:       1,
		MaxBytes:       1e6,
		MaxWait:        time.Millisecond * 100,
		SessionTimeout: time.Second * 10,
		Dialer: &kg.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})

	defer kafkaReader.Close()
	for {
		message, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read dns message from agent kafka failed: %s", err.Error())
			continue
		}

		switch string(message.Key) {
		case agentkafkaproducer.AgentEvent:
			if err = h.pushAgentEvent(message.Value); err != nil {
				log.Errorf("pushAgentEvent failed:%s", err.Error())
			}
		}
	}
}

func (h *AgentEventHandler) pushAgentEvent(message []byte) error {
	var ddiResponse pb.DDIResponse
	if err := proto.Unmarshal(message, &ddiResponse); err != nil {
		return fmt.Errorf("pushAgentEvent Unmarshal message error:%s", err.Error())
	}

	var resourceStr, methodStr string
	if ddiResponse.Header != nil {
		resourceStr = ddiResponse.Header.Resource
		methodStr = ddiResponse.Header.Method
	}

	agentEvent := &resource.AgentEvent{
		Node:          ddiResponse.Node,
		NodeType:      ddiResponse.NodeType,
		Resource:      resourceStr,
		Method:        methodStr,
		Succeed:       ddiResponse.Succeed,
		ErrorMessage:  ddiResponse.ErrorMessage,
		CmdMessage:    ddiResponse.CmdMessage,
		OperationTime: ddiResponse.OperationTime,
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(agentEvent); err != nil {
			return fmt.Errorf("insert agentEvent failed:%s ", err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	h.addEventList(agentEvent)
	return nil
}

func (h *AgentEventHandler) addEventList(agentEvent *resource.AgentEvent) {
	h.lock.Lock()
	if h.eventList.Len() > maxMessageLen {
		h.eventList.Remove(h.eventList.Front())
	}
	h.eventList.PushBack(&Event{index: h.eventOffset + 1, AgentEvent: agentEvent})
	h.lock.Unlock()

	atomic.AddUint64(&h.eventOffset, 1)
	h.cond.Broadcast()
}

func (h *AgentEventHandler) runTicker() {
	ticker := time.NewTicker(time.Hour * 24 * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := h.cleanAgentEventHistory(); err != nil {
				log.Warnf("cleanAgentEventHistory error:%s", err.Error())
			}
		}
	}
}

func (h *AgentEventHandler) cleanAgentEventHistory() error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		monthAgo := time.Now().AddDate(0, -1, 0)
		if _, err := tx.Exec("delete from gr_agent_event where create_time < $1", monthAgo); err != nil {
			return fmt.Errorf("delete gr_agent_event cleanPreMonthData data failed:%s", err.Error())
		}

		count, err := tx.Count(TableAgentEvent, map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("count gr_agent_event failed:%s", err.Error())
		}

		if int(count) <= maxMessageLen {
			return nil
		}

		_, err = tx.Exec("delete from gr_agent_event where id in (select id from gr_agent_event order by create_time limit $1)", maxMessageLen)
		if err != nil {
			return fmt.Errorf("delete from gr_agent_event failed:%s", err.Error())
		}

		return nil
	})
}

func (h *AgentEventHandler) loadEventList() error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var agentEventList []*resource.AgentEvent
		err := tx.Fill(map[string]interface{}{
			"orderby": "create_time desc", "limit": maxMessageLen, "offset": 0}, &agentEventList)
		if err != nil {
			return fmt.Errorf("list gr_agent_event failed:%s", err.Error())
		}

		if len(agentEventList) == 0 {
			return nil
		}

		monthAgo := time.Now().AddDate(0, -1, 0)
		if _, err := tx.Exec(deleteAgentSql, monthAgo,
			agentEventList[len(agentEventList)-1].GetCreationTimestamp()); err != nil {
			return fmt.Errorf("delete gr_agent_event cleanPreMonthData data failed:%s", err.Error())
		}

		for i, agentEvent := range agentEventList {
			h.eventList.PushBack(&Event{index: uint64(i + 1), AgentEvent: agentEvent})
		}

		atomic.StoreUint64(&h.eventOffset, uint64(h.eventList.Len()))
		return nil
	})
}
