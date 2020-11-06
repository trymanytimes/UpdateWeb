package handler

import (
	"fmt"
	"sort"

	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	FilterNameIpState = "ipstate"
)

type NicFilter struct {
	IpState ipamresource.IPState
	Mac     string
	Ip      string
}

type NetworkInterfaceHandler struct {
	scannedSubnetHandler *ScannedSubnetHandler
}

func NewNetworkInterfaceHandler(scannedSubnetHandler *ScannedSubnetHandler) *NetworkInterfaceHandler {
	return &NetworkInterfaceHandler{scannedSubnetHandler}
}

func (h *NetworkInterfaceHandler) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	scannedSubnetID := ctx.Resource.GetParent().GetID()
	filter, ok := getNicFilter(ctx.GetFilters())
	if ok == false {
		return nil, nil
	}

	networkInterfaces := h.scannedSubnetHandler.getNetworkInterfacesByFilter(scannedSubnetID, filter)
	sort.Sort(networkInterfaces)
	return networkInterfaces, nil
}

func getNicFilter(filters []restresource.Filter) (*NicFilter, bool) {
	if len(filters) == 0 {
		return nil, true
	}

	filter := NicFilter{}
	if ipstate, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameIpState, filters); ok {
		switch state := ipamresource.IPState(ipstate); state {
		case ipamresource.IPStateActive, ipamresource.IPStateInactive, ipamresource.IPStateConflict, ipamresource.IPStateZombie:
			filter.IpState = state
		default:
			return nil, false
		}
	}

	if mac, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameMac, filters); ok {
		filter.Mac = mac
	}

	if ip, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameIp, filters); ok {
		filter.Ip = ip
	}

	if filter.IpState.IsEmpty() && filter.Mac == "" && filter.Ip == "" {
		return nil, true
	}

	return &filter, true
}

func (h *NetworkInterfaceHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	networkInterfaceID := ctx.Resource.GetID()
	networkInterface := h.scannedSubnetHandler.getNetworkInterfacesByNicID(ctx.Resource.GetParent().GetID(), networkInterfaceID)
	if networkInterface == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("no found network interface %s", networkInterfaceID))
	} else {
		return networkInterface, nil
	}
}
