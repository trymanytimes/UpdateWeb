package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type MailSender struct {
	restresource.ResourceBase `json:",inline"`
	Host                      string `json:"host" rest:"required=true"`
	Port                      int    `json:"port" rest:"required=true"`
	Username                  string `json:"username" rest:"required=true"`
	Password                  string `json:"password" rest:"required=true"`
	Enabled                   bool   `json:"enabled"`
}

var TableMailSender = restdb.ResourceDBType(&MailSender{})
