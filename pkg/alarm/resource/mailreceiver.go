package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type MailReceiver struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true" db:"uk"`
	Address                   string `json:"address" rest:"required=true"`
}

var TableMailReceiver = restdb.ResourceDBType(&MailReceiver{})
