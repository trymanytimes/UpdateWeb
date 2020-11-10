package resource

import "github.com/zdnscloud/gorest/resource"

type Host struct {
	resource.ResourceBase `json:",inline"`
	V4UpFlow              uint64 `json:"v4UpFlow" rest:"description=readonly"`
	V4DownFlow            uint64 `json:"v4DownFlow" rest:"description=readonly"`
	V6UpFlow              uint64 `json:"v6UpFlow" rest:"description=readonly"`
	V6DownFlow            uint64 `json:"v6DownFlow" rest:"description=readonly"`
	TimeStamp             uint64 `json:"timeStamp" rest:"description=readonly"`
}
