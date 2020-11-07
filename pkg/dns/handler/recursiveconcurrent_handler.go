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
)

var (
	TableRecursiveConcurrent = restdb.ResourceDBType(&resource.RecursiveConcurrent{})
	defaultId                = "1"
)

type recursiveConcurrentHandler struct {
}

func NewRecursiveConcurrentHandler() *recursiveConcurrentHandler {
	return &recursiveConcurrentHandler{}
}

func (h *recursiveConcurrentHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	recursiveConcurrent := ctx.Resource.(*resource.RecursiveConcurrent)
	recursiveConcurrent.SetID(defaultId)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		count, err := tx.Count(TableRecursiveConcurrent, map[string]interface{}{})
		if err != nil {
			return err
		}
		if count < 1 {
			if _, err := tx.Insert(recursiveConcurrent); err != nil {
				return err
			}
		} else {
			if _, err := tx.Update(TableRecursiveConcurrent, map[string]interface{}{
				"recursive_clients": recursiveConcurrent.RecursiveClients,
				"fetches_per_zone":  recursiveConcurrent.FetchesPerZone,
			}, map[string]interface{}{restdb.IDField: defaultId}); err != nil {
				return err
			}
		}

		return SendKafkaMessage(defaultId, kafkaconsumer.UpdateRecursiveConcurrent,
			&pb.UpdateRecurConcuReq{
				Header:           &pb.DDIRequestHead{Method: "Update", Resource: string(TableRecursiveConcurrent)},
				Id:               defaultId,
				IsCreate:         count == 0,
				RecursiveClients: uint32(recursiveConcurrent.RecursiveClients),
				FetchesPerZone:   uint32(recursiveConcurrent.FetchesPerZone),
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update recursiveConcurrent failed: %s", err.Error()))
	}

	return recursiveConcurrent, nil
}

func (h *recursiveConcurrentHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var recursiveConcurrents []*resource.RecursiveConcurrent
	if err := db.GetResources(
		map[string]interface{}{"orderby": "create_time"}, &recursiveConcurrents); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list RecursiveConcurrent from db failed: %s", err.Error()))
	}
	return recursiveConcurrents, nil
}
