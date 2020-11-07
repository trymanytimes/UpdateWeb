package resource

import (
	"time"

	"github.com/zdnscloud/gorest/resource"
)

type Node struct {
	resource.ResourceBase `json:",inline"`

	Ip           string               `json:"ip"`
	Roles        []string             `json:"roles"`
	HostName     string               `json:"hostName"`
	NodeIsAlive  bool                 `json:"nodeIsAlive"`
	DhcpIsAlive  bool                 `json:"dhcpIsAlive"`
	DnsIsAlive   bool                 `json:"dnsIsAlive"`
	CpuUsage     []RatioWithTimestamp `json:"cpuUsage" db:"-"`
	MemoryUsage  []RatioWithTimestamp `json:"memoryUsage" db:"-"`
	DiscUsage    []RatioWithTimestamp `json:"discUsage" db:"-"`
	Network      []RatioWithTimestamp `json:"network" db:"-"`
	CpuRatio     string               `json:"cpuRatio"`
	MemRatio     string               `json:"memRatio"`
	Master       string               `json:"master"`
	ControllerIp string               `json:"controllerIP"`
	StartTime    time.Time            `json:"startTime"`
	Vip          string               `json:"vip"`
}

type RatioWithTimestamp struct {
	Timestamp resource.ISOTime `json:"timestamp"`
	Ratio     string           `json:"ratio"`
}
