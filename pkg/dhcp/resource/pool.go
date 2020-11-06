package resource

import (
	"net"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restresource "github.com/zdnscloud/gorest/resource"
)

type Pool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string   `json:"-" db:"ownby"`
	BeginAddress              string   `json:"beginAddress" rest:"required=true,description=immutable" db:"uk"`
	EndAddress                string   `json:"endAddress" rest:"required=true,description=immutable" db:"uk"`
	DomainServers             []string `json:"domainServers"`
	Routers                   []string `json:"routers"`
	ClientClass               string   `json:"clientClass"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string   `json:"usedRatio" rest:"description=readonly" db:"-"`
	Version                   Version  `json:"version" rest:"description=readonly"`
}

func (p Pool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

func (p *Pool) CheckConflictWithAnother(another *Pool) bool {
	if p.Version == Version4 {
		if util.Ipv4ToUint32(net.ParseIP(p.EndAddress)) < util.Ipv4ToUint32(net.ParseIP(another.BeginAddress)) ||
			util.Ipv4ToUint32(net.ParseIP(another.EndAddress)) < util.Ipv4ToUint32(net.ParseIP(p.BeginAddress)) {
			return false
		}
	} else {
		if util.Ipv6ToBigInt(net.ParseIP(p.EndAddress)).Cmp(util.Ipv6ToBigInt(net.ParseIP(another.BeginAddress))) == -1 ||
			util.Ipv6ToBigInt(net.ParseIP(another.EndAddress)).Cmp(util.Ipv6ToBigInt(net.ParseIP(p.BeginAddress))) == -1 {
			return false
		}
	}

	return true
}

func (p *Pool) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&Pool{BeginAddress: ip, EndAddress: ip})
}

func (p *Pool) Equals(another *Pool) bool {
	return p.Subnet == another.Subnet && p.BeginAddress == another.BeginAddress && p.EndAddress == another.EndAddress
}

type Pools []*Pool

func (p Pools) Len() int {
	return len(p)
}

func (p Pools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pools) Less(i, j int) bool {
	if p[i].Version == Version4 {
		return util.Ipv4ToUint32(net.ParseIP(p[i].BeginAddress)) < util.Ipv4ToUint32(net.ParseIP(p[j].BeginAddress))
	} else {
		return util.Ipv6ToBigInt(net.ParseIP(p[i].BeginAddress)).Cmp(util.Ipv6ToBigInt(net.ParseIP(p[j].BeginAddress))) == -1
	}
}
