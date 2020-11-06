package resource

const ResourceIDTopTenIPs = "toptenips"

type TopIp struct {
	Ip    string `json:"ip"`
	Count uint64 `json:"count"`
}
