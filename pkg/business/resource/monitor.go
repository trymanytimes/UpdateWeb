package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type HomePage struct {
	resource.ResourceBase `json:",inline"`
	NormalDomains         uint32 `json:"normalDomains" rest:"description=readonly"`
	AbnormalDomains       uint32 `json:"abnormalDomains" rest:"description=readonly"`
	DomainsCount          uint32 `json:"domainCount" rest:"description=readonly"`
	NormalIPv6Address     uint32 `json:"normalVIP6Count" rest:"description=readonly"`
	AbnormalIPv6Address   uint32 `json:"abnormalVIP6Count" rest:"description=readonly"`
	IPv6AddressCount      uint32 `json:"IPv6AddressesCount" rest:"description=readonly"`
	NormalNode            uint32 `json:"NormalNodeCount" rest:"description=readonly"`
	AbnormalNode          uint32 `json:"abnormalNodeCount" rest:"description=readonly"`
	NodesCount            uint32 `json:"nodesCount" rest:"description=readonly"`
	//DomainVisits          []DomainVisit `json:"domainVisits" rest:"description=readonly"`
	GroupVisit []*GroupVisit `json:"roupVisits" rest:"description=readonly"`
}

/*type DomainVisit struct {
	Domain         string
	IPv6VisitCount int
	IPv4VisitCount int
}*/

type WebsiteVisit struct {
	Domain string
	Count  uint64
}

type GroupVisit struct {
	WebsiteVisits []*WebsiteVisit
	Count         uint64
}
