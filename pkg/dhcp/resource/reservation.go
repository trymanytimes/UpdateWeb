package resource

import (
	"net"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restresource "github.com/zdnscloud/gorest/resource"
)

type Reservation struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string   `json:"-" db:"ownby"`
	HwAddress                 string   `json:"hwAddress" rest:"required=true" db:"uk"`
	IpAddress                 string   `json:"ipAddress" rest:"required=true"`
	DomainServers             []string `json:"domainServers"`
	Routers                   []string `json:"routers"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string   `json:"usedRatio" rest:"description=readonly" db:"-"`
	Version                   Version  `json:"version" rest:"description=readonly"`
}

func (r Reservation) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

type Reservations []*Reservation

func (r Reservations) Len() int {
	return len(r)
}

func (r Reservations) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Reservations) Less(i, j int) bool {
	if r[i].Version == Version4 {
		return util.Ipv4ToUint32(net.ParseIP(r[i].IpAddress)) < util.Ipv4ToUint32(net.ParseIP(r[j].IpAddress))
	} else {
		return util.Ipv6ToBigInt(net.ParseIP(r[i].IpAddress)).Cmp(util.Ipv6ToBigInt(net.ParseIP(r[j].IpAddress))) == -1
	}
}
