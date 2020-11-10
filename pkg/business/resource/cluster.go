package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Cluster struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name" rest:"required=true,minLen=1,maxLen=20"`
	OperType              int32       `json:"operType" rest:"required=true,min=1,max=3"`
	Balance               Balance     `json:"Balance" rest:"required=true"`
	Application           Application `json:"application" rest:"required=true"`
	LogInfo               LogInfo     `json:"logInfo" rest:"required=true"`
	Cache                 Cache       `json:"cache" rest:"required=true"`
}
