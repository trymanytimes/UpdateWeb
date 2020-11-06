package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type StaticAddress struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string  `json:"-" db:"ownby"`
	HwAddress                 string  `json:"hwAddress" rest:"required=true" db:"uk"`
	IpAddress                 string  `json:"ipAddress" rest:"required=true" db:"uk"`
	Capacity                  uint64  `json:"capacity" rest:"description=readonly"`
	Version                   Version `json:"version" rest:"description=readonly"`
}

func (r StaticAddress) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}
