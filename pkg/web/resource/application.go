package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Application struct {
	//resource.ResourceBase `json:",inline"`
	RaltRefererDefault                  int   `json:"raltReferDefault" rest:"required=true,min=1,max=3"`
	redirect                   string `json:"redirect"`
}