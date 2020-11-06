package resource

import (
	"net"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restresource "github.com/zdnscloud/gorest/resource"
)

type PdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string   `json:"-" db:"ownby"`
	Prefix                    string   `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32   `json:"prefixLen" rest:"required=true"`
	DelegatedLen              uint32   `json:"delegatedLen" rest:"required=true"`
	DomainServers             []string `json:"domainServers"`
	ClientClass               string   `json:"clientClass"`
}

func (p PdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

type PdPools []*PdPool

func (p PdPools) Len() int {
	return len(p)
}

func (p PdPools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p PdPools) Less(i, j int) bool {
	return util.Ipv6ToBigInt(net.ParseIP(p[i].Prefix)).Cmp(util.Ipv6ToBigInt(net.ParseIP(p[j].Prefix))) == -1
}
