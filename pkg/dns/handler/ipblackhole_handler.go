package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dns/resource"
)

var (
	TableIPBlackHole = restdb.ResourceDBType(&resource.IpBlackHole{})
)

type ipBlackHoleHandler struct{}

func NewIPBlackHoleHandler() *ipBlackHoleHandler {
	return &ipBlackHoleHandler{}
}

func (h *ipBlackHoleHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ipBlackHole := ctx.Resource.(*resource.IpBlackHole)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(ipBlackHole); err != nil {
			return err
		}

		return SendKafkaMessage(ipBlackHole.ID,
			kafkaconsumer.CreateIPBlackHole, &pb.CreateIPBlackHoleReq{
				Header: &pb.DDIRequestHead{Method: "Create", Resource: string(TableIPBlackHole)},
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return ipBlackHole, nil
}

func (h *ipBlackHoleHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	ipBlackHoleID := ctx.Resource.(*resource.IpBlackHole).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(TableIPBlackHole, map[string]interface{}{restdb.IDField: ipBlackHoleID}); err != nil {
			return err
		}

		return SendKafkaMessage(ipBlackHoleID,
			kafkaconsumer.DeleteIPBlackHole, &pb.DeleteIPBlackHoleReq{
				Header: &pb.DDIRequestHead{Method: "Delete", Resource: string(TableIPBlackHole)},
			})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete ipblackhole failed: %s", err.Error()))
	}

	return nil
}

func (h *ipBlackHoleHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ipBlackHole := ctx.Resource.(*resource.IpBlackHole)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableIPBlackHole, map[string]interface{}{
			"acl": ipBlackHole.Acl,
		}, map[string]interface{}{restdb.IDField: ipBlackHole.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(ipBlackHole.ID,
			kafkaconsumer.UpdateIPBlackHole, &pb.UpdateIPBlackHoleReq{
				Header: &pb.DDIRequestHead{Method: "Update", Resource: string(TableIPBlackHole)},
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update ipBlackHole failed: %s", err.Error()))
	}

	return ipBlackHole, nil
}

func (h *ipBlackHoleHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var ipBlackHoles []*resource.IpBlackHole
	if err := db.GetResources(
		map[string]interface{}{"orderby": "create_time"},
		&ipBlackHoles); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list IpBlackHole from db failed: %s", err.Error()))
	}

	return ipBlackHoles, nil
}

func (h *ipBlackHoleHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ipBlackHoleid := ctx.Resource.(*resource.IpBlackHole).GetID()
	var ipBlackHoles []*resource.IpBlackHole
	ipBlackHole, err := restdb.GetResourceWithID(db.GetDB(), ipBlackHoleid, &ipBlackHoles)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get IpBlackHole %s from db failed: %s", ipBlackHoleid, err.Error()))
	}

	return ipBlackHole.(*resource.IpBlackHole), nil
}
