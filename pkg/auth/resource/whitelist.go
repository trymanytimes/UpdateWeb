package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type WhiteList struct {
	restresource.ResourceBase `json:",inline"`
	Ips                       []string `json:"ips"`
	Privilege                 string   `json:"-"`
	Enabled                   bool     `json:"isEnable" rest:"required=true"`
}
