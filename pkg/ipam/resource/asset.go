package resource

import (
	"fmt"
	"net"
	"strconv"

	"github.com/zdnscloud/gorest/resource"
)

const TelephoneLen = 11

type DeviceType string

const (
	DeviceTypePC      DeviceType = "pc"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypePrinter DeviceType = "printer"
	DeviceTypeCamera  DeviceType = "camera"
	DeviceTypeNVR     DeviceType = "nvr"
	DeviceTypeIOT     DeviceType = "iot"
	DeviceTypeOther   DeviceType = "other"
)

type Asset struct {
	resource.ResourceBase `json:",inline"`

	Mac               string     `json:"mac" db:"uk" rest:"required=true"`
	Ipv4s             []string   `json:"ipv4s"`
	Ipv6s             []string   `json:"ipv6s"`
	Name              string     `json:"name" rest:"required=true"`
	DeviceType        DeviceType `json:"deviceType" rest:"required=true,options=pc|mobile|printer|camera|nvr|iot|other"`
	DeployedService   string     `json:"deployedService"`
	ComputerRoom      string     `json:"computerRoom"`
	ComputerRack      string     `json:"computerRack"`
	SwitchName        string     `json:"switchName"`
	SwitchPort        string     `json:"switchPort"`
	VlanId            int        `json:"vlanId"`
	Department        string     `json:"department"`
	ResponsiblePerson string     `json:"responsiblePerson"`
	Telephone         string     `json:"telephone"`
}

const ActionNameRegister = "register"

func (a Asset) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:  ActionNameRegister,
			Input: &AssetRegister{},
		},
	}
}

type AssetRegister struct {
	ComputerRoom string `json:"computerRoom"`
	ComputerRack string `json:"computerRack"`
	SwitchName   string `json:"switchName"`
	SwitchPort   string `json:"switchPort"`
	SubnetId     string `json:"subnetId"`
	Ip           string `json:"ip"`
	VlanId       int    `json:"vlanId"`
	Isv4         bool   `json:"-"`
}

func (a *AssetRegister) ToAsset(id string) *Asset {
	asset := &Asset{
		ComputerRoom: a.ComputerRoom,
		ComputerRack: a.ComputerRack,
		SwitchName:   a.SwitchName,
		SwitchPort:   a.SwitchPort,
		VlanId:       a.VlanId,
	}
	if a.Isv4 {
		asset.Ipv4s = []string{a.Ip}
	} else {
		asset.Ipv6s = []string{a.Ip}
	}

	asset.SetID(id)
	return asset
}

func (a *Asset) Validate() error {
	if _, err := net.ParseMAC(a.Mac); err != nil {
		return fmt.Errorf("mac %s isn't valid", a.Mac)
	}

	if err := checkIpsValid(a.Ipv4s, true); err != nil {
		return err
	}

	if err := checkIpsValid(a.Ipv6s, false); err != nil {
		return err
	}

	if err := checkVlanIdValid(a.VlanId); err != nil {
		return err
	}

	return checkTelephoneValid(a.Telephone)
}

func checkIpsValid(ips []string, isV4 bool) error {
	ipset := make(map[string]struct{})
	for _, ip := range ips {
		if checkIpValid(ip, isV4) == false {
			return fmt.Errorf("invalid ip %s", ip)
		}

		if _, ok := ipset[ip]; ok {
			return fmt.Errorf("duplicate ip %s", ip)
		} else {
			ipset[ip] = struct{}{}
		}
	}

	return nil
}

func checkIpValid(ipstr string, isV4 bool) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}

	if isV4 {
		return ip.To4() != nil
	} else {
		return ip.To4() == nil
	}
}

func checkVlanIdValid(id int) error {
	if id != 0 && (id < 1 || id > 4094) {
		return fmt.Errorf("invalid vlan %d, it must be in [1, 4094]", id)
	}

	return nil
}

func checkTelephoneValid(telephone string) error {
	if telephone == "" {
		return nil
	}

	if len(telephone) != TelephoneLen {
		return fmt.Errorf("telephone should be 11 number")
	}
	_, err := strconv.ParseUint(telephone, 10, 64)
	if err != nil {
		return fmt.Errorf("telephone isn't number:%s", err.Error())
	}

	return nil
}

func (a *AssetRegister) Validate() error {
	if a.SubnetId == "" {
		return fmt.Errorf("subnet id is missing")
	}

	if ip := net.ParseIP(a.Ip); ip == nil {
		return fmt.Errorf("invalid ip %s", a.Ip)
	} else {
		a.Isv4 = ip.To4() != nil
	}

	return checkVlanIdValid(a.VlanId)
}
