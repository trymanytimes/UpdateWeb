package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Acl struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=20" db:"uk"`
	Ips                   []string `json:"ips"`
	Isp                   string   `json:"isp" rest:"options=cmcc|cucc|ctcc|"`
	Status                string   `json:"status" rest:"required=true,options=forbidden|allow"`
	Comment               string   `json:"comment"`
}

type ViewAcl struct {
	resource.ResourceBase `json:",inline"`
	View                  string `db:"ownby"`
	Acl                   string `db:"referto"`
}
