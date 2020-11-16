package resource

import "github.com/zdnscloud/gorest/resource"

type Node struct {
	resource.ResourceBase `json:",inline"`
	HostID                string `json:"HostID"`
}
