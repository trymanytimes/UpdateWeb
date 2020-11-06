package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type ScannedSubnet struct {
	restresource.ResourceBase `json:",inline"`
	Ipnet                     string `json:"ipnet"`
	SubnetId                  uint32 `json:"-"`
	Tags                      string `json:"tags"`
	AssignedRatio             string `json:"assignedRatio"`
	UnassignedRatio           string `json:"unassignedRatio"`
	ReservationRatio          string `json:"reservationRatio"`
	StaticAddressRatio        string `json:"staticAddressRatio"`
	UnmanagedRatio            string `json:"unmanagedRatio"`
	ActiveRatio               string `json:"activeRatio"`
	InactiveRatio             string `json:"inactiveRatio"`
	ConflictRatio             string `json:"conflictRatio"`
	ZombieRatio               string `json:"zombieRatio"`
}

const ActionNameExportCSV = "exportcsv"

func (s ScannedSubnet) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameExportCSV,
			Input:  &ExportFilter{},
			Output: &FileInfo{},
		},
	}
}

type ExportFilter struct {
}

type FileInfo struct {
	Path string `json:"path"`
}

type ScannedSubnets []*ScannedSubnet

func (s ScannedSubnets) Len() int {
	return len(s)
}

func (s ScannedSubnets) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ScannedSubnets) Less(i, j int) bool {
	return s[i].SubnetId < s[j].SubnetId
}
