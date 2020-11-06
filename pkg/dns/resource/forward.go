package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Forward struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=50" db:"uk"`
	Ips                   []string `json:"ips" rest:"required=true"`
	Comment               string   `json:"comment"`
}

type ZoneForward struct {
	resource.ResourceBase `json:",inline"`
	ForwardZone           string `db:"ownby"`
	Forward               string `db:"referto"`
}
