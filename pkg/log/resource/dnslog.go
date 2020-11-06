package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type DNSLog struct {
	restresource.ResourceBase `json:",inline"`
	Time                      string `json:"time" rest:"description=readonly"`
	NodeIP                    string `json:"nodeIP" rest:"description=readonly"`
	Content                   string `json:"content" rest:"description=readonly"`
}

func (d DNSLog) GetActions() []restresource.Action {
	return exportActions
}

const ActionNameExportCSV = "exportcsv"

type ExportFilter struct {
	NodeIP string `json:"node_ip"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type FileInfo struct {
	Path string `json:"path"`
}

var exportActions = []restresource.Action{
	restresource.Action{
		Name:   ActionNameExportCSV,
		Input:  &ExportFilter{},
		Output: &FileInfo{},
	},
}
