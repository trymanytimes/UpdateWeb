package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type UrlRedirect struct {
	resource.ResourceBase `json:",inline"`
	Domain                string `json:"domain" rest:"required=true" db:"uk"`
	Url                   string `json:"url" rest:"required=true,minLen=1,maxLen=500"`
	View                  string `json:"-" db:"ownby"`
}

func (u UrlRedirect) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{View{}}
}
