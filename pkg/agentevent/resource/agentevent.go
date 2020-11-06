package resource

import "github.com/zdnscloud/gorest/resource"

type AgentEvent struct {
	resource.ResourceBase `json:",inline"`
	Node                  string `json:"node"`
	NodeType              string `json:"nodeType"`
	Resource              string `json:"resource"`
	Method                string `json:"method"`
	Succeed               bool   `json:"succeed"`
	ErrorMessage          string `json:"errorMessage"`
	CmdMessage            string `json:"cmdMessage"`
	OperationTime         string `json:"operationTime"`
}
