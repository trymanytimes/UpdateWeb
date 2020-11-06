package handler

import (
	"github.com/zdnscloud/gorest/db"
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"

	ipamdb "github.com/linkingthing/ddi-controller/pkg/db"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

type NetNodeHandler struct{}

func NewNetNodeHandler() *NetNodeHandler {
	return &NetNodeHandler{}
}

func (h *NetNodeHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	netType, ok := util.GetFilterValueWithEqModifierFromFilters(ipamresource.NetType, ctx.GetFilters())
	if !ok || (netType != ipamresource.NetType_V4 && netType != ipamresource.NetType_V6) {
		return nil, goresterr.NewAPIError(goresterr.NotFound, "Net type unknown!")
	}

	layout, err := ipamresource.LoadLayoutFromDB(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.NotFound, err.Error())
	}

	var plans []*ipamresource.Plan
	plan_, err := db.GetResourceWithID(ipamdb.GetDB(), layout.Plan, &plans)
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.NotFound, "Plan is unknown!")
	}

	plan := plan_.(*ipamresource.Plan)
	netNodes, err := ipamresource.GetNetNodesList(plan, layout, netType)
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return netNodes, nil
	}
}
