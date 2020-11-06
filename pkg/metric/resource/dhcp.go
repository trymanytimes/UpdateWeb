package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Dhcp struct {
	restresource.ResourceBase `json:",inline"`
	Lps                       Lps              `json:"lps"`
	Lease                     Lease            `json:"lease"`
	Packets                   []Packet         `json:"packets"`
	SubnetUsedRatios          SubnetUsedRatios `json:"subnetusedratios"`
}

func (d Dhcp) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Node{}}
}

const ActionNameExportCSV = "exportcsv"

type ExportFilter struct {
	Period int `json:"period"`
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

func (dhcp Dhcp) GetActions() []restresource.Action {
	return exportActions
}
