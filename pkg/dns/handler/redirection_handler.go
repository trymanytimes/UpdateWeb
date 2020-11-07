package handler

import (
	"fmt"
	"strings"

	"github.com/zdnscloud/g53"
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
	TableRedirection = restdb.ResourceDBType(&resource.Redirection{})
	localZoneType    = "localzone"
	nxDomainType     = "nxdomain"
)

type redirectionHandler struct{}

func NewRedirectionHandler() *redirectionHandler {
	return &redirectionHandler{}
}

func (h *redirectionHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	redirection := ctx.Resource.(*resource.Redirection)
	redirection.Name = strings.ToLower(redirection.Name)
	if err := h.checkRedirection(redirection); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create redirection %s failed: %s", redirection.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, err := tx.Exists(TableUrlRedirect, map[string]interface{}{"domain": redirection.Name})
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("the %s has exist in url redirect", redirection.Name)
		}
		redirection.View = redirection.GetParent().GetID()
		if _, err := tx.Insert(redirection); err != nil {
			return err
		}

		return SendKafkaMessage(redirection.ID, kafkaconsumer.CreateRedirection,
			&pb.CreateRedirectionReq{
				Header:       &pb.DDIRequestHead{Method: "Create", Resource: redirection.GetType()},
				Id:           redirection.ID,
				Name:         redirection.Name,
				Ttl:          uint32(redirection.Ttl),
				DataType:     redirection.DataType,
				RedirectType: redirection.RedirectType,
				RData:        redirection.Rdata,
				ViewId:       redirection.View,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create redirection failed: %s", err.Error()))
	}

	return redirection, nil
}

func (h *redirectionHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	redirectionId := ctx.Resource.(*resource.Redirection).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		redirectionRes, err := tx.Get(TableRedirection,
			map[string]interface{}{restdb.IDField: redirectionId})
		if err != nil {
			return err
		}
		redirection := redirectionRes.([]*resource.Redirection)[0]
		_, err = tx.Delete(TableRedirection,
			map[string]interface{}{restdb.IDField: redirectionId})
		if err != nil {
			return err
		}

		return SendKafkaMessage(redirection.GetID(), kafkaconsumer.DeleteRedirection,
			&pb.DeleteRedirectionReq{
				Header:       &pb.DDIRequestHead{Method: "Delete", Resource: redirection.GetType()},
				Id:           redirection.GetID(),
				View:         redirection.View,
				RedirectType: redirection.RedirectType,
			})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create redirection failed: %s", err.Error()))
	}

	return nil
}

func (h *redirectionHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	redirection := ctx.Resource.(*resource.Redirection)
	redirection.Name = strings.ToLower(redirection.Name)
	if err := h.checkRedirection(redirection); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update redirection %s failed: %s", redirection.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		redirections, err := tx.Get(TableRedirection,
			map[string]interface{}{restdb.IDField: redirection.ID})
		if err != nil {
			return err
		}
		oldRedirection := redirections.([]*resource.Redirection)[0]
		redirectTypeChanged := false
		if oldRedirection.RedirectType != redirection.RedirectType {
			redirectTypeChanged = true
		}

		exist, err := tx.Exists(TableUrlRedirect,
			map[string]interface{}{"domain": redirection.Name})
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("the %s has exist in url redirect", redirection.Name)
		}

		if _, err := tx.Update(TableRedirection, map[string]interface{}{
			"name":          redirection.Name,
			"ttl":           redirection.Ttl,
			"data_type":     redirection.DataType,
			"redirect_type": redirection.RedirectType,
			"rdata":         redirection.Rdata,
		}, map[string]interface{}{restdb.IDField: redirection.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(redirection.ID, kafkaconsumer.UpdateRedirection,
			&pb.UpdateRedirectionReq{
				Header:              &pb.DDIRequestHead{Method: "Update", Resource: redirection.GetType()},
				Id:                  redirection.ID,
				DataType:            redirection.DataType,
				RedirectType:        redirection.RedirectType,
				RData:               redirection.Rdata,
				Ttl:                 uint32(redirection.Ttl),
				RedirectTypeChanged: redirectTypeChanged,
				View:                redirection.GetParent().GetID(),
				Name:                redirection.Name,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update redirection %s failed: %s", redirection.Name, err.Error()))
	}

	return redirection, nil
}

func (h *redirectionHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var redirections []*resource.Redirection
	if err := db.GetResources(map[string]interface{}{
		"view":    ctx.Resource.GetParent().GetID(),
		"orderby": "create_time"}, &redirections); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list Redirection from db failed: %s", err.Error()))
	}

	return redirections, nil
}

func (h *redirectionHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	redirectionid := ctx.Resource.(*resource.Redirection).GetID()
	var redirections []*resource.Redirection
	redirection, err := restdb.GetResourceWithID(db.GetDB(), redirectionid, &redirections)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get redirection %s from db failed: %s", redirectionid, err.Error()))
	}

	return redirection.(*resource.Redirection), nil
}

func (h *redirectionHandler) checkRedirection(rr *resource.Redirection) error {
	if err := util.CheckDomainNameValid(rr.Name); err != nil {
		return err
	}

	rrType, err := g53.TypeFromString(rr.DataType)
	if err != nil {
		return fmt.Errorf("invalid rrType:%s", err.Error())
	}

	_, err = g53.RdataFromString(rrType, rr.Rdata)
	if err != nil {
		return fmt.Errorf("invalid rData:%s", err.Error())
	}

	return nil
}
