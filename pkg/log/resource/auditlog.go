package resource

import (
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type AuditLog struct {
	restresource.ResourceBase `json:",inline"`
	Username                  string    `json:"username" rest:"description=readonly"`
	SourceIp                  string    `json:"sourceIp" rest:"description=readonly"`
	Method                    string    `json:"method" rest:"description=readonly"`
	ResourceKind              string    `json:"resourceKind" rest:"description=readonly"`
	ResourcePath              string    `json:"resourcePath" rest:"description=readonly"`
	ResourceId                string    `json:"resourceId" rest:"description=readonly"`
	Parameters                string    `json:"parameters" rest:"description=readonly"`
	Succeed                   bool      `json:"succeed" rest:"description=readonly"`
	ErrMessage                string    `json:"errMessage" rest:"description=readonly"`
	Expire                    time.Time `json:"-"`
}

var TableAuditLog = restdb.ResourceDBType(&AuditLog{})
