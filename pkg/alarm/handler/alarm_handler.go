package handler

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/gomail.v2"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/eventbus"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	AlarmExpireTime         = 3 * 30 * 24 * time.Hour
	MailSubject             = "您好, DDI系统发出告警信息, 请查收!!!"
	DefaultAlarmValidPeriod = 90 //day
)

var (
	AlarmFilterNames = []string{"name", "level", "state"}
)

type AlarmHandler struct {
	defaultAlarmValidPeriod int
	untreatedCount          uint64
	lock                    sync.RWMutex
	cond                    *sync.Cond
}

func NewAlarmHandler() (*AlarmHandler, error) {
	defaultAlarmValidPeriod := DefaultAlarmValidPeriod
	if conf := config.GetConfig(); conf != nil && conf.Alarm.ValidPeriod != 0 {
		defaultAlarmValidPeriod = int(conf.Alarm.ValidPeriod)
	}

	h := &AlarmHandler{
		defaultAlarmValidPeriod: defaultAlarmValidPeriod,
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		count, err := tx.CountEx(resource.TableAlarm, "select count(*) from gr_alarm where state = $1 and expire > now()",
			resource.AlarmStateUntreated)
		if err == nil {
			h.untreatedCount = uint64(count)
		}

		return err
	}); err != nil {
		return nil, err
	}

	go h.run()
	go h.subscribeAlarmEvent()
	h.cond = sync.NewCond(&h.lock)
	return h, nil
}

func (h *AlarmHandler) run() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
				if _, err := tx.Exec("delete from gr_alarm where expire < now()"); err != nil {
					return err
				}

				count, err := tx.CountEx(resource.TableAlarm, "select count(*) from gr_alarm where state = $1 and expire > now()",
					resource.AlarmStateUntreated)
				if err == nil {
					atomic.StoreUint64(&h.untreatedCount, uint64(count))
					h.cond.Broadcast()
				}

				return err
			}); err != nil {
				log.Warnf("delete expired alarms failed: %s", err.Error())
			}
		}
	}
}

func (h *AlarmHandler) subscribeAlarmEvent() {
	alarmEventCh := eventbus.SubscribeResourceEvent(resource.Alarm{})
	for {
		select {
		case event := <-alarmEventCh:
			switch e := event.(type) {
			case eventbus.ResourceCreateEvent:
				alarm := e.Resource.(*resource.Alarm)
				if err := h.add(alarm); err != nil {
					log.Warnf("add alarm failed: %s", err.Error())
				}
			}
		}
	}
}

func (h *AlarmHandler) add(alarm *resource.Alarm) error {
	if exists, err := hasUntreatedAlarm(alarm); err != nil {
		return err
	} else if exists {
		return nil
	}

	alarm.Expire = time.Now().AddDate(0, 0, h.defaultAlarmValidPeriod)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(alarm)
		return err
	}); err != nil {
		return fmt.Errorf("insert alarm to db failed: %s", err.Error())
	}

	atomic.AddUint64(&h.untreatedCount, 1)
	h.cond.Broadcast()
	if err := h.sendMail(alarm); err != nil {
		return fmt.Errorf("send mail failed: %s", err.Error())
	}

	return nil
}

func hasUntreatedAlarm(alarm *resource.Alarm) (bool, error) {
	conditions := map[string]interface{}{"name": alarm.Name, "state": string(resource.AlarmStateUntreated)}
	switch resource.ThresholdName(alarm.Name) {
	case resource.ThresholdNameCpuUsedRatio, resource.ThresholdNameMemoryUsedRatio, resource.ThresholdNameStorageUsedRatio,
		resource.ThresholdNameNodeOffline, resource.ThresholdNameDNSOffline, resource.ThresholdNameDHCPOffline,
		resource.ThresholdNameQPS, resource.ThresholdNameLPS:
		conditions["node_ip"] = alarm.NodeIp
	case resource.ThresholdNameSubnetUsedRatio:
		conditions["node_ip"] = alarm.NodeIp
		conditions["subnet"] = alarm.Subnet
	case resource.ThresholdNameIPConflict:
		conditions["conflict_ip"] = alarm.ConflictIp
		conditions["conflict_ip_type"] = alarm.ConflictIpType
	default:
		return false, nil
	}

	existsUntreated := false
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exists, err := tx.Exists(resource.TableAlarm, conditions)
		existsUntreated = exists
		return err
	})
	return existsUntreated, err
}

func (h *AlarmHandler) sendMail(alarm *resource.Alarm) error {
	if alarm.SendMail == false {
		return nil
	}

	var senders []*resource.MailSender
	if err := db.GetResources(map[string]interface{}{restdb.IDField: MailSenderUID}, &senders); err != nil {
		return err
	}

	if len(senders) == 0 || senders[0].Enabled == false {
		return nil
	}

	var receivers []*resource.MailReceiver
	if err := db.GetResources(nil, &receivers); err != nil {
		return err
	}

	if len(receivers) == 0 {
		return nil
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", senders[0].Username)
	msg.SetHeader("Subject", MailSubject)
	msg.SetBody("text/plain", genLocalizationMessage(alarm))
	for _, receiver := range receivers {
		msg.SetHeader("To", receiver.Address)
		if err := gomail.NewDialer(senders[0].Host, senders[0].Port, senders[0].Username, senders[0].Password).DialAndSend(msg); err != nil {
			log.Warnf("send alarm mail to %s failed: %s", receiver.Address, err.Error())
		}
	}

	return nil
}

func (h *AlarmHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var alarms []*resource.Alarm
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		sql, args := util.GenSqlAndArgsByFileters(resource.TableAlarm, AlarmFilterNames, h.defaultAlarmValidPeriod, ctx.GetFilters())
		return tx.FillEx(&alarms, sql, args...)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list alarms faield: %s", err.Error()))
	}

	return alarms, nil
}

func (h *AlarmHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	alarm := ctx.Resource.(*resource.Alarm)
	if resource.AlarmState(alarm.State) != resource.AlarmStateSolved && resource.AlarmState(alarm.State) != resource.AlarmStateIgnored {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("unsupport update alarm state to %s", alarm.State))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableAlarm, map[string]interface{}{"state": alarm.State}, map[string]interface{}{restdb.IDField: alarm.GetID()})
		if err == nil {
			if count := atomic.LoadUint64(&h.untreatedCount); count > 0 {
				atomic.StoreUint64(&h.untreatedCount, count-1)
				h.cond.Broadcast()
			}
		}

		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update alarm %s faield: %s", alarm.GetID(), err.Error()))
	}

	return alarm, nil
}
