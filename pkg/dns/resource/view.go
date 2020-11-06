package resource

import (
	"encoding/base64"

	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/gorest/resource"
)

type View struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=20"`
	Priority              uint     `json:"priority" rest:"required=true,min=1,max=100"`
	Acls                  []string `json:"acls" rest:"required=true"`
	Dns64                 string   `json:"dns64" rest:"min=1,max=100"`
	Comment               string   `json:"comment"`
	LocalZoneSize         int      `json:"localzonesize"  db:"-"`
	NxdomainSize          int      `json:"nxdomainsize"  db:"-"`
	ForwardZoneSize       int      `json:"forwardzonesize"  db:"-"`
	MasterZoneSize        int      `json:"masterzonesize"  db:"-"`
	RRSize                int      `json:"rrsize"  db:"-"`
	UrlRedirectSize       int      `json:"urlRedirectSize" rest:"description=readonly"`
	Key                   string   `json:"-" db:"uk"`
}

func (v *View) GenerateKey() error {
	key, err := uuid.Gen()
	if err != nil {
		return err
	}
	v.Key = base64.StdEncoding.EncodeToString([]byte(key))
	return nil
}
