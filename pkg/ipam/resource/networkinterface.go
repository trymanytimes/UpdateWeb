package resource

import (
	"net"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restresource "github.com/zdnscloud/gorest/resource"
)

type IPState string

const (
	IPStateActive   IPState = "active"
	IPStateInactive IPState = "inactive"
	IPStateConflict IPState = "conflict"
	IPStateZombie   IPState = "zombie"
)

func (s IPState) IsEmpty() bool {
	return string(s) == ""
}

type IPType string

const (
	IPTypeAssigned    IPType = "assigned"
	IPTypeUnassigned  IPType = "unassigned"
	IPTypeUnmanagered IPType = "unmanagered"
	IPTypeReservation IPType = "reservation"
	IPTypeStatic      IPType = "static"
)

type NetworkInterface struct {
	restresource.ResourceBase `json:",inline"`
	Ip                        string               `json:"ip"`
	Mac                       string               `json:"mac"`
	Hostname                  string               `json:"hostname"`
	ValidLifetime             uint32               `json:"validLifetime"`
	Expire                    restresource.ISOTime `json:"expire"`
	IpType                    IPType               `json:"ipType" rest:"options=assigned|unassigned|unmanagered|reservation"`
	IpState                   IPState              `json:"ipState" rest:"options=active|inactive|conflict|zombie"`
	ComputerRoom              string               `json:"computerRoom"`
	ComputerRack              string               `json:"computerRack"`
	SwitchName                string               `json:"switchName"`
	SwitchPortName            string               `json:"switchPort"`
	VlanId                    int                  `json:"vlanId"`
}

func (n NetworkInterface) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{ScannedSubnet{}}
}

type NetworkInterfaces []*NetworkInterface

func (n NetworkInterfaces) Len() int {
	return len(n)
}

func (n NetworkInterfaces) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n NetworkInterfaces) Less(i, j int) bool {
	if niIp := net.ParseIP(n[i].Ip); niIp.To4() != nil {
		return util.Ipv4ToUint32(niIp) < util.Ipv4ToUint32(net.ParseIP(n[j].Ip))
	} else {
		return util.Ipv6ToBigInt(niIp).Cmp(util.Ipv6ToBigInt(net.ParseIP(n[j].Ip))) == -1
	}
}
