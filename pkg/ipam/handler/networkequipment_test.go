package handler

import (
	"fmt"
	"testing"

	"github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	ut "github.com/zdnscloud/cement/unittest"
)

func TestRouter(t *testing.T) {
	nes := []*resource.NetworkEquipment{
		/*
			&resource.NetworkEquipment{
				Name:                  "cisco_r",
				AdministrationAddress: "10.0.0.2",
				EquipmentType:         resource.EquipmentTypeRouter,
				Manufacturer:          "cisco",
				ComputerRoom:          "航天城上城13栋1403",
				ComputerRack:          "0-1",
				SnmpCommunity:         "linking123",
				AdministrationMac:     "68:9c:e2:57:05:23",
			},
			&resource.NetworkEquipment{
				Name:                  "huawei_swl3",
				AdministrationAddress: "10.0.0.254",
				EquipmentType:         resource.EquipmentTypeCoreSwitch,
				Manufacturer:          "huawei",
				ComputerRoom:          "航天城上城13栋1403",
				ComputerRack:          "1-1",
				SnmpCommunity:         "linking123",
			},
			&resource.NetworkEquipment{
				Name:                  "h3c_swl2",
				AdministrationAddress: "10.0.0.253",
				EquipmentType:         resource.EquipmentTypeConvergenceSwitch,
				Manufacturer:          "h3c",
				ComputerRoom:          "航天城上城13栋1403",
				ComputerRack:          "2-1",
				SnmpCommunity:         "linking123",
			},
		*/
		&resource.NetworkEquipment{
			Name:                  "h3c_vsr2k",
			AdministrationAddress: "10.104.0.1",
			EquipmentType:         resource.EquipmentTypeRouter,
			Manufacturer:          "h3c",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-1",
			SnmpCommunity:         "linking123",
		},
		&resource.NetworkEquipment{
			Name:                  "hw_fw",
			AdministrationAddress: "10.104.0.3",
			EquipmentType:         resource.EquipmentTypeFirewall,
			Manufacturer:          "huawei",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-1",
			SnmpCommunity:         "linking123",
		},
		&resource.NetworkEquipment{
			Name:                  "cisco_r1",
			AdministrationAddress: "10.104.0.4",
			EquipmentType:         resource.EquipmentTypeRouter,
			Manufacturer:          "cisco",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-2",
			SnmpCommunity:         "linking123",
		},
		&resource.NetworkEquipment{
			Name:                  "cisco_r2",
			AdministrationAddress: "10.104.0.5",
			EquipmentType:         resource.EquipmentTypeRouter,
			Manufacturer:          "cisco",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-2",
			SnmpCommunity:         "linking123",
		},
		&resource.NetworkEquipment{
			Name:                  "cisco_sw1",
			AdministrationAddress: "10.106.0.254",
			EquipmentType:         resource.EquipmentTypeCoreSwitch,
			Manufacturer:          "cisco",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-3",
			SnmpCommunity:         "linking123",
		},
		&resource.NetworkEquipment{
			Name:                  "cisco_sw2",
			AdministrationAddress: "10.107.0.254",
			EquipmentType:         resource.EquipmentTypeCoreSwitch,
			Manufacturer:          "cisco",
			ComputerRoom:          "航天城上城13栋1403",
			ComputerRack:          "3-3",
			SnmpCommunity:         "linking123",
		},
	}
	err := refreshSubnet(nes)
	ut.Assert(t, err == nil, "")
}

func refreshSubnet(ns []*resource.NetworkEquipment) error {
	var subnets []*resource.Subnet
	for _, n := range ns {
		ss, err := n.RefreshSubnet()
		if err != nil {
			fmt.Printf("err: %v\n", err.Error())
			continue
		}
		subnets = append(subnets, ss...)
	}

	ss := refreshSubnetAndSwitch(subnets, ns)
	dump(ss, ns)
	return nil
}

func refreshSubnetAndSwitch(subnets []*resource.Subnet, ns []*resource.NetworkEquipment) map[string]*resource.Subnet {
	ipamSubnets := make(map[string]*resource.Subnet)
	for _, subnet := range subnets {
		if s, ok := ipamSubnets[subnet.IPNet.String()]; ok {
			for _, nic := range subnet.NICs {
				s.NICs[nic.IP] = nic
			}

			ipamSubnets[subnet.IPNet.String()] = s
		} else {
			ipamSubnets[subnet.IPNet.String()] = subnet
		}
	}

	refreshNetworkEquipments(ns, ipamSubnets)
	return ipamSubnets
}

func dump(subnets map[string]*resource.Subnet, ns []*resource.NetworkEquipment) {
	for _, subnet := range subnets {
		fmt.Printf("subnet :%s and nic count: %d\n", subnet.IPNet.String(), len(subnet.NICs))
		for _, nic := range subnet.NICs {
			fmt.Printf(" %s - %s and location: %s-%s-%s-%s-%d\n",
				nic.IP, nic.Mac, nic.ComputerRoom, nic.ComputerRack, nic.SwitchName, nic.SwitchPortName, nic.VlanId)
		}
	}
	fmt.Printf("\n")

	for _, n := range ns {
		fmt.Printf("-----> network equipment: %s  ip: %s and mac: %s\n", n.Name, n.AdministrationAddress, n.AdministrationMac)
		fmt.Printf("uplink:\n")
		for portName, addr := range n.UplinkAddresses {
			fmt.Printf("%s-%s:%s\n", portName, addr.Ip, addr.Port)
		}
		fmt.Printf("downlink:\n")
		for portName, addr := range n.DownlinkAddresses {
			fmt.Printf("%s-%s:%s\n", portName, addr.Ip, addr.Port)
		}

		fmt.Printf("nexthop:\n")
		for ifname, addr := range n.NextHopAddresses {
			fmt.Printf("%s-%s:%s\n", ifname, addr.Ip, addr.Port)
		}
	}
}
