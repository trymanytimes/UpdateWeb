package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/authentification"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	FilterNameAdminAddr     = "administration_address"
	FilterNameEquipmentType = "equipment_type"
	FilterNameManufacturer  = "manufacturer"
)

var TableNetworkEquipment = restdb.ResourceDBType(&resource.NetworkEquipment{})

type NetworkEquipmentHandler struct {
}

func NewNetworkEquipmentHandler() *NetworkEquipmentHandler {
	return &NetworkEquipmentHandler{}
}

func (h *NetworkEquipmentHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	networkequipment := ctx.Resource.(*resource.NetworkEquipment)
	if err := networkequipment.Validate(); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidFormat, fmt.Sprintf("create networkequipment %s failed: %s", networkequipment.Name, err.Error()))
	}
	if !authentification.PrefixFilter(ctx, networkequipment.AdministrationAddress) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("the networkequipment is not allow for creating"))
	}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		networkequipment.SetID(networkequipment.Name)
		_, err := tx.Insert(networkequipment)
		return err
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("create networkequipment %s failed: %s", networkequipment.Name, err.Error()))
	}

	return networkequipment, nil
}

func (h *NetworkEquipmentHandler) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	var topologyConds map[string]interface{}
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameName, ctx.GetFilters()); ok {
		topologyConds = map[string]interface{}{"network_equipment": name}
	}

	var networkequipments []*resource.NetworkEquipment
	var networktopologies []*resource.NetworkTopology
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(util.GenStrConditionsFromFilters(ctx.GetFilters(), FilterNameName, FilterNameAdminAddr,
			FilterNameEquipmentType, FilterNameManufacturer), &networkequipments); err != nil {
			return err
		}

		return tx.Fill(topologyConds, &networktopologies)
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("list networkequipments info failed: %s", err.Error()))
	}

	refreshNetworkEquipmentsByTopology(networkequipments, networktopologies)
	var visible []*resource.NetworkEquipment
	for _, networkequipment := range networkequipments {
		if authentification.PrefixFilter(ctx, networkequipment.AdministrationAddress) {
			visible = append(visible, networkequipment)
		}
	}
	return visible, nil
}

func refreshNetworkEquipmentsByTopology(networkequipments []*resource.NetworkEquipment, networktopologies []*resource.NetworkTopology) {
	for _, equipment := range networkequipments {
		equipment.UplinkAddresses = make(map[string]resource.LinkedNetworkEquipment)
		equipment.DownlinkAddresses = make(map[string]resource.LinkedNetworkEquipment)
		equipment.NextHopAddresses = make(map[string]resource.LinkedNetworkEquipment)
		for _, topology := range networktopologies {
			if equipment.Name == topology.NetworkEquipment {
				linkedNetworkEquipment := resource.LinkedNetworkEquipment{
					Ip:   topology.LinkedNetworkEquipmentIp,
					Port: topology.LinkedNetworkEquipmentPort,
				}
				switch topology.NetworkEquipmentPortType {
				case resource.EquipmentPortTypeUplink:
					equipment.UplinkAddresses[topology.NetworkEquipmentPort] = linkedNetworkEquipment
				case resource.EquipmentPortTypeDownlink:
					equipment.DownlinkAddresses[topology.NetworkEquipmentPort] = linkedNetworkEquipment
				case resource.EquipmentPortTypeNextHop:
					equipment.NextHopAddresses[topology.NetworkEquipmentPort] = linkedNetworkEquipment
				}
			}
		}
	}
}

func (h *NetworkEquipmentHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	networkequipmentID := ctx.Resource.GetID()
	var networkequipments []*resource.NetworkEquipment
	var networktopologies []*resource.NetworkTopology
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: networkequipmentID}, &networkequipments); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"network_equipment": networkequipmentID}, &networktopologies)
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get networkequipment %s failed: %s", networkequipmentID, err.Error()))
	}

	if len(networkequipments) != 1 {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("no found networkequipment %s", networkequipmentID))
	}

	refreshNetworkEquipmentsByTopology(networkequipments, networktopologies)
	return networkequipments[0], nil
}

func (h *NetworkEquipmentHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	networkequipment := ctx.Resource.(*resource.NetworkEquipment)
	if !authentification.PrefixFilter(ctx, networkequipment.AdministrationAddress) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("the networkequipment is not allow for creating"))
	}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(TableNetworkEquipment, map[string]interface{}{
			"administration_address": networkequipment.AdministrationAddress,
			"equipment_type":         networkequipment.EquipmentType,
			"manufacturer":           networkequipment.Manufacturer,
			"serial_number":          networkequipment.SerialNumber,
			"firmware_version":       networkequipment.FirmwareVersion,
			"computer_room":          networkequipment.ComputerRoom,
			"computer_rack":          networkequipment.ComputerRack,
			"location":               networkequipment.Location,
			"department":             networkequipment.Department,
			"responsible_person":     networkequipment.ResponsiblePerson,
			"telephone":              networkequipment.Telephone,
			"snmp_port":              networkequipment.SnmpPort,
			"snmp_community":         networkequipment.SnmpCommunity,
			"administration_mac":     networkequipment.AdministrationMac,
		}, map[string]interface{}{restdb.IDField: networkequipment.GetID()})
		return err
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("update networkequipment %s failed: %s", networkequipment.GetID(), err.Error()))
	}

	return networkequipment, nil
}

func (h *NetworkEquipmentHandler) Delete(ctx *restresource.Context) *resterr.APIError {
	networkequipmentID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(TableNetworkEquipment, map[string]interface{}{restdb.IDField: networkequipmentID})
		return err
	}); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("delete networkequipment %s failed: %s", networkequipmentID, err.Error()))
	}

	return nil
}

func (h *NetworkEquipmentHandler) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameSNMP:
		return h.snmp(ctx)
	default:
		return nil, resterr.NewAPIError(resterr.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *NetworkEquipmentHandler) snmp(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	snmpConfig, ok := ctx.Resource.GetAction().Input.(*resource.SnmpConfig)
	if ok == false {
		return nil, resterr.NewAPIError(resterr.InvalidFormat, fmt.Sprintf("parse action snmp input invalid"))
	}

	networkEquipmentID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(TableNetworkEquipment, map[string]interface{}{
			"snmp_port":      snmpConfig.Port,
			"snmp_community": snmpConfig.Community,
		}, map[string]interface{}{restdb.IDField: networkEquipmentID})
		return err
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("update networkequipment %s failed: %s", networkEquipmentID, err.Error()))
	}

	return nil, nil
}
