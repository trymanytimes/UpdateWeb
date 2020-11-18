package handler

import (
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
)

type BalanceHandler struct{}

func NewBalanceHandler() *BalanceHandler {
	return &BalanceHandler{}
}

func (h *BalanceHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	Balance := ctx.Resource.(*resource.Balance)
	return Balance, nil
}

func (h *BalanceHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	return nil
}

func (h *BalanceHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	Balance := ctx.Resource.(*resource.Balance)
	return Balance, nil
}

func (h *BalanceHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	Balance := ctx.Resource.(*resource.Balance)
	return Balance, nil
}

func (h *BalanceHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var Balances []*resource.Balance
	return Balances, nil
}
