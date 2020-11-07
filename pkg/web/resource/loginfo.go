package resource

type LogInfo struct {
	//resource.ResourceBase `json:",inline"`
	IsOn          string `json:"isOn" rest:"required=true,min=1,max=3"`
	NodeLogSize   int32  `json:"nodeLogSize"`
	RemoteLogIp   string `json:"remoteLogIp"`
	RemoteLogPort int32  `json:"remoteLogPort"`
}
