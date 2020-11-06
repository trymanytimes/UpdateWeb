package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type IpBlackHole struct {
	resource.ResourceBase `json:",inline"`
	Acl                   string `json:"acl" rest:"required=true"`
}
