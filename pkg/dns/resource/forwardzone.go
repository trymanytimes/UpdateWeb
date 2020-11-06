package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type ForwardZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string     `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	ForwardIDs            []string   `json:"forwardids" db:"-" rest:"required=true"`
	Forwards              []*Forward `json:"forwards" db:"-"`
	ForwardType           string     `json:"forwardtype" rest:"required=true,options=only|first"`
	Comment               string     `json:"comment"`
	View                  string     `json:"-" db:"ownby,uk"`
}

func (z ForwardZone) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{View{}}
}
