package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

const MailSenderUID = "alarmmailsender"

type MailSenderHandler struct{}

func NewMailSenderHandler() *MailSenderHandler {
	return &MailSenderHandler{}
}

func (h *MailSenderHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sender := ctx.Resource.(*resource.MailSender)
	sender.SetID(MailSenderUID)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(sender)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("add mail sender faield: %s", err.Error()))
	}

	return ctx.Resource, nil
}

func (h *MailSenderHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var senders []*resource.MailSender
	if err := db.GetResources(nil, &senders); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list mail sender faield: %s", err.Error()))
	}

	return senders, nil
}

func (h *MailSenderHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	senderID := ctx.Resource.(*resource.MailSender).GetID()
	var senders []*resource.MailSender
	sender, err := restdb.GetResourceWithID(db.GetDB(), senderID, &senders)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get mail sender %s faield: %s", senderID, err.Error()))
	}

	return sender.(restresource.Resource), nil
}

func (h *MailSenderHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sender := ctx.Resource.(*resource.MailSender)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableMailSender, map[string]interface{}{
			"host":     sender.Host,
			"port":     sender.Port,
			"username": sender.Username,
			"password": sender.Password,
			"enabled":  sender.Enabled,
		}, map[string]interface{}{restdb.IDField: sender.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update mail sender faield: %s", err.Error()))
	}

	return sender, nil
}
