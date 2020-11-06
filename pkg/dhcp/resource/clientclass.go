package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type ClientClass struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"description=immutable" db:"uk"`
	Regexp                    string `json:"regexp"`
}

type ClientClasses []*ClientClass

func (c ClientClasses) Len() int {
	return len(c)
}

func (c ClientClasses) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ClientClasses) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}
