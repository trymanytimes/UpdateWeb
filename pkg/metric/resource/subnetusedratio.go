package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

const ResourceIDSubnetUsedRatios = "subnetusedratios"

type SubnetUsedRatio struct {
	Ipnet      string               `json:"ipnet"`
	UsedRatios []RatioWithTimestamp `json:"usedRatios"`
}

type RatioWithTimestamp struct {
	Timestamp restresource.ISOTime `json:"timestamp"`
	Ratio     string               `json:"ratio"`
}

type SubnetUsedRatios []SubnetUsedRatio

func (s SubnetUsedRatios) Len() int {
	return len(s)
}

func (s SubnetUsedRatios) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SubnetUsedRatios) Less(i, j int) bool {
	siUsedRatio := s[i].getFirstUsedRatio()
	sjUsedRatio := s[j].getFirstUsedRatio()
	if siUsedRatio == sjUsedRatio {
		return s[i].Ipnet < s[j].Ipnet
	} else {
		return siUsedRatio < sjUsedRatio
	}
}

func (s SubnetUsedRatio) getFirstUsedRatio() string {
	for _, u := range s.UsedRatios {
		return u.Ratio
	}

	return ""
}
