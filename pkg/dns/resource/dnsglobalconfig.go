package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type DnsGlobalConfig struct {
	restresource.ResourceBase `json:",inline"`
	LogEnable                 bool `json:"isLogOpen" rest:"required=true"`
	Ttl                       int  `json:"ttl" rest:"required=true,min=0,max=3000000"`
	DnssecEnable              bool `json:"isDnssecOpen" rest:"required=true"`
}

func (DnsGlobalConfig) CreateDefaultResource() restresource.Resource {
	return &DnsGlobalConfig{LogEnable: true, Ttl: 3600, DnssecEnable: false}
}
