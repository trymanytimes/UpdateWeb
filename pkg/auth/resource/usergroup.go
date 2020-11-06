package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type UserGroup struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string   `json:"name" rest:"required=true,minLen=1,maxLen=40" db:"uk"`
	Comment                   string   `json:"comment"`
	UserIds                   []string `json:"userIDs" db:"-"`
	RoleIds                   []string `json:"roleIDs"`
}
