package handler

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/set"
	restdb "github.com/zdnscloud/gorest/db"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	alarm "github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/auth/authentification"
	"github.com/linkingthing/ddi-controller/pkg/db"
	dhcpresource "github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/grpcclient"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	ScanInterval = 60
)

var (
	ThresholdIDIPConflict = strings.ToLower(string(alarm.ThresholdNameIPConflict))
)

type (
	IPAMSubnets map[string]*ipamresource.Subnet
	DHCPSubnets map[string]*DHCPSubnet
)

type DHCPSubnet struct {
	subnet              *dhcpresource.Subnet
	pools               []*dhcpresource.Pool
	reservations        []*dhcpresource.Reservation
	staticAddresses     []*dhcpresource.StaticAddress
	capacity            uint64
	poolCapacity        uint64
	dynamicPoolCapacity uint64
	reservationRatio    string
	staticAddressRatio  string
	unmanagedRatio      string
}

type ScannedSubnetHandler struct {
	scannedSubnets map[string]*ScannedSubnetAndNICs
	lock           sync.RWMutex
}

type ScannedSubnetAndNICs struct {
	scannedSubnet     *ipamresource.ScannedSubnet
	networkInterfaces map[ipamresource.IPState][]*ipamresource.NetworkInterface
}

func NewScannedSubnetHandler() *ScannedSubnetHandler {
	h := &ScannedSubnetHandler{
		scannedSubnets: make(map[string]*ScannedSubnetAndNICs),
	}
	go h.NetScan()
	return h
}

func (h *ScannedSubnetHandler) NetScan() {
	ticker := time.NewTicker(time.Duration(ScanInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var networkEquipments []*ipamresource.NetworkEquipment
			if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
				return tx.FillEx(&networkEquipments, "select * from gr_network_equipment where administration_address != '' and snmp_community != ''")
			}); err != nil {
				log.Warnf("get network equipment failed: %s", err.Error())
				continue
			}

			resultCh, _ := errgroup.Batch(networkEquipments, func(networkEquipment interface{}) (interface{}, error) {
				ipamNetworkEquipment := networkEquipment.(*ipamresource.NetworkEquipment)
				subnets, err := ipamNetworkEquipment.RefreshSubnet()
				if err != nil {
					log.Warnf("refresh subnets with address %s and community %s failed: %s",
						ipamNetworkEquipment.AdministrationAddress, ipamNetworkEquipment.SnmpCommunity, err.Error())
				}

				return subnets, nil
			})

			ipamSubnets := make(IPAMSubnets)
			for result := range resultCh {
				for _, subnet := range result.([]*ipamresource.Subnet) {
					if s, ok := ipamSubnets[subnet.IPNet.String()]; ok {
						for _, nic := range subnet.NICs {
							s.NICs[nic.IP] = nic
						}

						ipamSubnets[subnet.IPNet.String()] = s
					} else {
						ipamSubnets[subnet.IPNet.String()] = subnet
					}
				}
			}

			refreshNetworkEquipments(networkEquipments, ipamSubnets)
			if err := h.loadDHCPLeases(ipamSubnets); err != nil {
				log.Warnf("load leases failed: %s", err.Error())
			}

			if err := updateNetworkEquipmentToDB(networkEquipments); err != nil {
				log.Warnf("update network equipment after netscan failed: %s", err.Error())
			}
		}
	}
}

func refreshNetworkEquipments(networkEquipments []*ipamresource.NetworkEquipment, ipamSubnets IPAMSubnets) {
	macAndEquipments := getMacAndItsNetworkEquipment(networkEquipments, ipamSubnets)
	refreshIPLocation(networkEquipments, ipamSubnets, macAndEquipments)
	refreshNetworkTopology(networkEquipments, macAndEquipments)
}

func getMacAndItsNetworkEquipment(equipments []*ipamresource.NetworkEquipment, ipamSubnets IPAMSubnets) map[string]*ipamresource.NetworkEquipment {
	macAndEquipments := make(map[string]*ipamresource.NetworkEquipment)
	for _, equipment := range equipments {
		if equipment.AdministrationMac != "" {
			macAndEquipments[equipment.AdministrationMac] = equipment
		}

		for _, subnet := range ipamSubnets {
			if nic, ok := subnet.NICs[equipment.AdministrationAddress]; ok {
				equipment.AdministrationMac = nic.Mac
				macAndEquipments[nic.Mac] = equipment
			} else {
				for _, port := range equipment.Ports {
					if nic, ok := subnet.NICs[port.Ip]; ok {
						macAndEquipments[nic.Mac] = equipment
					}
				}
			}
		}
	}

	return macAndEquipments
}

func refreshIPLocation(equipments []*ipamresource.NetworkEquipment, ipamSubnets IPAMSubnets, macAndEquipments map[string]*ipamresource.NetworkEquipment) {
	for _, equipment := range equipments {
		if equipment.IsSwitchEquipment() == false {
			continue
		}

		for ifname, port := range equipment.Ports {
			hasRouteMac := false
			hasOtherSwitchMac := false
			for mac := range port.LearnedMacs {
				if equipment_, ok := macAndEquipments[mac]; ok {
					if equipment_.IsRouteEquipment() {
						hasRouteMac = true
						port.Type = ipamresource.EquipmentPortTypeUplink
					} else if equipment_.IsSwitchEquipment() {
						hasOtherSwitchMac = true
					}
				}
			}

			if hasRouteMac == false {
				if hasOtherSwitchMac {
					port.Type = ipamresource.EquipmentPortTypeDownlink
				} else {
					port.Type = ipamresource.EquipmentPortTypeAccess
					for _, subnet := range ipamSubnets {
						for _, nic := range subnet.NICs {
							if port.LearnedMacs.Member(nic.Mac) {
								nic.SwitchPortName = ifname
								nic.SwitchName = equipment.Name
								nic.ComputerRoom = equipment.ComputerRoom
								nic.ComputerRack = equipment.ComputerRack
								nic.VlanId = equipment.VlanIds[ifname]
							}
						}
					}
				}
			} else if hasOtherSwitchMac == false {
				port.Type = ipamresource.EquipmentPortTypeNextHop
			}
			equipment.Ports[ifname] = port
		}
	}
}

func refreshNetworkTopology(equipments []*ipamresource.NetworkEquipment, macAndEquipments map[string]*ipamresource.NetworkEquipment) {
	refreshNetLayerTopology(equipments)
	refreshLinkLayerTopology(equipments, macAndEquipments)
}

func refreshNetLayerTopology(equipments []*ipamresource.NetworkEquipment) {
	for _, equipment := range equipments {
		if equipment.IsRouteEquipment() == false {
			continue
		}

		nextHop := make(map[string]ipamresource.LinkedNetworkEquipment)
		for ifname, linkedNetworkEquipment := range equipment.NextHopAddresses {
			for _, equipment_ := range equipments {
				if linkedNetworkEquipment.Ip == equipment_.AdministrationAddress {
					nextHop[ifname] = ipamresource.LinkedNetworkEquipment{
						Ip: equipment_.AdministrationAddress,
					}
					break
				} else {
					found := false
					for ifname_, port_ := range equipment_.Ports {
						if port_.Ip == linkedNetworkEquipment.Ip {
							nextHop[ifname] = ipamresource.LinkedNetworkEquipment{
								Ip:   equipment_.AdministrationAddress,
								Port: ifname_,
							}
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}
		}
		equipment.NextHopAddresses = nextHop
	}
}

func refreshLinkLayerTopology(equipments []*ipamresource.NetworkEquipment, macAndEquipments map[string]*ipamresource.NetworkEquipment) {
	for _, equipment := range equipments {
		if equipment.IsSwitchEquipment() == false {
			continue
		}

		equipment.UplinkAddresses = make(map[string]ipamresource.LinkedNetworkEquipment)
		equipment.DownlinkAddresses = make(map[string]ipamresource.LinkedNetworkEquipment)
		for ifname, port := range equipment.Ports {
			if port.Type == ipamresource.EquipmentPortTypeAccess {
				continue
			}

			if port.Type == ipamresource.EquipmentPortTypeNextHop {
				for mac := range port.LearnedMacs {
					if equipment_, ok := macAndEquipments[mac]; ok {
						if equipment_.IsRouteEquipment() == false {
							break
						}

						if mac == equipment_.AdministrationMac {
							equipment.UplinkAddresses[ifname] = ipamresource.LinkedNetworkEquipment{
								Ip: equipment_.AdministrationAddress,
							}
						} else {
							for ifname_, port_ := range equipment_.Ports {
								if port_.Mac == mac {
									equipment.UplinkAddresses[ifname] = ipamresource.LinkedNetworkEquipment{
										Ip:   equipment_.AdministrationAddress,
										Port: ifname_,
									}
									break
								}
							}
						}
					}
				}
				continue
			}

			relationshipEquipments := make(map[string]map[string]*ipamresource.NetworkEquipment)
			for mac := range port.LearnedMacs {
				equipment_, ok := macAndEquipments[mac]
				if ok == false {
					continue
				}

				for ifname_, port_ := range equipment_.Ports {
					if port_.Type == ipamresource.EquipmentPortTypeAccess {
						continue
					}

					if port_.LearnedMacs.Member(equipment.AdministrationMac) {
						relationshipEquipments[equipment_.Name] = map[string]*ipamresource.NetworkEquipment{
							ifname_: equipment_,
						}
						break
					} else {
						found := false
						for _, port := range equipment.Ports {
							if port_.LearnedMacs.Member(port.Mac) {
								relationshipEquipments[equipment_.Name] = map[string]*ipamresource.NetworkEquipment{
									ifname_: equipment_,
								}
								found = true
								break
							}
						}
						if found {
							break
						}
					}
				}
			}

			for _, ifnameAndEquipments := range relationshipEquipments {
				for ifname_, equipment_ := range ifnameAndEquipments {
					if port_, ok := equipment_.Ports[ifname_]; ok {
						if setsHasIntersection(port.LearnedMacs, port_.LearnedMacs) == false {
							if port.Type == ipamresource.EquipmentPortTypeUplink {
								equipment.UplinkAddresses[ifname] = ipamresource.LinkedNetworkEquipment{
									Ip:   equipment_.AdministrationAddress,
									Port: ifname_,
								}
							} else {
								equipment.DownlinkAddresses[ifname] = ipamresource.LinkedNetworkEquipment{
									Ip:   equipment_.AdministrationAddress,
									Port: ifname_,
								}
							}
							break
						}
					}
				}
			}
		}
	}
}

func setsHasIntersection(macs, others set.StringSet) bool {
	for mac := range macs {
		if others.Member(mac) {
			return true
		}
	}

	return false
}

var TableNetworkTopology = restdb.ResourceDBType(&ipamresource.NetworkTopology{})

func updateNetworkEquipmentToDB(equipments []*ipamresource.NetworkEquipment) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, equipment := range equipments {
			if _, err := tx.Update(TableNetworkEquipment, map[string]interface{}{
				"lastRefreshTime": equipment.LastRefreshTime,
			}, map[string]interface{}{restdb.IDField: equipment.Name}); err != nil {
				return err
			}

			if err := addOrUpdateNetworkTopology(tx, equipment.Name, ipamresource.EquipmentPortTypeUplink, equipment.UplinkAddresses); err != nil {
				return fmt.Errorf("add or update network topology for network equipment %s with uplink failed: %s", equipment.Name, err.Error())
			}
			if err := addOrUpdateNetworkTopology(tx, equipment.Name, ipamresource.EquipmentPortTypeDownlink, equipment.DownlinkAddresses); err != nil {
				return fmt.Errorf("add or update network topology for network equipment %s with downlink failed: %s", equipment.Name, err.Error())
			}
			if err := addOrUpdateNetworkTopology(tx, equipment.Name, ipamresource.EquipmentPortTypeNextHop, equipment.NextHopAddresses); err != nil {
				return fmt.Errorf("add or update network topology for network equipment %s with nexthop failed: %s", equipment.Name, err.Error())
			}
		}
		return nil
	})
}

func addOrUpdateNetworkTopology(tx restdb.Transaction, equipmentName string, portType ipamresource.EquipmentPortType, linkedEquipments map[string]ipamresource.LinkedNetworkEquipment) error {
	for ifname, linkedEquipment := range linkedEquipments {
		var topologies []*ipamresource.NetworkTopology
		if err := tx.Fill(map[string]interface{}{
			"network_equipment":      equipmentName,
			"network_equipment_port": ifname},
			&topologies); err != nil {
			return err
		} else if len(topologies) == 0 {
			if _, err := tx.Insert(&ipamresource.NetworkTopology{
				NetworkEquipment:           equipmentName,
				NetworkEquipmentPort:       ifname,
				NetworkEquipmentPortType:   portType,
				LinkedNetworkEquipmentIp:   linkedEquipment.Ip,
				LinkedNetworkEquipmentPort: linkedEquipment.Port,
			}); err != nil {
				return err
			}
		} else {
			if _, err := tx.Update(TableNetworkTopology, map[string]interface{}{
				"network_equipment_port_type":   portType,
				"linked_network_equipment_ip":   linkedEquipment.Ip,
				"linked_network_equipment_port": linkedEquipment.Port,
			}, map[string]interface{}{restdb.IDField: topologies[0].GetID()}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *ScannedSubnetHandler) loadDHCPLeases(ipamSubnets IPAMSubnets) error {
	dhcpSubnets, err := loadDHCPSubnets()
	if err != nil {
		return err
	}

	return h.loadLeases(dhcpSubnets, ipamSubnets)
}

func loadDHCPSubnets() (DHCPSubnets, error) {
	var dhcpSubnets []*dhcpresource.Subnet
	var dhcpPools []*dhcpresource.Pool
	var dhcpReservations []*dhcpresource.Reservation
	var dhcpStaticAddresses []*dhcpresource.StaticAddress
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(nil, &dhcpSubnets); err != nil {
			return fmt.Errorf("load subnets from db failed: %s", err.Error())
		}

		if err := tx.Fill(nil, &dhcpPools); err != nil {
			return fmt.Errorf("load pools from db failed: %s", err.Error())
		}

		if err := tx.Fill(nil, &dhcpReservations); err != nil {
			return fmt.Errorf("load reservations from db failed: %s", err.Error())
		}

		if err := tx.Fill(nil, &dhcpStaticAddresses); err != nil {
			return fmt.Errorf("load static addresses from db failed: %s", err.Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	subnets := make(DHCPSubnets)
	for _, dhcpSubnet := range dhcpSubnets {
		subnet := &DHCPSubnet{
			subnet: dhcpSubnet,
		}

		var dynamicPoolCapacity uint64
		for _, pool := range dhcpPools {
			if pool.Subnet == dhcpSubnet.GetID() {
				subnet.pools = append(subnet.pools, pool)
				dynamicPoolCapacity += pool.Capacity
			}
		}

		var reservationCapacity uint64
		for _, reservation := range dhcpReservations {
			if reservation.Subnet == dhcpSubnet.GetID() {
				subnet.reservations = append(subnet.reservations, reservation)
				reservationCapacity += reservation.Capacity
				if ipBelongsToPool(reservation.IpAddress, subnet.pools) {
					dynamicPoolCapacity -= reservation.Capacity
				}
			}
		}

		var staticAddressCapacity uint64
		for _, staticAddress := range dhcpStaticAddresses {
			if staticAddress.Subnet == dhcpSubnet.GetID() {
				subnet.staticAddresses = append(subnet.staticAddresses, staticAddress)
				staticAddressCapacity += staticAddress.Capacity
			}
		}

		_, ipnet, _ := net.ParseCIDR(dhcpSubnet.Ipnet)
		ones, _ := ipnet.Mask.Size()
		if dhcpSubnet.Version == dhcpresource.Version4 {
			subnet.capacity = uint64(1<<(32-ones) - 1)
		} else {
			subnet.capacity = uint64(1<<64 - 1)
		}

		subnet.poolCapacity = dynamicPoolCapacity + reservationCapacity + staticAddressCapacity
		subnet.reservationRatio = calculateRatio(reservationCapacity, subnet.poolCapacity)
		subnet.staticAddressRatio = calculateRatio(staticAddressCapacity, subnet.poolCapacity)
		subnet.unmanagedRatio = calculateRatio(subnet.capacity-subnet.poolCapacity, subnet.capacity)
		subnet.dynamicPoolCapacity = dynamicPoolCapacity
		subnets[dhcpSubnet.Ipnet] = subnet
	}

	return subnets, nil
}

func (h *ScannedSubnetHandler) loadLeases(dhcpSubnets DHCPSubnets, ipamSubnets IPAMSubnets) error {
	var thresholds []*alarm.Threshold
	if _, err := restdb.GetResourceWithID(db.GetDB(), ThresholdIDIPConflict, &thresholds); err != nil {
		log.Warnf("load threshold ipconflict failed: %s", err.Error())
	}

	scannedSubnets := make(map[string]*ScannedSubnetAndNICs)
	for ipnet, dhcpSubnet := range dhcpSubnets {
		var ipStateActiveCount, ipStateInactiveCount, ipStateZombieCount, ipStateConflictCount, ipTypeAssignCount uint64
		resp, err := loadSubnetLeases(dhcpSubnet.subnet)
		if err != nil {
			log.Warnf("load subnet %s failed: %s", ipnet, err.Error())
			continue
		}

		networkInterfaces := make(map[ipamresource.IPState][]*ipamresource.NetworkInterface)
		if ipamSubnet, ok := ipamSubnets[ipnet]; ok {
			for _, nic := range ipamSubnet.NICs {
				ipType := ipamresource.IPTypeUnmanagered
				ipState := ipamresource.IPStateConflict
				reservated := ipBelongsToReservation(nic.IP, dhcpSubnet.reservations)
				static := ipBelongsToStaticAddress(nic.IP, dhcpSubnet.staticAddresses)
				if reservated || static || ipBelongsToPool(nic.IP, dhcpSubnet.pools) {
					hasLeaseInfo := false
					ipType = ipamresource.IPTypeUnassigned
					if reservated {
						ipType = ipamresource.IPTypeReservation
					} else if static {
						ipType = ipamresource.IPTypeStatic
					}

					for _, lease := range resp.GetLeases() {
						if lease.GetAddress() == nic.IP {
							hasLeaseInfo = true
							if static {
								ipStateConflictCount += 1
							} else {
								if reservated == false {
									ipType = ipamresource.IPTypeAssigned
									ipTypeAssignCount += 1
								}

								if lease.GetHwAddress() == nic.Mac {
									ipState = ipamresource.IPStateActive
									ipStateActiveCount += 1
								} else {
									ipStateConflictCount += 1
								}
							}

							networkInterfaces[ipState] = append(networkInterfaces[ipState], leaseToNetworkInterface(lease, ipType, ipState, nic))
							break
						}
					}

					if hasLeaseInfo == false {
						if static {
							ipState = ipamresource.IPStateActive
							ipStateActiveCount += 1
						} else {
							ipStateConflictCount += 1
						}
						networkInterfaces[ipState] = append(networkInterfaces[ipState], nicToNetworkInterface(nic, ipType, ipState))
					}
				} else {
					networkInterfaces[ipState] = append(networkInterfaces[ipState], nicToNetworkInterface(nic, ipType, ipState))
					ipStateConflictCount += 1
				}

				if ipState == ipamresource.IPStateConflict {
					sendEventWithConflictIPIfNeed(nic.IP, ipType, thresholds)
				}
			}

			for _, lease := range resp.GetLeases() {
				scanned := false
				for _, nic := range ipamSubnet.NICs {
					if nic.IP == lease.GetAddress() {
						scanned = true
						break
					}
				}

				if scanned == false {
					networkInterface := leaseNoScannedToNetworkInterface(lease, dhcpSubnet.reservations)
					if networkInterface.IpType == ipamresource.IPTypeReservation {
						ipStateZombieCount += 1
					} else {
						ipStateInactiveCount += 1
						ipTypeAssignCount += 1
					}
					networkInterfaces[networkInterface.IpState] = append(networkInterfaces[networkInterface.IpState], networkInterface)
				}
			}

			for _, staticAddress := range dhcpSubnet.staticAddresses {
				scanned := false
				for _, nic := range ipamSubnet.NICs {
					if nic.IP == staticAddress.IpAddress {
						scanned = true
						break
					}
				}

				if scanned == false {
					ipStateInactiveCount += 1
					networkInterfaces[ipamresource.IPStateInactive] = append(networkInterfaces[ipamresource.IPStateInactive],
						staticAddressToNetworkInterface(staticAddress, ipamresource.IPStateInactive))
				}
			}

		} else {
			for _, lease := range resp.GetLeases() {
				networkInterface := leaseNoScannedToNetworkInterface(lease, dhcpSubnet.reservations)
				if networkInterface.IpType == ipamresource.IPTypeReservation {
					ipStateZombieCount += 1
				} else {
					ipStateInactiveCount += 1
					ipTypeAssignCount += 1
				}
				networkInterfaces[networkInterface.IpState] = append(networkInterfaces[networkInterface.IpState], networkInterface)
			}

			for _, staticAddress := range dhcpSubnet.staticAddresses {
				ipStateInactiveCount += 1
				networkInterfaces[ipamresource.IPStateInactive] = append(networkInterfaces[ipamresource.IPStateInactive],
					staticAddressToNetworkInterface(staticAddress, ipamresource.IPStateInactive))
			}
		}

		ipStateTotal := ipStateActiveCount + ipStateInactiveCount + ipStateConflictCount + ipStateZombieCount
		scannedSubnet := &ipamresource.ScannedSubnet{
			Ipnet:              ipnet,
			SubnetId:           dhcpSubnet.subnet.SubnetId,
			Tags:               dhcpSubnet.subnet.Tags,
			AssignedRatio:      calculateRatio(ipTypeAssignCount, dhcpSubnet.poolCapacity),
			UnassignedRatio:    calculateRatio(dhcpSubnet.dynamicPoolCapacity-ipTypeAssignCount, dhcpSubnet.poolCapacity),
			ReservationRatio:   dhcpSubnet.reservationRatio,
			StaticAddressRatio: dhcpSubnet.staticAddressRatio,
			UnmanagedRatio:     dhcpSubnet.unmanagedRatio,
			ActiveRatio:        calculateRatio(ipStateActiveCount, ipStateTotal),
			InactiveRatio:      calculateRatio(ipStateInactiveCount, ipStateTotal),
			ConflictRatio:      calculateRatio(ipStateConflictCount, ipStateTotal),
			ZombieRatio:        calculateRatio(ipStateZombieCount, ipStateTotal),
		}

		subnetID := strconv.Itoa(int(dhcpSubnet.subnet.SubnetId))
		scannedSubnet.SetID(subnetID)
		scannedSubnets[subnetID] = &ScannedSubnetAndNICs{
			scannedSubnet:     scannedSubnet,
			networkInterfaces: networkInterfaces,
		}
	}

	h.lock.Lock()
	h.scannedSubnets = scannedSubnets
	h.lock.Unlock()
	return nil
}

func loadSubnetLeases(subnet *dhcpresource.Subnet) (*pb.GetLeasesResponse, error) {
	var resp *pb.GetLeasesResponse
	var err error
	if subnet.Version == dhcpresource.Version4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet4Leases(context.TODO(),
			&pb.GetSubnet4LeasesRequest{Id: subnet.SubnetId})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet6Leases(context.TODO(),
			&pb.GetSubnet6LeasesRequest{Id: subnet.SubnetId})
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func calculateRatio(numerator uint64, denominator uint64) string {
	if denominator == 0 {
		return ""
	}

	return fmt.Sprintf("%.4f", float64(numerator)/float64(denominator))
}

func ipBelongsToReservation(ip string, reservations []*dhcpresource.Reservation) bool {
	for _, reservation := range reservations {
		if reservation.IpAddress == ip {
			return true
		}
	}

	return false
}

func ipBelongsToPool(ip string, pools []*dhcpresource.Pool) bool {
	for _, pool := range pools {
		if pool.Contains(ip) {
			return true
		}
	}

	return false
}

func ipBelongsToStaticAddress(ip string, staticAddresses []*dhcpresource.StaticAddress) bool {
	for _, staticAddress := range staticAddresses {
		if staticAddress.IpAddress == ip {
			return true
		}
	}

	return false
}

func leaseToNetworkInterface(lease *pb.DHCPLease, typ ipamresource.IPType, state ipamresource.IPState, nic *ipamresource.NIC) *ipamresource.NetworkInterface {
	networkInterface := &ipamresource.NetworkInterface{
		Ip:            lease.GetAddress(),
		Mac:           lease.GetHwAddress(),
		Hostname:      lease.GetHostname(),
		ValidLifetime: lease.GetValidLifetime(),
		Expire:        restresource.ISOTime(time.Unix(lease.GetExpire(), 0)),
		IpType:        typ,
		IpState:       state,
	}
	if nic != nil && nic.SwitchPortName != "" {
		networkInterface.ComputerRoom = nic.ComputerRoom
		networkInterface.ComputerRack = nic.ComputerRack
		networkInterface.SwitchName = nic.SwitchName
		networkInterface.SwitchPortName = nic.SwitchPortName
		networkInterface.VlanId = nic.VlanId
	}

	networkInterface.SetID(lease.GetAddress())
	return networkInterface
}

func nicToNetworkInterface(nic *ipamresource.NIC, typ ipamresource.IPType, state ipamresource.IPState) *ipamresource.NetworkInterface {
	networkInterface := &ipamresource.NetworkInterface{
		Ip:             nic.IP,
		Mac:            nic.Mac,
		IpType:         typ,
		IpState:        state,
		ComputerRoom:   nic.ComputerRoom,
		ComputerRack:   nic.ComputerRack,
		SwitchName:     nic.SwitchName,
		SwitchPortName: nic.SwitchPortName,
		VlanId:         nic.VlanId,
	}
	networkInterface.SetID(nic.IP)
	return networkInterface
}

func leaseNoScannedToNetworkInterface(lease *pb.DHCPLease, reservations []*dhcpresource.Reservation) *ipamresource.NetworkInterface {
	ipType := ipamresource.IPTypeAssigned
	ipState := ipamresource.IPStateInactive
	if ipBelongsToReservation(lease.GetAddress(), reservations) {
		ipType = ipamresource.IPTypeReservation
		ipState = ipamresource.IPStateZombie
	}

	return leaseToNetworkInterface(lease, ipType, ipState, nil)
}

func staticAddressToNetworkInterface(staticAddress *dhcpresource.StaticAddress, state ipamresource.IPState) *ipamresource.NetworkInterface {
	networkInterface := &ipamresource.NetworkInterface{
		Ip:      staticAddress.IpAddress,
		Mac:     staticAddress.HwAddress,
		IpType:  ipamresource.IPTypeStatic,
		IpState: state,
	}

	networkInterface.SetID(staticAddress.IpAddress)
	return networkInterface
}

func sendEventWithConflictIPIfNeed(ip string, typ ipamresource.IPType, thresholds []*alarm.Threshold) {
	if len(thresholds) != 1 {
		return
	}

	alarm.NewEvent().Name(thresholds[0].Name).Level(thresholds[0].Level).ThresholdType(thresholds[0].ThresholdType).
		Time(time.Now()).SendMail(thresholds[0].SendMail).ConflictIp(ip).ConflictIpType(string(typ)).Publish()
}

func (h *ScannedSubnetHandler) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	var subnets ipamresource.ScannedSubnets
	for _, subnet := range h.scannedSubnets {
		subnets = append(subnets, subnet.scannedSubnet)
	}

	sort.Sort(subnets)
	var visible ipamresource.ScannedSubnets
	for _, subnet := range subnets {
		if authentification.PrefixFilter(ctx, subnet.Ipnet) {
			visible = append(visible, subnet)
		}
	}

	return visible, nil
}

func (h *ScannedSubnetHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if subnet, ok := h.scannedSubnets[ctx.Resource.GetID()]; ok {
		return subnet.scannedSubnet, nil
	}

	return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("no found scanned subnet %s", ctx.Resource.GetID()))
}

func (h *ScannedSubnetHandler) getNetworkInterfacesByNicID(scannedSubnetID, networkInterfaceID string) *ipamresource.NetworkInterface {
	h.lock.RLock()
	subnet, ok := h.scannedSubnets[scannedSubnetID]
	h.lock.RUnlock()
	if ok == false {
		return nil
	}

	for _, nics := range subnet.networkInterfaces {
		for _, nic := range nics {
			if nic.GetID() == networkInterfaceID {
				return nic
			}
		}
	}

	return nil
}

func (h *ScannedSubnetHandler) getNetworkInterfacesByFilter(scannedSubnetID string, filter *NicFilter) ipamresource.NetworkInterfaces {
	h.lock.RLock()
	subnet, ok := h.scannedSubnets[scannedSubnetID]
	h.lock.RUnlock()
	if ok == false {
		return nil
	}

	var nics ipamresource.NetworkInterfaces
	for _, nics_ := range subnet.networkInterfaces {
		nics = append(nics, nics_...)
	}

	if filter == nil {
		return nics
	}

	if filter.IpState.IsEmpty() == false {
		if nics_, ok := subnet.networkInterfaces[filter.IpState]; ok == false {
			return nil
		} else {
			nics = nics_
		}
	}

	var networkInterfaces ipamresource.NetworkInterfaces
	for _, nic := range nics {
		if (filter.Mac != "" && filter.Mac != nic.Mac) ||
			(filter.Ip != "" && filter.Ip != nic.Ip) {
			continue
		}

		networkInterfaces = append(networkInterfaces, nic)
	}

	return networkInterfaces
}

func (h *ScannedSubnetHandler) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	switch ctx.Resource.GetAction().Name {
	case ipamresource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterr.NewAPIError(resterr.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

var TableHeaderScannedSubnet = []string{"IP地址", "MAC地址", "地址类型", "地址状态", "租赁时间", "租赁过期时间"}

func (h *ScannedSubnetHandler) export(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	scannedSubnetID := ctx.Resource.GetID()
	h.lock.RLock()
	subnet, ok := h.scannedSubnets[scannedSubnetID]
	h.lock.RUnlock()
	if ok == false {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("no found subnet %s", scannedSubnetID))
	}

	var strMatrix [][]string
	for _, nics := range subnet.networkInterfaces {
		sort.Sort(ipamresource.NetworkInterfaces(nics))
		for _, nic := range nics {
			strMatrix = append(strMatrix, localizationNicToStrSlice(nic))
		}
	}

	filepath := fmt.Sprintf(util.CSVFilePath, "subnet-"+scannedSubnetID+"-"+time.Now().Format(util.TimeFormat))
	if err := util.GenCSVFile(filepath, TableHeaderScannedSubnet, strMatrix); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("export subnet %s failed: %s", scannedSubnetID, err.Error()))
	}

	return &ipamresource.FileInfo{Path: filepath}, nil
}
