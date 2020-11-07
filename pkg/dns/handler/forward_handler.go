package handler

import (
	"fmt"
	"net"

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
	TableForward = restdb.ResourceDBType(&resource.Forward{})
)

type forwardHandler struct {
}

func NewForwardHandler() *forwardHandler {
	return &forwardHandler{}
}

func (h *forwardHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forward := ctx.Resource.(*resource.Forward)
	if err := util.CheckDomainNameValid(forward.Name); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("name %s is invalid", forward.Name))
	}
	if err := h.checkIPValid(forward.Ips); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create forward %s failed: %s", forward.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forward); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create forward %s failed: %s", forward.Name, err.Error()))
	}

	return forward, nil
}

func (h *forwardHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	forwardID := ctx.Resource.(*resource.Forward).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(TableForward,
			map[string]interface{}{restdb.IDField: forwardID}); err != nil {
			return fmt.Errorf("delete forward %s from db failed: %s", forwardID, err.Error())
		}

		return nil
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete forward failed: %s", err.Error()))
	}

	return nil
}

func (h *forwardHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forward := ctx.Resource.(*resource.Forward)
	if err := h.checkIPValid(forward.Ips); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update forward %s failed: %s", forward.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableForward, map[string]interface{}{
			"ips":     forward.Ips,
			"comment": forward.Comment,
		}, map[string]interface{}{restdb.IDField: forward.GetID()}); err != nil {
			return err
		}

		forwardZoneData, err := getForwardZoneData(tx)
		if err != nil {
			return err
		}

		return SendKafkaMessage(forward.ID, kafkaconsumer.UpdateForward,
			&pb.UpdateForwardReq{
				Header:       &pb.DDIRequestHead{Method: "Update", Resource: forward.GetType()},
				ForwardZones: forwardZoneData,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update forward %s failed: %s", forward.Name, err.Error()))
	}

	return forward, nil
}

func (h *forwardHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forwardid := ctx.Resource.(*resource.Forward).GetID()
	var forwards []*resource.Forward
	forward, err := restdb.GetResourceWithID(db.GetDB(), forwardid, &forwards)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get forward %s from db failed: %s",
				forward.(*resource.Forward).Name, err.Error()))
	}

	return forward.(*resource.Forward), nil
}

func (h *forwardHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var forwards []*resource.Forward
	if err := db.GetResources(
		map[string]interface{}{"orderby": "create_time"}, &forwards); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list forward from db failed: %s", err.Error()))
	}

	return forwards, nil
}

func (h *forwardHandler) checkIPValid(ips []string) error {
	for k1, ip := range ips {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("ip %s format not correct", ip)
		}

		for k2 := k1 + 1; k2 < len(ips); k2++ {
			if ip == ips[k2] {
				return fmt.Errorf("ip should not the same:%s", ip)
			}
		}
	}
	return nil
}

func getForwardZoneData(tx restdb.Transaction) ([]*pb.ForwardZone, error) {
	var forwardZones []*resource.ForwardZone
	if err := tx.Fill(map[string]interface{}{"orderby": "create_time"}, &forwardZones); err != nil {
		return nil, fmt.Errorf("list forwardZones from db failed: %s", err.Error())
	}
	var zoneForwards []*resource.ZoneForward
	if err := tx.Fill(map[string]interface{}{}, &zoneForwards); err != nil {
		return nil, fmt.Errorf("list zoneForwards from db failed: %s", err.Error())
	}

	var forwards []*resource.Forward
	if err := tx.Fill(map[string]interface{}{}, &forwards); err != nil {
		return nil, fmt.Errorf("list forwards from db failed: %s", err.Error())
	}

	var forwardZoneData []*pb.ForwardZone
	for _, zone := range forwardZones {
		oneZone := &pb.ForwardZone{Id: zone.ID}
		for _, zf := range zoneForwards {
			if zf.ForwardZone == zone.ID {
				for _, fw := range forwards {
					if fw.ID == zf.Forward {
						oneZone.ForwardIps = append(oneZone.ForwardIps, fw.Ips...)
					}
				}
			}
		}

		for k1 := 0; k1 < len(oneZone.ForwardIps)-1; k1++ {
			for k2, ip2 := range oneZone.ForwardIps {
				if k1 != k2 && oneZone.ForwardIps[k1] == ip2 {
					oneZone.ForwardIps = append(oneZone.ForwardIps[:k1], oneZone.ForwardIps[k1+1:]...)
				}
			}
		}

		forwardZoneData = append(forwardZoneData, oneZone)
	}

	return forwardZoneData, nil
}
