package resource
import (
	"github.com/zdnscloud/gorest/resource"
)

type Cache struct {
	//resource.ResourceBase `json:",inline"`
	IsOn                  int   `json:"isOn" rest:"required=true,min=1,max=3"`
	CacheDBSize                   int `json:"cacheDBSize"`
}
