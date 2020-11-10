package resource

import "github.com/zdnscloud/gorest/resource"

type Website struct {
	resource.ResourceBase `json:",inline"`
	GroupID               string          `json:"groupID" rest:"required=true,minlen=1,maxlen=30"`
	SourceDomain          string          `json:"sourceDomain" rest:"required=true,minlen=1,maxlen=30"`
	DestDomain            string          `json:"destDomain" rest:"required=true,minlen=1,maxlen=30"`
	VirtualIP             string          `json:"virtualIP" rest:"required=true,minlen=1,maxlen=30"`
	ProtocolPorts         []*ProtocolPort `json:"protocolPorts" rest:"required=true"`
	HrefDomain            string          `json:"hrefDomain" rest:"required=true,minlen=1,maxlen=30"`
	TransferMod           int32           `json:"transferMod" rest:"min=1,max=5"`
}

type ProtocolPort struct {
	SourceProtocol string `json:"sourceProtocol" rest:"required=true,minlen=1,maxlen=30"`
	SourcePort     int32  `json:"sourcePort" rest:"required=true,min=1,max=65536"`
	DestProtocol   string `json:"destProtocol" rest:"required=true,minlen=1,maxlen=30"`
	DestPort       int32  `json:"destPort" rest:"required=true,min=1,max=65536"`
}
