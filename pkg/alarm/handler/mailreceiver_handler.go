package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

type MailReceiverHandler struct{}

func NewMailReceiverHandler() *MailReceiverHandler {
	return &MailReceiverHandler{}
}

func (h *MailReceiverHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	receiver := ctx.Resource.(*resource.MailReceiver)
	receiver.SetID(receiver.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(ctx.Resource)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("add mail receiver %s faield: %s", receiver.Name, err.Error()))
	}

	return ctx.Resource, nil
}

func (h *MailReceiverHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var receivers []*resource.MailReceiver
	if err := db.GetResources(nil, &receivers); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list mail receiver faield: %s", err.Error()))
	}

	return receivers, nil
}

func (h *MailReceiverHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	receiverID := ctx.Resource.(*resource.MailReceiver).GetID()
	var receivers []*resource.MailReceiver
	receiver, err := restdb.GetResourceWithID(db.GetDB(), receiverID, &receivers)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get mail receiver %s faield: %s", receiverID, err.Error()))
	}

	return receiver.(restresource.Resource), nil
}

func (h *MailReceiverHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	receiver := ctx.Resource.(*resource.MailReceiver)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableMailReceiver, map[string]interface{}{
			"address": receiver.Address,
		}, map[string]interface{}{restdb.IDField: receiver.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update mail receiver %s faield: %s", receiver.GetID(), err.Error()))
	}

	return receiver, nil
}

func (h *MailReceiverHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	receiverID := ctx.Resource.(*resource.MailReceiver).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TableMailReceiver, map[string]interface{}{restdb.IDField: receiverID})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete mail receiver %s failed: %s", receiverID, err.Error()))
	}

	return nil
}
