package resource

import (
	"net/http"

	restresource "github.com/zdnscloud/gorest/resource"
)

type Role struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string                   `json:"name" rest:"required=true,minLen=1,maxLen=20" db:"uk"`
	Comment                   string                   `json:"comment"`
	Views                     []string                 `json:"views"`
	Plans                     []string                 `json:"plans"`
	RoleAuthority             map[string]RoleAuthority `json:"-" db:"-"`
}

type OperationsType string
type RoleType string

const (
	OperationsTypeGET    OperationsType = http.MethodGet
	OperationsTypePUT    OperationsType = http.MethodPut
	OperationsTypePOST   OperationsType = http.MethodPost
	OperationsTypeDELETE OperationsType = http.MethodDelete
	OperationsTypeACTION OperationsType = "ACTION"
)

const (
	RoleTypeSUPER  RoleType = "SUPER"
	RoleTypeNORMAL RoleType = "NORMAL"
)

type RoleAuthority struct {
	Resource   string           `json:"resource"`
	Views      []string         `json:"views"`
	Plans      []string         `json:"plans"`
	Filter     bool             `json:"filter"`
	Operations []OperationsType `json:"operations"`
}

type RoleAuthorityClass struct {
	BaseAuthority []RoleAuthority `json:"baseAuthority"`
	DnsAuthority  []RoleAuthority `json:"dnsAuthority"`
	DhcpAuthority []RoleAuthority `json:"dhcpAuthority"`
}

type RoleTemplate struct {
	Role RoleTemplateType `json:"role"`
}

type RoleTemplateType struct {
	Super  RoleAuthorityClass `json:"SUPER"`
	Normal RoleAuthorityClass `json:"NORMAL"`
}
