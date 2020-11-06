package resource

import (
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/eventbus"
)

type AlarmState string

const (
	AlarmStateUntreated AlarmState = "untreated"
	AlarmStateSolved    AlarmState = "solved"
	AlarmStateIgnored   AlarmState = "ignored"
)

type Alarm struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string    `json:"name" rest:"description=readonly"`
	Level                     string    `json:"level" rest:"description=readonly"`
	State                     string    `json:"state" rest:"options=untreated|solved|ignored"`
	NodeIp                    string    `json:"nodeIp" rest:"description=readonly"`
	Subnet                    string    `json:"subnet" rest:"description=readonly"`
	Value                     uint64    `json:"value" rest:"description=readonly"`
	Threshold                 uint64    `json:"threshold" rest:"description=readonly"`
	ThresholdType             string    `json:"thresholdType" rest:"description=readonly"`
	MasterIp                  string    `json:"masterIp" rest:"description=readonly"`
	SlaveIp                   string    `json:"slaveIp" rest:"description=readonly"`
	HaCmd                     string    `json:"haCmd" rest:"description=readonly"`
	ConflictIp                string    `json:"conflictIp" rest:"description=readonly"`
	ConflictIpType            string    `json:"conflictIpType" rest:"description=readonly"`
	SendMail                  bool      `json:"-"`
	Time                      time.Time `json:"-"`
	Expire                    time.Time `json:"-"`
}

var TableAlarm = restdb.ResourceDBType(&Alarm{})

type AlarmEvent struct {
	Alarm
}

func NewEvent() *AlarmEvent {
	return &AlarmEvent{
		Alarm{
			State: string(AlarmStateUntreated),
		},
	}
}

func (a *AlarmEvent) Node(ip string) *AlarmEvent {
	a.Alarm.NodeIp = ip
	return a
}

func (a *AlarmEvent) Name(name ThresholdName) *AlarmEvent {
	a.Alarm.Name = string(name)
	return a
}

func (a *AlarmEvent) Level(level ThresholdLevel) *AlarmEvent {
	a.Alarm.Level = string(level)
	return a
}

func (a *AlarmEvent) Subnet(subnet string) *AlarmEvent {
	a.Alarm.Subnet = subnet
	return a
}

func (a *AlarmEvent) Time(t time.Time) *AlarmEvent {
	a.Alarm.Time = t
	return a
}

func (a *AlarmEvent) Value(value uint64) *AlarmEvent {
	a.Alarm.Value = value
	return a
}

func (a *AlarmEvent) Threshold(threshold uint64) *AlarmEvent {
	a.Alarm.Threshold = threshold
	return a
}

func (a *AlarmEvent) ThresholdType(typ ThresholdType) *AlarmEvent {
	a.Alarm.ThresholdType = string(typ)
	return a
}

func (a *AlarmEvent) MasterIp(ip string) *AlarmEvent {
	a.Alarm.MasterIp = ip
	return a
}

func (a *AlarmEvent) SlaveIp(ip string) *AlarmEvent {
	a.Alarm.SlaveIp = ip
	return a
}

func (a *AlarmEvent) HaCmd(cmd string) *AlarmEvent {
	a.Alarm.HaCmd = cmd
	return a
}

func (a *AlarmEvent) SendMail(send bool) *AlarmEvent {
	a.Alarm.SendMail = send
	return a
}

func (a *AlarmEvent) ConflictIp(ip string) *AlarmEvent {
	a.Alarm.ConflictIp = ip
	return a
}

func (a *AlarmEvent) ConflictIpType(ipType string) *AlarmEvent {
	a.Alarm.ConflictIpType = ipType
	return a
}

func (a *AlarmEvent) Publish() {
	eventbus.PublishResourceCreateEvent(&a.Alarm)
}
