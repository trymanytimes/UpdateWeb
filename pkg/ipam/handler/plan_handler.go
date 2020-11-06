package handler

import (
	"fmt"

	"github.com/linkingthing/ddi-controller/pkg/auth/handler"

	"github.com/zdnscloud/gorest/db"
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/authentification"
	ipamdb "github.com/linkingthing/ddi-controller/pkg/db"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
)

type PlanHandler struct{}

func NewPlanHandler() *PlanHandler {
	return &PlanHandler{}
}

func (h *PlanHandler) Create(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	user, ok := ctx.Get(authentification.AuthUser)
	if !ok || user != handler.Admin {
		return nil, goresterr.NewAPIError(goresterr.PermissionDenied,
			fmt.Sprintf("no right to create"))
	}

	plan := ctx.Resource.(*ipamresource.Plan)
	if err := plan.SavePlanToDB(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return plan, nil
	}
}

func (h *PlanHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	var plans []*ipamresource.Plan
	if err := db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		return tx.Fill(nil, &plans)
	}); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	}

	var visible []*ipamresource.Plan
	for _, plan := range plans {
		if authentification.PrefixFilter(ctx, plan.Prefix) {
			visible = append(visible, plan)
		}
	}

	return visible, nil
}

func (h *PlanHandler) Get(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	var plans []*ipamresource.Plan
	if plan, err := db.GetResourceWithID(ipamdb.GetDB(), ctx.Resource.GetID(), &plans); err != nil {
		return nil, goresterr.NewAPIError(goresterr.NotFound, err.Error())
	} else {
		return plan.(resource.Resource), nil
	}
}

func (h *PlanHandler) Delete(ctx *resource.Context) *goresterr.APIError {
	user, ok := ctx.Get(authentification.AuthUser)
	if !ok || user != handler.Admin {
		return goresterr.NewAPIError(goresterr.PermissionDenied,
			fmt.Sprintf("no right to create"))
	}

	if err := ipamresource.DeletePlanFromDB(ctx.Resource.GetID()); err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return nil
	}
}

func (h *PlanHandler) Update(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	user, ok := ctx.Get(authentification.AuthUser)
	if !ok || user != handler.Admin {
		return nil, goresterr.NewAPIError(goresterr.PermissionDenied,
			fmt.Sprintf("no right to create"))
	}

	plan := ctx.Resource.(*ipamresource.Plan)
	if err := plan.UpdatePlanToDB(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return plan, nil
	}
}
