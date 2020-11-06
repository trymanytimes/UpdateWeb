package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Version uint32

const (
	Version4 Version = 4
	Version6 Version = 6
)

type Subnet struct {
	restresource.ResourceBase `json:",inline"`
	Ipnet                     string   `json:"ipnet" rest:"required=true,description=immutable" db:"uk"`
	SubnetId                  uint32   `json:"-" rest:"description=readonly"`
	ValidLifetime             uint32   `json:"validLifetime"`
	MaxValidLifetime          uint32   `json:"maxValidLifetime"`
	MinValidLifetime          uint32   `json:"minValidLifetime"`
	DomainServers             []string `json:"domainServers"`
	Routers                   []string `json:"routers"`
	ClientClass               string   `json:"clientClass"`
	RelayAgentAddresses       []string `json:"relayAgentAddresses"`
	Tags                      string   `json:"tags"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string   `json:"usedRatio" rest:"description=readonly" db:"-"`
	Version                   Version  `json:"version" rest:"description=readonly"`
}

type Subnets []*Subnet

func (s Subnets) Len() int {
	return len(s)
}

func (s Subnets) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Subnets) Less(i, j int) bool {
	return s[i].SubnetId < s[j].SubnetId
}
