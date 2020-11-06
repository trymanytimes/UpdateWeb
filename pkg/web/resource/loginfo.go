package resource
import (
	"github.com/zdnscloud/gorest/resource"
)

type Balance struct {
	//resource.ResourceBase `json:",inline"`
	IsOn                  string   `json:"isOn" rest:"required=true,min=1,max=3"`
	NodeLogSize                   string `json:"nodeLogSize"`
	RemoteLogIp                  string   `json:"remoteLogIp"`
	RemoteLogPort               int   `json:"remoteLogPort"`
}
