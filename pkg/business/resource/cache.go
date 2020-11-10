package resource

type Cache struct {
	//resource.ResourceBase `json:",inline"`
	IsOn        int32 `json:"isOn" rest:"required=true,min=1,max=3"`
	CacheDBSize int32 `json:"cacheDBSize"`
}
