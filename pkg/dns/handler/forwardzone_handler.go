package handler

import (
	"fmt"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dns/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var (
	master           = "master"
	forward          = "forward"
	TableForwardZone = restdb.ResourceDBType(&resource.ForwardZone{})
	TableZoneForward = restdb.ResourceDBType(&resource.ZoneForward{})
)

type ForwardZoneHandler struct {
}

func NewForwardZoneHandler() *ForwardZoneHandler {
	return &ForwardZoneHandler{}
}

func (h *ForwardZoneHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forwardZone := ctx.Resource.(*resource.ForwardZone)
	forwardZone.View = forwardZone.GetParent().GetID()
	forwardZone.Name = strings.ToLower(forwardZone.Name)
	if err := util.CheckDomainNameValid(forwardZone.Name); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("name invalid:%s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forwardZone); err != nil {
			return err
		}
		for _, id := range forwardZone.ForwardIDs {
			if _, err := tx.Insert(&resource.ZoneForward{
				ForwardZone: forwardZone.GetID(), Forward: id}); err != nil {
				return err
			}
		}

		forwardIps, err := getForwardZoneIps(forwardZone.ForwardIDs, tx)
		if err != nil {
			return err
		}

		return SendKafkaMessage(forwardZone.ID, kafkaconsumer.CreateForwardZone,
			&pb.CreateForwardZoneReq{
				Header:      &pb.DDIRequestHead{Method: "Create", Resource: forwardZone.GetType()},
				Id:          forwardZone.ID,
				Name:        forwardZone.Name,
				ForwardType: forwardZone.ForwardType,
				ForwardIds:  forwardZone.ForwardIDs,
				ForwardIps:  forwardIps,
				ViewId:      forwardZone.View,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return forwardZone, nil
}

func (h *ForwardZoneHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forwardZone := ctx.Resource.(*resource.ForwardZone)
	if err := util.CheckDomainNameValid(forwardZone.Name); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("name invalid:%s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableForwardZone, map[string]interface{}{
			"forward_type": forwardZone.ForwardType,
			"comment":      forwardZone.Comment,
			"view":         forwardZone.GetParent().GetID(),
		}, map[string]interface{}{restdb.IDField: forwardZone.GetID()}); err != nil {
			return err
		}
		if err := deleteZoneForward(forwardZone, tx); err != nil {
			return err
		}
		if err := createZoneForward(forwardZone, tx); err != nil {
			return err
		}

		forwardIps, err := getForwardZoneIps(forwardZone.ForwardIDs, tx)
		if err != nil {
			return err
		}

		return SendKafkaMessage(forwardZone.ID, kafkaconsumer.UpdateForwardZone,
			&pb.UpdateForwardZoneReq{
				Header:      &pb.DDIRequestHead{Method: "Update", Resource: forwardZone.GetType()},
				Id:          forwardZone.ID,
				ForwardType: forwardZone.ForwardType,
				ForwardIds:  forwardZone.ForwardIDs,
				ForwardIps:  forwardIps,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("Update forwardZone %s failed:%s", forwardZone.Name, err.Error()))
	}

	return forwardZone, nil
}

func deleteZoneForward(forwardZone *resource.ForwardZone, tx restdb.Transaction) error {
	if _, err := tx.Delete(TableZoneForward,
		map[string]interface{}{"forward_zone": forwardZone.GetID()}); err != nil {
		return err
	}
	return nil
}

func createZoneForward(forwardZone *resource.ForwardZone, tx restdb.Transaction) error {
	for _, forwardid := range forwardZone.ForwardIDs {
		if _, err := tx.Insert(&resource.ZoneForward{
			ForwardZone: forwardZone.GetID(), Forward: forwardid}); err != nil {
			return err
		}
	}
	return nil
}

func getForwardZoneIps(forwardIds []string, tx restdb.Transaction) ([]string, error) {
	var forwardList []*resource.Forward
	sql := fmt.Sprintf(`select * from gr_forward where id in ('%s')`,
		strings.Join(forwardIds, "','"))
	if err := tx.FillEx(&forwardList, sql); err != nil {
		return nil, fmt.Errorf("getForwardZoneIps get forward ids:%s from db failed:%s", forwardIds, err.Error())
	}
	if len(forwardList) == 0 {
		return nil, fmt.Errorf("getForwardZoneIps get forward ids:%s from db failed:len(forwards)==0", forwardIds)
	}

	var forwardIps []string
	for _, forward := range forwardList {
		forwardIps = append(forwardIps, forward.Ips...)
	}

	return forwardIps, nil
}

func (h *ForwardZoneHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	forwardZone := ctx.Resource.(*resource.ForwardZone)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(TableForwardZone,
			map[string]interface{}{restdb.IDField: forwardZone.GetID()}); err != nil {
			return fmt.Errorf("delete zone %s to db failed: %s", forwardZone.GetID(), err.Error())
		}

		return SendKafkaMessage(forwardZone.GetID(), kafkaconsumer.DeleteForwardZone,
			&pb.DeleteForwardZoneReq{
				Header: &pb.DDIRequestHead{Method: "Delete", Resource: forwardZone.GetType()},
				Id:     forwardZone.GetID()})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete zone failed: %s", err.Error()))
	}

	return nil
}

func (h *ForwardZoneHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var forwardZones []*resource.ForwardZone
	if err := db.GetResources(map[string]interface{}{
		"orderby": "create_time",
		"view":    ctx.Resource.GetParent().GetID()}, &forwardZones); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list forwardZones from db failed: %s", err.Error()))
	}
	for i, zone := range forwardZones {
		var zoneForwards []*resource.ZoneForward
		if err := db.GetResources(map[string]interface{}{
			"forward_zone": zone.GetID()}, &zoneForwards); err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("list zoneforward from db failed: %s", err.Error()))
		}
		for _, zoneforward := range zoneForwards {
			var forwards []*resource.Forward
			forward, err := restdb.GetResourceWithID(db.GetDB(), zoneforward.Forward, &forwards)
			if err != nil {
				return nil, resterror.NewAPIError(resterror.ServerError,
					fmt.Sprintf("get forward from db failed: %s", err.Error()))
			}
			forwardZones[i].Forwards = append(forwardZones[i].Forwards, forward.(*resource.Forward))
		}
	}

	return forwardZones, nil
}

func (h *ForwardZoneHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	forwardZoneID := ctx.Resource.(*resource.ForwardZone).GetID()
	var forwardZones []*resource.ForwardZone
	forwardZone, err := restdb.GetResourceWithID(db.GetDB(), forwardZoneID, &forwardZones)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get zone %s from db failed: %s", forwardZoneID, err.Error()))
	}
	var zoneForwards []*resource.ZoneForward
	if err := db.GetResources(map[string]interface{}{
		"forward_zone": forwardZoneID}, &zoneForwards); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list zoneforward from db failed: %s", err.Error()))
	}
	for _, zoneforward := range zoneForwards {
		var forwards []*resource.Forward
		forward, err := restdb.GetResourceWithID(db.GetDB(), zoneforward.Forward, &forwards)
		if err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("get forward from db failed: %s", err.Error()))
		}
		forwardZone.(*resource.ForwardZone).Forwards = append(
			forwardZone.(*resource.ForwardZone).Forwards, forward.(*resource.Forward))
	}

	return forwardZone.(*resource.ForwardZone), nil
}
