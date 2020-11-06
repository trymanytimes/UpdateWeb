package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Rr struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=256" db:"uk"`
	DataType              string `json:"datatype" rest:"required=true,options=A|AAAA|CNAME|HINFO|MX|NS|NAPTR|PTR|SRV|TXT" db:"uk"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=3000000"`
	Rdata                 string `json:"rdata" rest:"required=true" db:"uk"`
	RdataBackup           string `json:"rdataBackup"`
	ActiveRdata           string `json:"activeRdata" db:"-"`
	Zone                  string `json:"-" db:"uk" db:"-"`
	View                  string `json:"-" db:"ownby,uk"`
}

const (
	DataTypeCNAME string = "CNAME"
	DataTypeNS    string = "NS"
	DataTypeMX    string = "MX"
)

func (r Rr) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Zone{}}
}
