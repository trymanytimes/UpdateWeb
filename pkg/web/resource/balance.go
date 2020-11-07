package resource

type Balance struct {
	//resource.ResourceBase `json:",inline"`
	Name        string         `json:"name" rest:"required=true,minLen=1,maxLen=20" db:"uk"`
	ClusterType string         `json:"clusterType"`
	NodeHosts   []*NodeHost    `json:"nodeHosts"`
	Ipv6Vips    []*VipInterval `json:"ipv6Vips" rest:"required=true"`
}

type NodeHost struct {
	HostID string
	NodeID string
}

type VipInterval struct {
	BeginVip string
	EndVip   string
	Length   int32
}
