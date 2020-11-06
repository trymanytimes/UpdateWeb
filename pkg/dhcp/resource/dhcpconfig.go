package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type DhcpConfig struct {
	restresource.ResourceBase `json:",inline"`
	Identify                  string   `json:"-" db:"uk"`
	ValidLifetime             uint32   `json:"validLifetime"`
	MaxValidLifetime          uint32   `json:"maxValidLifetime"`
	MinValidLifetime          uint32   `json:"minValidLifetime"`
	DomainServers             []string `json:"domainServers"`
}
