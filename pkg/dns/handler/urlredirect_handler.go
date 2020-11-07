package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/dns/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

var (
	TableUrlRedirect = restdb.ResourceDBType(&resource.UrlRedirect{})
)

type UrlRedirectHandler struct{}

func NewUrlRedirectHandler() *UrlRedirectHandler {
	return &UrlRedirectHandler{}
}

func (h *UrlRedirectHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	urlRedirect := ctx.Resource.(*resource.UrlRedirect)
	urlRedirect.View = urlRedirect.GetParent().GetID()
	if err := util.CheckDomainNameValid(urlRedirect.Domain); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create url redirect %s failed: %s", urlRedirect.Domain, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, err := tx.Exists(TableRedirection, map[string]interface{}{"name": urlRedirect.Domain})
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("the %s has exist in redirection", urlRedirect.Domain)
		}
		if _, err := tx.Insert(urlRedirect); err != nil {
			return err
		}

		return SendKafkaMessage(urlRedirect.ID, kafkaconsumer.CreateUrlRedirect,
			&pb.CreateUrlRedirectReq{
				Header: &pb.DDIRequestHead{Method: "Create", Resource: urlRedirect.GetType()},
				Id:     urlRedirect.ID,
				Domain: urlRedirect.Domain,
				Url:    urlRedirect.Url,
				ViewId: urlRedirect.View,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create url redirect %s failed: %s", urlRedirect.Domain, err.Error()))
	}

	return urlRedirect, nil
}

func (h *UrlRedirectHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	urlRedirect := ctx.Resource.(*resource.UrlRedirect)

	if err := util.CheckDomainNameValid(urlRedirect.Domain); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update url redirect %s failed: %s", urlRedirect.Domain, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, err := tx.Exists(TableRedirection, map[string]interface{}{"name": urlRedirect.Domain})
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("the %s has exist in redirection", urlRedirect.Domain)
		}

		if _, err := tx.Update(TableUrlRedirect, map[string]interface{}{
			"domain": urlRedirect.Domain,
			"url":    urlRedirect.Url,
		}, map[string]interface{}{restdb.IDField: urlRedirect.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(urlRedirect.ID, kafkaconsumer.UpdateUrlRedirect,
			&pb.UpdateUrlRedirectReq{
				Header: &pb.DDIRequestHead{Method: "Update", Resource: urlRedirect.GetType()},
				Id:     urlRedirect.ID,
				Domain: urlRedirect.Domain,
				Url:    urlRedirect.Url,
				View:   urlRedirect.GetParent().GetID(),
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update url redirect %s failed: %s", urlRedirect.Domain, err.Error()))
	}

	return urlRedirect, nil
}

func (h *UrlRedirectHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	urlRedirectId := ctx.Resource.(*resource.UrlRedirect).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		urlRedirectRes, err := tx.Get(TableUrlRedirect,
			map[string]interface{}{restdb.IDField: urlRedirectId})
		if err != nil {
			return err
		}
		urlRedirect := urlRedirectRes.([]*resource.UrlRedirect)[0]
		if _, err = tx.Delete(TableUrlRedirect,
			map[string]interface{}{restdb.IDField: urlRedirectId}); err != nil {
			return err
		}

		return SendKafkaMessage(urlRedirectId, kafkaconsumer.DeleteUrlRedirect,
			&pb.DeleteUrlRedirectReq{
				Header: &pb.DDIRequestHead{Method: "Delete",
					Resource: ctx.Resource.(*resource.UrlRedirect).GetType()},
				Id:   urlRedirectId,
				View: urlRedirect.View,
			})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("Delete url redirect %s failed: %s", urlRedirectId, err.Error()))
	}

	return nil
}

func (h *UrlRedirectHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var urlRedirects []*resource.UrlRedirect
	if err := db.GetResources(map[string]interface{}{
		"view":    ctx.Resource.GetParent().GetID(),
		"orderby": "create_time"}, &urlRedirects); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get urlRedirects from db failed: %s", err.Error()))
	}

	return urlRedirects, nil
}

func (h *UrlRedirectHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	urlRedirectId := ctx.Resource.(*resource.UrlRedirect).GetID()
	var urlRedirects []*resource.UrlRedirect
	urlRedirect, err := restdb.GetResourceWithID(db.GetDB(), urlRedirectId, &urlRedirects)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get urlRedirect %s from db failed: %s", urlRedirectId, err.Error()))
	}

	return urlRedirect.(*resource.UrlRedirect), nil
}
