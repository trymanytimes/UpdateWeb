package resource

import "github.com/zdnscloud/gorest/resource"

type WebGroup struct {
	resource.ResourceBase `json:",inline"`
	Name                  string            `json:"name" rest:"required=true,minlen=1,maxlen=30"`
	HrefDomain            string            `json:"hrefDomain" rest:"required=true,minlen=1,maxlen=30"`
	TransformMod          int32             `json:"transFormMod" rest:"required=true,min=1,max=3"`
	ClusterID             string            `json:"clusterid" rest:"required=true,minlen=1,maxlen=20"`
	UpdateSwithcher       *FuncSwitcherInfo `json:"updateSwitch" rest:"required=true"`
	Rules                 []*RuleInfo       `json:"rules"`
}

type FuncSwitcherInfo struct {
	IsReplaceHrefOn bool `json:"isReplaceHrefOn" rest:"required=true"`
	IsHttpsToHttpOn bool `json:"isHttpsToHttpOn" rest:"required=true"`
	IsCacheOn       bool `json:"isCacheOn" rest:"required=true"`
}

type RuleInfo struct {
	ID            string `json:"id" rest:"required=true"`
	RuleType      int32  `json:"ruleType" rest:"required=true"`
	SearchString  string `json:"searchString" rest:"required=true"`
	ReplaceString string `json:"replaceString" rest:"required=true"`
}

func (wg WebGroup) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
