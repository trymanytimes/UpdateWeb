package resource

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/soniah/gosnmp"
	"github.com/zdnscloud/cement/set"
	restresource "github.com/zdnscloud/gorest/resource"

	ipamutil "github.com/linkingthing/ddi-controller/pkg/ipam/util"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	AddrPrefixOidZeroDotZero = ".0.0"
	OIDIpAddressPrefixOrigin = ".1.3.6.1.2.1.4.32.1.5"

	OIDIpv4AddressPrefix          = ".1.3.6.1.2.1.4.34.1.5.1.4"
	OIDIpv6AddressPrefix          = ".1.3.6.1.2.1.4.34.1.5.2.16"
	OIDIpNetToPhysicalPhysAddress = ".1.3.6.1.2.1.4.35.1.4"

	OIDIpv4AdEntNetMask          = ".1.3.6.1.2.1.4.20.1.3"
	OIDIpv4NetToMediaPhysAddress = ".1.3.6.1.2.1.4.22.1.2"

	OIDIpv6AddrPfxLength         = ".1.3.6.1.2.1.55.1.8.1.2"
	OIDIpv6NetToMediaPhysAddress = ".1.3.6.1.2.1.55.1.12.1.2"

	OIDDot1dTpFdbPort       = ".1.3.6.1.2.1.17.4.3.1.2"
	OIDDot1dBasePortIfIndex = ".1.3.6.1.2.1.17.1.4.1.2"
	OIDIfName               = ".1.3.6.1.2.1.31.1.1.1.1"
	OIDDot1qPvid            = ".1.3.6.1.2.1.17.7.1.4.5.1.1"

	OIDIpRouteIfIndex = ".1.3.6.1.2.1.4.21.1.2"
	OIDIpRouteNextHop = ".1.3.6.1.2.1.4.21.1.7"
	OIDIpRouteType    = ".1.3.6.1.2.1.4.21.1.8"

	OIDIpCidrRouteNextHop = ".1.3.6.1.2.1.4.24.4.1.4"
	OIDIpCidrRouteIfIndex = ".1.3.6.1.2.1.4.24.4.1.5"
	OIDIpCidrRouteType    = ".1.3.6.1.2.1.4.24.4.1.6"

	OIDIpAdEntIfIndex = ".1.3.6.1.2.1.4.20.1.2"
	OIDIfPhysAddress  = ".1.3.6.1.2.1.2.2.1.6"

	SNMPConnTimeout           = 3 * time.Second
	SNMPConnRetryCount        = 3
	SNMPGetMaxOIDS            = 60
	SNMPPort           uint16 = 161
)

var AllZeroMAC = []byte{0, 0, 0, 0, 0, 0}

type Subnet struct {
	IPNet net.IPNet
	NICs  map[string]*NIC
}

type NIC struct {
	IP             string
	Mac            string
	ComputerRoom   string
	ComputerRack   string
	SwitchName     string
	SwitchPortName string
	VlanId         int
}

type FDB struct {
	Mac        string
	BridgePort int
	IfName     string
}

type Route struct {
	AddrMaskNextHop string
	NextHop         string
	IfName          string
}

type LinkedNetworkEquipment struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

type EquipmentPortType string

const (
	EquipmentPortTypeUplink   EquipmentPortType = "uplink"
	EquipmentPortTypeDownlink EquipmentPortType = "downlink"
	EquipmentPortTypeAccess   EquipmentPortType = "access"
	EquipmentPortTypeNextHop  EquipmentPortType = "nexthop"
)

type EquipmentPort struct {
	Mac         string
	Ip          string
	Type        EquipmentPortType
	LearnedMacs set.StringSet
}

type EquipmentType string

const (
	EquipmentTypeRouter            EquipmentType = "router"
	EquipmentTypeSecurityGateWay   EquipmentType = "security_gateway"
	EquipmentTypeCoreSwitch        EquipmentType = "core_switch"
	EquipmentTypeAccessSwitch      EquipmentType = "access_switch"
	EquipmentTypeConvergenceSwitch EquipmentType = "convergence_switch"
	EquipmentTypeFirewall          EquipmentType = "firewall"
	EquipmentTypeWirelessAp        EquipmentType = "wirelessAp"
	EquipmentTypeWirelessAc        EquipmentType = "wirelessAc"
	EquipmentTypeOther             EquipmentType = "other"
)

type NetworkEquipment struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string                            `json:"name" rest:"required=true" db:"uk"`
	AdministrationAddress     string                            `json:"administrationAddress" rest:"required=true"`
	EquipmentType             EquipmentType                     `json:"equipmentType" rest:"required=true,options=router|security_gateway|core_switch|access_switch|convergence_switch|firewall|wirelessAp|wirelessAc|other"`
	Manufacturer              string                            `json:"manufacturer"`
	SerialNumber              string                            `json:"serialNumber"`
	FirmwareVersion           string                            `json:"firmwareVersion"`
	UplinkAddresses           map[string]LinkedNetworkEquipment `json:"uplinkAddresses" db:"-"`
	DownlinkAddresses         map[string]LinkedNetworkEquipment `json:"downlinkAddresses" db:"-"`
	NextHopAddresses          map[string]LinkedNetworkEquipment `json:"nextHopAddresses" db:"-"`
	ComputerRoom              string                            `json:"computerRoom"`
	ComputerRack              string                            `json:"computerRack"`
	Location                  string                            `json:"location"`
	Department                string                            `json:"department"`
	ResponsiblePerson         string                            `json:"responsiblePerson"`
	Telephone                 string                            `json:"telephone"`
	SnmpPort                  uint16                            `json:"snmpPort"`
	SnmpCommunity             string                            `json:"snmpCommunity"`
	LastRefreshTime           string                            `json:"lastRefreshTime"`
	Ports                     map[string]EquipmentPort          `json:"-" db:"-"`
	IfNames                   map[int]string                    `json:"-" db:"-"`
	IfIndexs                  map[int]int                       `json:"-" db:"-"`
	VlanIds                   map[string]int                    `json:"-" db:"-"`
	AdministrationMac         string                            `json:"administrationMac"`
}

const ActionNameSNMP = "snmp"

func (n NetworkEquipment) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:  ActionNameSNMP,
			Input: &SnmpConfig{},
		},
	}
}

type SnmpConfig struct {
	Port      uint16 `json:"port"`
	Community string `json:"community"`
}

func (n *NetworkEquipment) Validate() error {
	if net.ParseIP(n.AdministrationAddress) == nil {
		return fmt.Errorf("invalid administration address %s", n.AdministrationAddress)
	}

	if err := checkAddresses(n.UplinkAddresses); err != nil {
		return fmt.Errorf("invalid uplink address %s", err.Error())
	}

	if err := checkAddresses(n.DownlinkAddresses); err != nil {
		return fmt.Errorf("invalid downlink address %s", err.Error())
	}

	if err := checkAddresses(n.NextHopAddresses); err != nil {
		return fmt.Errorf("invalid nexthop address %s", err.Error())
	}

	return checkTelephoneValid(n.Telephone)
}

func checkAddresses(addrs map[string]LinkedNetworkEquipment) error {
	for _, addr := range addrs {
		if net.ParseIP(addr.Ip) == nil {
			return fmt.Errorf("invalid address %s", addr)
		}
	}

	return nil
}

func (n *NetworkEquipment) RefreshSubnet() ([]*Subnet, error) {
	if n.SnmpCommunity == "" {
		return nil, nil
	}

	n.LastRefreshTime = time.Now().Format(util.TimeFormat)
	var subnets []*Subnet
	subnet4s, err := n.refreshV4Subnet()
	if err != nil {
		return nil, err
	} else {
		subnets = append(subnets, subnet4s...)
	}

	subnet6s, err := n.refreshV6Subnet()
	if err != nil {
		return nil, err
	} else {
		subnets = append(subnets, subnet6s...)
	}

	if err := n.refreshV4AndV6IPAndMacs(subnets); err != nil {
		return nil, err
	}

	if err := n.refreshIfIndexs(); err != nil {
		return nil, err
	}

	if err := n.refreshIfNames(); err != nil {
		return nil, err
	}

	if err := n.refreshVlanIds(); err != nil {
		return nil, err
	}

	if err := n.refreshPorts(); err != nil {
		return nil, err
	}

	if err := n.refreshRouteInfo(); err != nil {
		return nil, err
	}

	return subnets, nil
}

func (n *NetworkEquipment) refreshV4Subnet() ([]*Subnet, error) {
	ipnets, err := n.getV4DirectConnectSubnet()
	if err != nil {
		return nil, err
	}

	subnets := make([]*Subnet, len(ipnets))
	for i, ipnet := range ipnets {
		subnets[i] = &Subnet{
			IPNet: ipnet,
			NICs:  make(map[string]*NIC),
		}
	}

	if err := n.refreshV4IPAndMacs(subnets); err != nil {
		return nil, err
	}

	return subnets, nil
}

func (n *NetworkEquipment) getV4DirectConnectSubnet() ([]net.IPNet, error) {
	ipnets, err := n.getSubnetByOIDIpv4AdEntNetMask()
	if err != nil {
		return nil, err
	}

	if len(ipnets) == 0 {
		return n.getSubnetByOIDIpv4AddressPrefix()
	} else {
		return ipnets, nil
	}
}

func (n *NetworkEquipment) getSubnetByOIDIpv4AddressPrefix() ([]net.IPNet, error) {
	var subnets []net.IPNet
	if err := n.walkOID(OIDIpv4AddressPrefix, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.ObjectIdentifier {
			return fmt.Errorf("the value of v4 address prefix should be object identifier but get %v", pdu.Type)
		}

		prefixOid := pdu.Value.(string)
		if prefixOid == AddrPrefixOidZeroDotZero {
			return nil
		}

		prefixs := strings.Split(strings.TrimPrefix(prefixOid, OIDIpAddressPrefixOrigin), ".")
		if len(prefixs) != 9 {
			return fmt.Errorf("invalid v4 address prefix oid: %v", pdu.Value)
		}

		prefixLen, err := strconv.Atoi(prefixs[8])
		if err != nil {
			return fmt.Errorf("parse address prefix len %s failed: %s", prefixs[8], err.Error())
		}

		if prefixLen > 0 && prefixLen < 32 {
			ipnet, err := getIPNet4FromPduName(pdu.Name, OIDIpv4AddressPrefix, net.CIDRMask(prefixLen, 32))
			if err != nil {
				return err
			}

			if ipnet == nil {
				return nil
			}

			subnets = append(subnets, *ipnet)
		}

		return nil
	}); err != nil {
		return nil, err
	} else {
		return subnets, nil
	}
}

func getIPNet4FromPduName(pduName, oid string, mask net.IPMask) (*net.IPNet, error) {
	ip, err := parseIPFromPduName(pduName, oid)
	if err != nil {
		return nil, err
	}

	if ip.IsLoopback() {
		return nil, nil
	}

	return &net.IPNet{
		IP:   ip.Mask(mask),
		Mask: mask,
	}, nil
}

func parseIPFromPduName(pduName, oid string) (net.IP, error) {
	ipstr := strings.TrimPrefix(pduName, oid)[1:]
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, fmt.Errorf("invalid ipv4 %s", ipstr)
	}

	return ip, nil
}

func (n *NetworkEquipment) getSubnetByOIDIpv4AdEntNetMask() ([]net.IPNet, error) {
	var subnets []net.IPNet
	if err := n.walkOID(OIDIpv4AdEntNetMask, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.IPAddress {
			return fmt.Errorf("the value of v4 address entry net mask should be ip address but get %v", pdu.Type)
		}

		mask, err := ipamutil.ParseIPv4Mask(pdu.Value.(string))
		if err != nil {
			return err
		}

		if ones, _ := mask.Size(); ones > 0 && ones < 32 {
			ipnet, err := getIPNet4FromPduName(pdu.Name, OIDIpv4AdEntNetMask, mask)
			if err != nil {
				return err
			}

			if ipnet == nil {
				return nil
			}

			subnets = append(subnets, *ipnet)
		}

		return nil
	}); err != nil {
		return nil, err
	} else {
		return subnets, nil
	}
}

func (n *NetworkEquipment) refreshV4IPAndMacs(subnets []*Subnet) error {
	return n.walkOID(OIDIpv4NetToMediaPhysAddress, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.OctetString {
			return nil
		}

		macBytes := pdu.Value.([]byte)
		if bytes.Equal(macBytes, AllZeroMAC) {
			return nil
		}

		ips := strings.Split(strings.TrimPrefix(pdu.Name, OIDIpv4NetToMediaPhysAddress), ".")
		if len(ips) != 6 {
			return fmt.Errorf("invalid arp entry name %s", pdu.Name)
		}

		ip, err := ipamutil.ParseIPv4Address(ips[2:])
		if err != nil {
			return fmt.Errorf("arp entry name %s has invalid ip address", pdu.Name)
		}

		for _, subnet := range subnets {
			if subnet.IPNet.Contains(ip) {
				subnet.NICs[ip.String()] = &NIC{
					IP:  ip.String(),
					Mac: net.HardwareAddr(macBytes).String(),
				}
				break
			}
		}
		return nil
	})
}

func (n *NetworkEquipment) refreshV4AndV6IPAndMacs(subnets []*Subnet) error {
	return n.walkOID(OIDIpNetToPhysicalPhysAddress, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.OctetString {
			return nil
		}

		macBytes := pdu.Value.([]byte)
		if bytes.Equal(macBytes, AllZeroMAC) {
			return nil
		}

		ips := strings.Split(strings.TrimPrefix(pdu.Name, OIDIpNetToPhysicalPhysAddress), ".")
		if len(ips) == 24 {
			return nil
		}

		if len(ips) != 8 && len(ips) != 20 {
			return fmt.Errorf("invalid arp entry name %s[%d]", pdu.Name, len(ips))
		}

		var ip net.IP
		var err error
		if ips[3] == "4" {
			ip, err = ipamutil.ParseIPv4Address(ips[4:])
		} else if ips[3] == "16" {
			ip, err = ipamutil.ParseIPv6Address(ips[4:])
		} else {
			return fmt.Errorf("arp entry name %s has invalid ip address version %s", pdu.Name, ips[3])
		}

		if err != nil {
			return fmt.Errorf("arp entry name %s has invalid ip address", pdu.Name)
		}

		for _, subnet := range subnets {
			if subnet.IPNet.Contains(ip) {
				subnet.NICs[ip.String()] = &NIC{
					IP:  ip.String(),
					Mac: net.HardwareAddr(macBytes).String(),
				}
				break
			}
		}
		return nil
	})
}

func (n *NetworkEquipment) refreshIfIndexs() error {
	n.IfIndexs = make(map[int]int)
	return n.walkOID(OIDDot1dBasePortIfIndex, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return nil
		}

		portStr := strings.TrimPrefix(pdu.Name, OIDDot1dBasePortIfIndex)
		port, err := strconv.Atoi(portStr[1:])
		if err != nil {
			return fmt.Errorf("parse bridge port with pdu name %s failed: %s", pdu.Name, err.Error())
		}

		n.IfIndexs[port] = pdu.Value.(int)
		return nil
	})
}

func (n *NetworkEquipment) refreshIfNames() error {
	n.IfNames = make(map[int]string)
	return n.walkOID(OIDIfName, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.OctetString {
			return nil
		}

		ifIndexStr := strings.TrimPrefix(pdu.Name, OIDIfName)
		ifIndex, err := strconv.Atoi(ifIndexStr[1:])
		if err != nil {
			return fmt.Errorf("parse ifindex with pdu name %s failed: %s", pdu.Name, err.Error())
		}

		n.IfNames[ifIndex] = string(pdu.Value.([]byte))
		return nil
	})
}

func (n *NetworkEquipment) refreshVlanIds() error {
	n.VlanIds = make(map[string]int)
	return n.walkOID(OIDDot1qPvid, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Gauge32 {
			return nil
		}

		portStr := strings.TrimPrefix(pdu.Name, OIDDot1qPvid)
		port, err := strconv.Atoi(portStr[1:])
		if err != nil {
			return fmt.Errorf("parse bridge port with pdu name %s failed: %s", pdu.Name, err.Error())
		}

		if ifName, ok := n.IfNames[n.IfIndexs[port]]; ok {
			n.VlanIds[ifName] = int(pdu.Value.(uint))
		}
		return nil
	})
}

func (n *NetworkEquipment) refreshPorts() error {
	n.Ports = make(map[string]EquipmentPort)
	if err := n.refreshIfPhysAddresses(); err != nil {
		return err
	}

	if err := n.refreshSwitchPorts(); err != nil {
		return err
	}

	return n.refreshRoutePorts()
}

func (n *NetworkEquipment) refreshIfPhysAddresses() error {
	return n.walkOID(OIDIfPhysAddress, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.OctetString {
			return nil
		}

		macBytes := pdu.Value.([]byte)
		if bytes.Equal(macBytes, AllZeroMAC) {
			return nil
		}

		ifIndex, err := strconv.Atoi(strings.TrimPrefix(pdu.Name, OIDIfPhysAddress)[1:])
		if err != nil {
			return fmt.Errorf("parse ifindex with pdu name %s failed: %s", pdu.Name, err.Error())
		}

		if ifName, ok := n.IfNames[ifIndex]; ok {
			n.Ports[ifName] = EquipmentPort{
				Mac: net.HardwareAddr(macBytes).String(),
			}
		}

		return nil
	})
}

func (n *NetworkEquipment) refreshSwitchPorts() error {
	if n.IsSwitchEquipment() == false {
		return nil
	}

	fdbs, err := n.getMacForwardingDB()
	if err != nil {
		return err
	}

	n.refreshSwitchWithFDB(fdbs)
	return nil
}

func (n *NetworkEquipment) IsSwitchEquipment() bool {
	return n.EquipmentType == EquipmentTypeCoreSwitch || n.EquipmentType == EquipmentTypeAccessSwitch || n.EquipmentType == EquipmentTypeConvergenceSwitch
}

func (n *NetworkEquipment) getMacForwardingDB() ([]*FDB, error) {
	var fdbs []*FDB
	if err := n.walkOID(OIDDot1dTpFdbPort, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return nil
		}

		macStrs := strings.Split(strings.TrimPrefix(pdu.Name, OIDDot1dTpFdbPort), ".")
		if len(macStrs) != 7 {
			return fmt.Errorf("invalid mac port entry name %s", pdu.Name)
		}

		macBytes, err := ipamutil.StringArrayToBytes(macStrs[1:], 6)
		if err != nil {
			return fmt.Errorf("parse mac from pdu name %s failed: %s", pdu.Name, err.Error())
		}

		fdbs = append(fdbs, &FDB{
			Mac:    net.HardwareAddr(macBytes).String(),
			IfName: n.IfNames[n.IfIndexs[pdu.Value.(int)]],
		})

		return nil
	}); err != nil {
		return nil, err
	} else {
		return fdbs, nil
	}
}

func (n *NetworkEquipment) refreshSwitchWithFDB(fdbs []*FDB) {
	for _, fdb := range fdbs {
		port := n.Ports[fdb.IfName]
		if port.LearnedMacs == nil {
			port = EquipmentPort{LearnedMacs: set.NewStringSet()}
		}

		port.LearnedMacs.Add(fdb.Mac)
		n.Ports[fdb.IfName] = port
	}
}

func (n *NetworkEquipment) refreshRoutePorts() error {
	if n.IsRouteEquipment() == false {
		return nil
	}

	return n.walkOID(OIDIpAdEntIfIndex, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return nil
		}

		ip, err := parseIpFromPduNameAndOid(pdu.Name, OIDIpAdEntIfIndex)
		if err != nil {
			return fmt.Errorf("parse ifindex with pdu name %s failed: %s", pdu.Name, err.Error())
		}

		if ifName, ok := n.IfNames[pdu.Value.(int)]; ok {
			port := n.Ports[ifName]
			port.Ip = ip.String()
			n.Ports[ifName] = port
		}
		return nil
	})
}

func (n *NetworkEquipment) IsRouteEquipment() bool {
	return n.EquipmentType == EquipmentTypeRouter || n.EquipmentType == EquipmentTypeFirewall
}

func parseIpFromPduNameAndOid(pduName, oid string) (net.IP, error) {
	return parseIpFromPduNameSuffix(strings.TrimPrefix(pduName, oid))
}

func parseIpFromPduNameSuffix(pduName string) (net.IP, error) {
	ips := strings.Split(pduName, ".")
	if len(ips) != 5 && len(ips) != 14 {
		return nil, fmt.Errorf("invalid pdu name %s for route type", pduName)
	}

	return ipamutil.ParseIPv4Address(ips[1:5])
}

func (n *NetworkEquipment) refreshV6Subnet() ([]*Subnet, error) {
	ipnets, err := n.getV6DirectConnectSubnet()
	if err != nil {
		return nil, err
	}

	subnets := make([]*Subnet, len(ipnets))
	for i, ipnet := range ipnets {
		subnets[i] = &Subnet{
			IPNet: ipnet,
			NICs:  make(map[string]*NIC),
		}
	}

	if err := n.refreshV6IPAndMacs(subnets); err != nil {
		return nil, err
	}

	return subnets, nil
}

func (n *NetworkEquipment) getV6DirectConnectSubnet() ([]net.IPNet, error) {
	ipnets, err := n.getSubnetByOIDIpv6AddrPfxLength()
	if err != nil {
		return nil, err
	}

	if len(ipnets) == 0 {
		return n.getSubnetByOIDIpv6AddressPrefix()
	} else {
		return ipnets, nil
	}
}

func (n *NetworkEquipment) getSubnetByOIDIpv6AddressPrefix() ([]net.IPNet, error) {
	var subnets []net.IPNet
	if err := n.walkOID(OIDIpv6AddressPrefix, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.ObjectIdentifier {
			return fmt.Errorf("the value of v6 address prefix should be object identifier but get %v", pdu.Type)
		}

		prefixOid := pdu.Value.(string)
		if prefixOid == AddrPrefixOidZeroDotZero {
			return nil
		}

		prefixs := strings.Split(strings.TrimPrefix(prefixOid, OIDIpAddressPrefixOrigin), ".")
		if len(prefixs) != 5 && len(prefixs) != 21 {
			return fmt.Errorf("invalid v6 address prefix oid: %v", pdu.Value)
		}

		prefixLen, err := strconv.Atoi(prefixs[len(prefixs)-1])
		if err != nil {
			return fmt.Errorf("parse address prefix len %s failed: %s", prefixs[len(prefixs)-1], err.Error())
		}

		ipnet, err := getIpNet6FromPduName(pdu.Name, OIDIpv6AddressPrefix, prefixLen)
		if err != nil {
			return err
		}

		if ipnet != nil {
			subnets = append(subnets, *ipnet)
		}

		return nil
	}); err != nil {
		return nil, err
	} else {
		return subnets, nil
	}
}

func getIpNet6FromPduName(pduName, oid string, prefixLen int) (*net.IPNet, error) {
	if prefixLen <= 0 || prefixLen >= 128 {
		return nil, nil
	}

	ips := strings.Split(strings.TrimPrefix(pduName, oid), ".")
	if len(ips) == 23 {
		return nil, nil
	}

	if len(ips) != 17 && len(ips) != 18 {
		return nil, fmt.Errorf("prefix length key isn't valid %s", pduName)
	}

	ip, err := ipamutil.ParseIPv6Address(ips[len(ips)-16:])
	if err != nil {
		return nil, err
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return nil, nil
	}

	mask := net.CIDRMask(prefixLen, 128)
	return &net.IPNet{
		IP:   ip.Mask(mask),
		Mask: mask,
	}, nil
}

func (n *NetworkEquipment) getSubnetByOIDIpv6AddrPfxLength() ([]net.IPNet, error) {
	var subnets []net.IPNet
	if err := n.walkOID(OIDIpv6AddrPfxLength, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return fmt.Errorf("prefix length return no integer but %v", pdu.Type)
		}

		ipnet, err := getIpNet6FromPduName(pdu.Name, OIDIpv6AddrPfxLength, pdu.Value.(int))
		if err != nil {
			return err
		}

		if ipnet != nil {
			subnets = append(subnets, *ipnet)
		}

		return nil
	}); err != nil {
		return nil, err
	} else {
		return subnets, nil
	}
}

func (n *NetworkEquipment) refreshV6IPAndMacs(subnets []*Subnet) error {
	return n.walkOID(OIDIpv6NetToMediaPhysAddress, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.OctetString {
			return nil
		}

		macBytes := pdu.Value.([]byte)
		if bytes.Equal(macBytes, AllZeroMAC) {
			return nil
		}

		ips := strings.Split(strings.TrimPrefix(pdu.Name, OIDIpv6NetToMediaPhysAddress), ".")
		if len(ips) != 18 {
			return fmt.Errorf("invalid arp entry name %s", pdu.Name)
		}

		ip, err := ipamutil.ParseIPv6Address(ips[2:])
		if err != nil {
			return fmt.Errorf("arp entry name %s has invalid ip address", pdu.Name)
		}

		for _, subnet := range subnets {
			if subnet.IPNet.Contains(ip) {
				subnet.NICs[ip.String()] = &NIC{
					IP:  ip.String(),
					Mac: net.HardwareAddr(macBytes).String(),
				}
				break
			}
		}

		return nil
	})
}

func (n *NetworkEquipment) refreshRouteInfo() error {
	if n.IsRouteEquipment() == false {
		return nil
	}

	routes, err := n.getRouteInfoFromIpRouteEntry(OIDIpRouteType, OIDIpRouteNextHop, OIDIpRouteIfIndex)
	if err != nil {
		return err
	}

	if len(routes) == 0 {
		routes, err = n.getRouteInfoFromIpRouteEntry(OIDIpCidrRouteType, OIDIpCidrRouteNextHop, OIDIpCidrRouteIfIndex)
		if err != nil {
			return err
		}
	}

	n.NextHopAddresses = make(map[string]LinkedNetworkEquipment)
	for _, route := range routes {
		n.NextHopAddresses[route.IfName] = LinkedNetworkEquipment{Ip: route.NextHop}
	}

	return nil
}

func (n *NetworkEquipment) getRouteInfoFromIpRouteEntry(oidIpRouteType, oidIpRouteNextHop, oidIpRouteIfIndex string) ([]*Route, error) {
	routes, err := n.getRoutesByIpRouteType(oidIpRouteType)
	if err != nil {
		return nil, err
	}

	if len(routes) == 0 {
		return nil, nil
	}

	if err := n.refreshRoutesByIpRouteNextHop(oidIpRouteNextHop, routes); err != nil {
		return nil, err
	}

	if err := n.refreshRoutesByIpRouteIfIndex(oidIpRouteIfIndex, routes); err != nil {
		return nil, err
	}

	return routes, nil
}

func (n *NetworkEquipment) getRoutesByIpRouteType(oid string) ([]*Route, error) {
	var routes []*Route
	if err := n.walkOID(oid, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return fmt.Errorf("get route type return no integer but %v", pdu.Type)
		}

		routeType := pdu.Value.(int)
		if routeType != 4 {
			return nil
		}

		addrMaskNextHop := strings.TrimPrefix(pdu.Name, oid)
		_, err := parseIpFromPduNameSuffix(addrMaskNextHop)
		if err != nil {
			return err
		}

		routes = append(routes, &Route{AddrMaskNextHop: addrMaskNextHop})
		return nil
	}); err != nil {
		return nil, err
	} else {
		return routes, nil
	}
}

func (n *NetworkEquipment) refreshRoutesByIpRouteNextHop(oid string, routes []*Route) error {
	return n.walkOID(oid, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.IPAddress {
			return fmt.Errorf("get route nexthop return no ip address but %v", pdu.Type)
		}

		addrMaskNextHop := strings.TrimPrefix(pdu.Name, oid)
		_, err := parseIpFromPduNameSuffix(addrMaskNextHop)
		if err != nil {
			return err
		}

		for _, route := range routes {
			if route.AddrMaskNextHop == addrMaskNextHop {
				route.NextHop = pdu.Value.(string)
				break
			}
		}

		return nil
	})
}

func (n *NetworkEquipment) refreshRoutesByIpRouteIfIndex(oid string, routes []*Route) error {
	return n.walkOID(oid, func(pdu gosnmp.SnmpPDU) error {
		if pdu.Type != gosnmp.Integer {
			return fmt.Errorf("get route ifIndex return no integer but %v", pdu.Type)
		}

		addrMaskNextHop := strings.TrimPrefix(pdu.Name, oid)
		_, err := parseIpFromPduNameSuffix(addrMaskNextHop)
		if err != nil {
			return err
		}

		for _, route := range routes {
			if route.AddrMaskNextHop == addrMaskNextHop {
				if ifName, ok := n.IfNames[pdu.Value.(int)]; ok {
					route.IfName = ifName
					break
				}
			}
		}

		return nil
	})
}

func (n *NetworkEquipment) walkOID(oid string, handler gosnmp.WalkFunc) error {
	return n.withConn(func(conn *gosnmp.GoSNMP) error {
		return conn.Walk(oid, handler)
	})
}

func (n *NetworkEquipment) withConn(f func(*gosnmp.GoSNMP) error) error {
	snmpPort := SNMPPort
	if n.SnmpPort != 0 {
		snmpPort = n.SnmpPort
	}
	conn := &gosnmp.GoSNMP{
		Target:             n.AdministrationAddress,
		Port:               snmpPort,
		Transport:          "udp",
		Community:          n.SnmpCommunity,
		Version:            gosnmp.Version2c,
		Timeout:            SNMPConnTimeout,
		Retries:            SNMPConnRetryCount,
		ExponentialTimeout: true,
		MaxOids:            SNMPGetMaxOIDS,
	}
	if err := conn.Connect(); err != nil {
		return err
	}
	defer conn.Conn.Close()

	return f(conn)
}
