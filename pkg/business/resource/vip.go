package resource

import "github.com/zdnscloud/gorest/resource"

type VipInterval struct {
	resource.ResourceBase `json:",inline"`
	BeginVip              string
	EndVip                string
	Length                int32
}

func (v VipInterval) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Balance{}}
}
