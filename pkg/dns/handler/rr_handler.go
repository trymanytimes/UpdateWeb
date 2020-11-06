package handler

import (
	"fmt"
	"net"
	"strings"

	"github.com/zdnscloud/g53"
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
	TableRR = restdb.ResourceDBType(&resource.Rr{})
)

type rrHandler struct{}

func NewRRHandler() *rrHandler {
	return &rrHandler{}
}

func (h *rrHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rr := ctx.Resource.(*resource.Rr)
	rr.Zone = rr.GetParent().GetID()
	rr.View = rr.GetParent().GetParent().GetID()
	rr.Name = strings.ToLower(rr.Name)
	if rr.Rdata == rr.RdataBackup {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("%s and %s can not be the same", rr.Rdata, rr.RdataBackup))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zones, err := tx.Get(TableZone, map[string]interface{}{restdb.IDField: rr.Zone})
		if err != nil {
			return err
		}
		views, err := tx.Get(TableView, map[string]interface{}{restdb.IDField: rr.View})
		if err != nil {
			return err
		}
		zone := zones.([]*resource.Zone)[0]
		view := views.([]*resource.View)[0]
		if rr.Ttl == 0 {
			rr.Ttl = zone.Ttl
		}

		if err := h.checkRR(rr, zone.Name, tx); err != nil {
			return err
		}

		if err := h.CheckRdataExists(rr, tx); err != nil {
			return err
		}

		if _, err := tx.Insert(rr); err != nil {
			return err
		}

		return SendKafkaMessage(rr.ID, kafkaconsumer.CreateRR,
			&pb.CreateRRReq{
				Header:      &pb.DDIRequestHead{Method: "Create", Resource: rr.GetType()},
				Id:          rr.ID,
				Name:        rr.Name,
				ZoneId:      rr.Zone,
				ViewId:      rr.View,
				DataType:    rr.DataType,
				RData:       rr.Rdata,
				BackupRData: rr.RdataBackup,
				Ttl:         uint32(rr.Ttl),
				ViewKey:     view.Key,
				ZoneName:    zone.Name,
				ZoneRrsRole: zone.RrsRole,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create rr %s failed: %s", rr.Name, err.Error()))
	}

	return rr, nil
}

func checkMX(rr *resource.Rr, zoneName string, tx restdb.Transaction) error {
	if rr.DataType != resource.DataTypeMX {
		return nil
	}

	rdataDomain := strings.Split(rr.Rdata, " ")
	if len(rdataDomain) < 2 {
		return fmt.Errorf("bad radta")
	}

	if rdataDomain[1] == zoneName {
		return fmt.Errorf("radta should be zone name")
	}

	ip := net.ParseIP(rdataDomain[1])
	if ip != nil && ip.To4() == nil {
		return fmt.Errorf("rdata should not be ipv6")
	}

	if !strings.HasSuffix(rdataDomain[1], "."+zoneName) {
		return nil
	}

	rdata := strings.Split(rdataDomain[1], "."+zoneName)[0]
	var rrList []*resource.Rr
	if err := tx.Fill(map[string]interface{}{"name": rdata, "zone": rr.Zone}, &rrList); err != nil {
		return err
	}
	if len(rrList) == 0 {
		return fmt.Errorf("radata:%s should exist before mx created", rr.Rdata)
	}

	rdataRR := rrList[0]
	if rdataRR.DataType == resource.DataTypeCNAME {
		return fmt.Errorf("the rdata:%s of mx shouldn't be cname", rr.Rdata)
	}

	return nil
}

func checkNS(rr *resource.Rr, zoneName string, tx restdb.Transaction) error {
	if rr.DataType != resource.DataTypeNS {
		return nil
	}

	ip := net.ParseIP(rr.Rdata)
	if ip != nil && ip.To4() == nil {
		return fmt.Errorf("ns should not be ipv6")
	}

	if !strings.HasSuffix(rr.Rdata, "."+zoneName) {
		return nil
	}

	rdata := strings.Split(rr.Rdata, "."+zoneName)[0]
	var rrList []*resource.Rr
	if err := tx.Fill(map[string]interface{}{"name": rdata, "zone": rr.Zone}, &rrList); err != nil {
		return err
	}
	if len(rrList) == 0 {
		return fmt.Errorf("radata:%s should exist before ns created", rr.Rdata)
	}

	rdataRR := rrList[0]
	if rdataRR.DataType == resource.DataTypeCNAME {
		return fmt.Errorf("the rdata:%s of ns shouldn't be cname", rr.Rdata)
	}

	return nil
}

func checkNSDelete(rr *resource.Rr, zoneName string, tx restdb.Transaction) error {
	if rr.DataType == resource.DataTypeNS {
		return nil
	}

	var rrList []*resource.Rr
	if err := tx.Fill(map[string]interface{}{
		"rdata": rr.Name + "." + zoneName,
		"zone":  rr.Zone}, &rrList); err != nil {
		return err
	}

	if len(rrList) == 0 {
		return nil
	}

	for _, rdataRR := range rrList {
		if rdataRR.View == rr.View && rdataRR.DataType == resource.DataTypeNS {
			return fmt.Errorf("this rr has been binded by NS:%s", rdataRR.Name)
		}
	}

	return nil
}

//TODO switch zoneRole move from agent to contoller and agent don't save RdataBackup
func (h *rrHandler) GetActiveRdata(ctx *restresource.Context, rr *resource.Rr) (string, error) {
	var zones []*resource.Zone
	zone, err := restdb.GetResourceWithID(db.GetDB(), ctx.Resource.GetParent().GetID(), &zones)
	if err != nil {
		return "", fmt.Errorf("get zone %s from db failed: %s", ctx.Resource.GetParent().GetID(), err.Error())
	}
	var rdata string
	if zone.(*resource.Zone).RrsRole == RoleMain {
		rdata = rr.Rdata
	} else {
		if rr.RdataBackup != "" {
			rdata = rr.RdataBackup
		} else {
			rdata = rr.Rdata
		}
	}
	return rdata, nil
}

func (h *rrHandler) CheckRdataExists(rr *resource.Rr, tx restdb.Transaction) error {
	if rr.RdataBackup != "" {
		count, err := tx.Exec("select * from gr_rr where id != $1 and  view = $2 and name = $3 and data_type = $4 and (rdata in ($5,$6) or rdata_backup in ($7,$8)) and zone = $9",
			rr.GetID(), rr.View, rr.Name, rr.DataType, rr.Rdata, rr.RdataBackup, rr.Rdata, rr.RdataBackup, rr.Zone)
		if err != nil {
			return fmt.Errorf("query rr %s's backup rdata %s to db failed: %s",
				rr.Name, rr.RdataBackup, err.Error())
		}
		if count > 0 {
			return fmt.Errorf("create/update rr %s fail, had existd rdata %s or backup rdata %s",
				rr.Name, rr.Rdata, rr.RdataBackup)
		}
		if rr.Rdata == rr.RdataBackup {
			return fmt.Errorf("create/update rr %s fail, rdata %s and backup rdata %s is the same",
				rr.Name, rr.Rdata, rr.RdataBackup)
		}
	} else {
		count, err := tx.Exec("select * from gr_rr where id != $1 and  view = $2 and name = $3 and data_type = $4 and (rdata = $5 or rdata_backup =$6) and zone = $7",
			rr.GetID(), rr.View, rr.Name, rr.DataType, rr.Rdata, rr.Rdata, rr.Zone)
		if err != nil {
			return fmt.Errorf("query rr %s's backup rdata %s to db failed: %s",
				rr.Name, rr.RdataBackup, err.Error())
		}
		if count > 0 {
			return fmt.Errorf("create/update rr %s fail, rdata %s had existd",
				rr.Name, rr.Rdata)
		}
	}
	return nil
}

func (h *rrHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rr := ctx.Resource.(*resource.Rr)
	rr.Name = strings.ToLower(rr.Name)
	rr.View = ctx.Resource.GetParent().GetParent().GetID()
	rr.Zone = ctx.Resource.GetParent().GetID()

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := h.CheckRdataExists(rr, tx); err != nil {
			return err
		}
		zones, err := tx.Get(TableZone, map[string]interface{}{restdb.IDField: rr.Zone})
		if err != nil {
			return err
		}
		views, err := tx.Get(TableView, map[string]interface{}{restdb.IDField: rr.View})
		if err != nil {
			return err
		}
		if len(zones.([]*resource.Zone)) == 0 || len(views.([]*resource.View)) == 0 {
			return fmt.Errorf("get zones or views from db failed:zero length")
		}
		zone := zones.([]*resource.Zone)[0]
		view := views.([]*resource.View)[0]
		if rr.Ttl == 0 {
			rr.Ttl = zone.Ttl
		}

		if err := h.checkRR(rr, zone.Name, tx); err != nil {
			return err
		}

		if _, err := tx.Update(TableRR, map[string]interface{}{
			"name":         rr.Name,
			"data_type":    rr.DataType,
			"ttl":          rr.Ttl,
			"rdata":        rr.Rdata,
			"rdata_backup": rr.RdataBackup,
			"zone":         rr.Zone,
			"view":         rr.View,
		}, map[string]interface{}{restdb.IDField: rr.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(rr.ID, kafkaconsumer.UpdateRR,
			&pb.UpdateRRReq{
				Header:      &pb.DDIRequestHead{Method: "Update", Resource: rr.GetType()},
				Id:          rr.ID,
				DataType:    rr.DataType,
				RData:       rr.Rdata,
				BackupRData: rr.RdataBackup,
				Ttl:         uint32(rr.Ttl),
				ViewName:    view.Name,
				ViewKey:     view.Key,
				ZoneName:    zone.Name,
				ZoneRrsRole: zone.RrsRole,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update rr %s failed: %s", rr.Name, err.Error()))
	}

	return rr, nil
}

func (h *rrHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	rrId := ctx.Resource.(*resource.Rr).GetID()

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zones, err := tx.Get(TableZone,
			map[string]interface{}{restdb.IDField: ctx.Resource.GetParent().GetID()})
		if err != nil {
			return err
		}
		views, err := tx.Get(TableView,
			map[string]interface{}{restdb.IDField: ctx.Resource.GetParent().GetParent().GetID()})
		if err != nil {
			return err
		}
		rrRes, err := tx.Get(TableRR, map[string]interface{}{restdb.IDField: rrId})
		if err != nil {
			return err
		}
		zone := zones.([]*resource.Zone)[0]
		view := views.([]*resource.View)[0]
		rr := rrRes.([]*resource.Rr)[0]

		if err := checkNSDelete(rr, zone.Name, tx); err != nil {
			return err
		}
		if _, err := tx.Delete(TableRR,
			map[string]interface{}{restdb.IDField: rrId}); err != nil {
			return err
		}

		return SendKafkaMessage(rrId, kafkaconsumer.DeleteRR,
			&pb.DeleteRRReq{
				Header:      &pb.DDIRequestHead{Method: "Delete", Resource: rr.GetType()},
				Id:          rrId,
				Name:        rr.Name,
				DataType:    rr.DataType,
				Ttl:         uint32(rr.Ttl),
				RData:       rr.Rdata,
				BackupRData: rr.RdataBackup,
				ZoneId:      rr.Zone,
				ViewName:    view.Name,
				ViewKey:     view.Key,
				ZoneName:    zone.Name,
				ZoneRrsRole: zone.RrsRole,
			})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete rr %s failed: %s", rrId, err.Error()))
	}

	return nil
}

func (h *rrHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var rrs []*resource.Rr
	zoneid := ctx.Resource.(*resource.Rr).GetParent().GetID()
	viewid := ctx.Resource.(*resource.Rr).GetParent().GetParent().GetID()
	if err := db.GetResources(map[string]interface{}{
		"zone":    zoneid,
		"view":    viewid,
		"orderby": "create_time"}, &rrs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list rr from db failed: %s", err.Error()))
	}
	var zones []*resource.Zone
	zone, err := restdb.GetResourceWithID(db.GetDB(), ctx.Resource.GetParent().GetID(), &zones)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get zones %s from db failed: %s",
				ctx.Resource.GetParent().GetID(), err.Error()))
	}
	for _, rr := range rrs {
		h.SetActiveRdata(zone, rr)
	}
	return rrs, nil
}

func (h *rrHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rrid := ctx.Resource.(*resource.Rr).GetID()
	var rrs []*resource.Rr
	rr, err := restdb.GetResourceWithID(db.GetDB(), rrid, &rrs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get rr %s from db failed: %s", rrid, err.Error()))
	}

	var zones []*resource.Zone
	zone, err := restdb.GetResourceWithID(db.GetDB(), ctx.Resource.GetParent().GetID(), &zones)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get zones %s from db failed: %s", ctx.Resource.GetParent().GetID(), err.Error()))
	}
	h.SetActiveRdata(zone, rr)

	return rr.(*resource.Rr), nil
}

func (h *rrHandler) SetActiveRdata(zone interface{}, rr interface{}) {
	if zone.(*resource.Zone).RrsRole == RoleMain {
		rr.(*resource.Rr).ActiveRdata = rr.(*resource.Rr).Rdata
	} else if zone.(*resource.Zone).RrsRole == RoleBackup {
		if rr.(*resource.Rr).RdataBackup != "" {
			rr.(*resource.Rr).ActiveRdata = rr.(*resource.Rr).RdataBackup
		} else {
			rr.(*resource.Rr).ActiveRdata = rr.(*resource.Rr).Rdata
		}
	}
}

func (h *rrHandler) checkRR(rr *resource.Rr, zoneName string, tx restdb.Transaction) error {
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

	if rr.RdataBackup != "" {
		_, err := g53.RdataFromString(rrType, rr.RdataBackup)
		if err != nil {
			return fmt.Errorf("invalid rData:%s", err.Error())
		}
	}

	if err := checkNS(rr, zoneName, tx); err != nil {
		return err
	}

	if err := checkMX(rr, zoneName, tx); err != nil {
		return err
	}

	return h.checkRRConflict(rr, tx)
}

func (h *rrHandler) checkRRConflict(rr *resource.Rr, tx restdb.Transaction) error {
	if rr.DataType == resource.DataTypeCNAME && rr.Name == "@" {
		return fmt.Errorf("the name of cname should not be:%s", rr.Name)
	}

	sql := "select count(1) from gr_rr where name = $1 and zone = $2 and id != $3"
	if rr.DataType != resource.DataTypeCNAME {
		sql = "select count(1) from gr_rr where name = $1 and zone = $2 and id != $3 and data_type='CNAME'"
	}

	if count, err := tx.CountEx(TableRR, sql, rr.Name, rr.Zone, rr.ID); err != nil {
		return fmt.Errorf("checkRRConflict list rrs failed:%s", err.Error())
	} else if count > 0 {
		return fmt.Errorf("the CNAME and other types are conflict")
	}

	return nil
}
