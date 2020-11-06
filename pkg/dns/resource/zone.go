package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Zone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	IsArpa                bool   `json:"isarpa" rest:"required=true"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=3000000"`
	ZoneFile              string `json:"-"`
	RRSize                int    `json:"rrsize" db:"-"`
	RrsRole               string `json:"rrsRole"`
	Comment               string `json:"comment"`
	View                  string `json:"-" db:"ownby,uk"`
}

func (z Zone) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{View{}}
}

func (z Zone) GetActions() []resource.Action {
	return rrRoleAction
}
func (z *Zone) SetZoneFile(zoneFileSuffix string) {
	z.ZoneFile = z.GetParent().GetID() + "#" + z.Name + zoneFileSuffix
}

const ChangingRRs = "changingRRs"

var rrRoleAction = []resource.Action{
	{
		Name:   ChangingRRs,
		Input:  &EnableRole{},
		Output: &OperResult{},
	},
}

type EnableRole struct {
	Role string `json:"role" rest:"options=main|backup"`
}

type OperResult struct {
	Result bool `json:"result"`
}
