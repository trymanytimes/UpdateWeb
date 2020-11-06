package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type RecursiveConcurrent struct {
	resource.ResourceBase `json:",inline"`
	RecursiveClients      int `json:"recursiveclients" rest:"required=true"`
	FetchesPerZone        int `json:"fetchesperzone" rest:"required=true"`
}
