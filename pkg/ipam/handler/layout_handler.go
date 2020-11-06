package handler

import (
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"

	ipamdb "github.com/linkingthing/ddi-controller/pkg/db"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
)

type LayoutHandler struct{}

func NewLayoutHandler() *LayoutHandler {
	return &LayoutHandler{}
}

func (h *LayoutHandler) Create(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	layout := ctx.Resource.(*ipamresource.Layout)
	layout.Plan = ctx.Resource.GetParent().GetID()
	if err := layout.SaveLayoutToDB(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return layout, nil
	}
}

func (h *LayoutHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	var layouts []*ipamresource.Layout
	if err := db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		return tx.Fill(map[string]interface{}{"plan": ctx.Resource.GetParent().GetID()}, &layouts)
	}); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return layouts, nil
	}
}

func (h *LayoutHandler) Get(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	if layout, err := ipamresource.LoadLayoutFromDB(ctx.Resource.GetID()); err != nil {
		return nil, goresterr.NewAPIError(goresterr.NotFound, err.Error())
	} else {
		return layout, nil
	}
}

func (h *LayoutHandler) Update(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	layout := ctx.Resource.(*ipamresource.Layout)
	layout.Plan = ctx.Resource.GetParent().GetID()
	if err := layout.UpdateToDB(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return layout, nil
	}
}

func (h *LayoutHandler) Delete(ctx *resource.Context) *goresterr.APIError {
	if err := ipamresource.DeleteLayoutFromDB(ctx.Resource.GetID()); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return nil
	}
}
