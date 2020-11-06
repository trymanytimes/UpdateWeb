package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Dns struct {
	restresource.ResourceBase `json:",inline"`
	Qps                       Qps              `json:"qps"`
	CacheHitRatio             CacheHitRatio    `json:"cachehitratio"`
	ResolvedRatios            []ResolvedRatio  `json:"resolvedratios"`
	QueryTypeRatios           []QueryTypeRatio `json:"querytyperatios"`
	TopTenIps                 []TopIp          `json:"toptenips"`
	TopTenDomains             []TopDomain      `json:"toptendomains"`
}

func (d Dns) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Node{}}
}

func (d Dns) GetActions() []restresource.Action {
	return exportActions
}
