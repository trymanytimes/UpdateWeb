package handler

import (
	"fmt"
	"net"
	"strings"

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
	zoneFileSuffix = ".zone"
	TableZone      = restdb.ResourceDBType(&resource.Zone{})
	RoleMain       = "main"
	RoleBackup     = "backup"
)

type ZoneHandler struct{}

func NewZoneHandler() *ZoneHandler {
	return &ZoneHandler{}
}

func (h *ZoneHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	zone := ctx.Resource.(*resource.Zone)
	zone.Name = strings.ToLower(zone.Name)
	zone.View = zone.GetParent().GetID()
	zone.RrsRole = RoleMain

	if err := h.checkAndConvertZoneName(zone); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("name %s is invalid", zone.Name))
	}

	zone.SetZoneFile(zoneFileSuffix)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(zone); err != nil {
			return err
		}

		return SendKafkaMessage(zone.ID, kafkaconsumer.CreateZone,
			&pb.CreateZoneReq{
				Header:       &pb.DDIRequestHead{Method: "Create", Resource: zone.GetType()},
				ZoneId:       zone.ID,
				ViewId:       zone.View,
				ZoneName:     zone.Name,
				ZoneFileName: zone.ZoneFile,
				Ttl:          uint32(zone.Ttl),
				RrsRole:      zone.RrsRole,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create zone failed: %s", err.Error()))
	}

	return zone, nil
}

func (h *ZoneHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	zone := ctx.Resource.(*resource.Zone)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zoneRes, err := tx.Get(TableZone, map[string]interface{}{restdb.IDField: zone.ID})
		if err != nil {
			return err
		}
		oldZone := zoneRes.([]*resource.Zone)[0]

		if _, err := tx.Update(TableZone, map[string]interface{}{
			"ttl":     zone.Ttl,
			"comment": zone.Comment,
		}, map[string]interface{}{restdb.IDField: zone.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(zone.ID, kafkaconsumer.UpdateZone,
			&pb.UpdateZoneReq{
				Header:       &pb.DDIRequestHead{Method: "Update", Resource: zone.GetType()},
				Id:           zone.ID,
				Ttl:          uint32(zone.Ttl),
				ZoneFileName: oldZone.ZoneFile,
				Name:         oldZone.Name,
				View:         oldZone.View,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update zone %s failed: %s", zone.ID, err.Error()))
	}

	return zone, nil
}

func (h *ZoneHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	zoneId := ctx.Resource.(*resource.Zone).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zoneRes, err := tx.Get(TableZone, map[string]interface{}{restdb.IDField: zoneId})
		if err != nil {
			return err
		}
		zone := zoneRes.([]*resource.Zone)[0]
		c, err := tx.Delete(TableZone, map[string]interface{}{restdb.IDField: zoneId})
		if err != nil {
			return err
		}
		if c > 0 {
			if _, err := tx.Delete(TableRR,
				map[string]interface{}{"zone": zone.GetID(), "view": zone.View}); err != nil {
				return err
			}
		}

		return SendKafkaMessage(zone.GetID(), kafkaconsumer.DeleteZone,
			&pb.DeleteZoneReq{
				Header: &pb.DDIRequestHead{Method: "Delete", Resource: zone.GetType()},
				Id:     zone.GetID(),
				Name:   zone.Name,
				View:   zone.View,
			})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete zone %s failed: %s", zoneId, err.Error()))
	}

	return nil
}

func (h *ZoneHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var zones []*resource.Zone
	if err := db.GetResources(map[string]interface{}{
		"view":    ctx.Resource.GetParent().GetID(),
		"orderby": "create_time"}, &zones); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list zones from db failed: %s", err.Error()))
	}
	if _, err := GetRRCount(zones); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list zones's rr count from db failed: %s", err.Error()))
	}
	return zones, nil
}

func GetRRCount(zones []*resource.Zone) (int, error) {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	var totalCount int
	for i, zone := range zones {
		count, err := tx.Count(TableRR, map[string]interface{}{
			"zone": zone.GetID(),
			"view": zone.View,
		})
		if err != nil {
			return 0, err
		}
		zones[i].RRSize = int(count)
		totalCount += int(count)
	}
	return totalCount, nil
}

func (h *ZoneHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	zoneId := ctx.Resource.(*resource.Zone).GetID()
	var zones []*resource.Zone
	zone, err := restdb.GetResourceWithID(db.GetDB(), zoneId, &zones)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get zone %s from db failed: %s", zoneId, err.Error()))
	}

	return zone.(*resource.Zone), nil
}

func (h *ZoneHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ChangingRRs:
		return h.changeRRs(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *ZoneHandler) changeRRs(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	enableRole, ok := ctx.Resource.GetAction().Input.(*resource.EnableRole)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action changingRRs %s input invalid", ctx.Resource.GetAction().Name))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zones, err := tx.Get(TableZone,
			map[string]interface{}{restdb.IDField: ctx.Resource.GetID()})
		if err != nil {
			return err
		}
		views, err := tx.Get(TableView,
			map[string]interface{}{restdb.IDField: ctx.Resource.GetParent().GetID()})
		if err != nil {
			return err
		}
		if len(zones.([]*resource.Zone)) == 0 || len(views.([]*resource.View)) == 0 {
			return fmt.Errorf("get zones or views from db failed:zero length")
		}
		zone := zones.([]*resource.Zone)[0]
		view := views.([]*resource.View)[0]

		if _, err := tx.Update(TableZone, map[string]interface{}{
			"rrs_role": enableRole.Role,
		}, map[string]interface{}{restdb.IDField: ctx.Resource.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(ctx.Resource.GetID(), kafkaconsumer.UpdateRRsByZone,
			&pb.UpdateRRsByZoneReq{
				Header:     &pb.DDIRequestHead{Method: "Update", Resource: string(TableZone)},
				ZoneId:     ctx.Resource.GetID(),
				ZoneName:   zone.Name,
				ViewKey:    view.Key,
				ViewName:   view.Name,
				OldRrsRole: zone.RrsRole,
				NewRrsRole: enableRole.Role,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s failed:%s", ctx.Resource.GetAction().Name, err.Error()))
	}

	return &resource.OperResult{Result: true}, nil
}

func (h *ZoneHandler) checkAndConvertZoneName(zone *resource.Zone) error {
	if zone.IsArpa {
		ip, ipnet, err := net.ParseCIDR(zone.Name)
		if err != nil {
			return err
		}
		maskSize, _ := ipnet.Mask.Size()
		if ip.To4() != nil {
			parts := strings.Split(ip.String(), ".")
			zone.Name = ""
			for i := maskSize/8 - 1; i >= 0; i-- {
				zone.Name += parts[i] + "."
			}
			zone.Name += "in-addr.arpa"
		} else {
			zone.Name = ""
			if zone.Name, err = getIPv6ReverserString(ip, maskSize); err != nil {
				return err
			}
			zone.Name += "ip6.arpa"
		}

		return nil
	}

	return util.CheckZoneNameValid(zone.Name)
}

func getIPv6ReverserString(ip net.IP, maskSize int) (string, error) {
	fullAddr := ip.To16()
	if fullAddr == nil {
		return "", fmt.Errorf("ipv6 address format err")
	}
	var name string
	for i := (maskSize - 1) / 4; i >= 0; i-- {
		name += fmt.Sprintf("%x.", fullAddr[i/2]>>((i+1)%2*4)&0xF)
	}
	return name, nil
}
