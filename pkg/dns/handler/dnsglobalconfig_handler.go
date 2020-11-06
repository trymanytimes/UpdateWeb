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
	TableGlobalConfig        = restdb.ResourceDBType(&resource.DnsGlobalConfig{})
	globalConfigid           = "globalConfig"
	updateZonesTTLSQL        = "update gr_zone set ttl = $1"
	updateRRsTTLSQL          = "update gr_rr set ttl = $1"
	updateRedirectionsTTLSQL = "update gr_redirection set ttl = $1"
)

type GlobalConfigHandler struct {
}

func NewDNSGlobalConfigHandler() (*GlobalConfigHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableGlobalConfig,
			map[string]interface{}{"id": globalConfigid}); err != nil {
			return err
		} else if exists {
			return nil
		} else {
			globalConfig := resource.DnsGlobalConfig{LogEnable: true, Ttl: 3600, DnssecEnable: false}
			globalConfig.SetID(globalConfigid)
			_, err = tx.Insert(&globalConfig)
			return err
		}
	}); err != nil {
		return nil, fmt.Errorf("insert dns globalConfig into database failed:%s", err.Error())
	}
	return &GlobalConfigHandler{}, nil
}

func (h *GlobalConfigHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	globalConfig := ctx.Resource.(*resource.DnsGlobalConfig)
	var ttlChanged bool

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldGlobalConfigRes, err := tx.Get(TableGlobalConfig,
			map[string]interface{}{restdb.IDField: globalConfig.GetID()})
		if err != nil {
			return fmt.Errorf("get globalconfig from db failed:%s", err.Error())
		}
		oldGlobalConfig := oldGlobalConfigRes.([]*resource.DnsGlobalConfig)[0]

		if ttlChanged = globalConfig.Ttl != oldGlobalConfig.Ttl; !ttlChanged &&
			globalConfig.LogEnable == oldGlobalConfig.LogEnable &&
			globalConfig.DnssecEnable == oldGlobalConfig.DnssecEnable {
			return nil
		}

		if _, err := tx.Update(TableGlobalConfig,
			map[string]interface{}{
				"log_enable":    globalConfig.LogEnable,
				"ttl":           globalConfig.Ttl,
				"dnssec_enable": globalConfig.DnssecEnable},
			map[string]interface{}{"id": globalConfigid}); err != nil {
			return fmt.Errorf("update globalConfig resource err:%s", err.Error())
		}

		if ttlChanged {
			if err := h.UpdateZonesTTL(tx, globalConfig.Ttl); err != nil {
				return fmt.Errorf("update zones's ttl err:%s", err.Error())
			}
			if err := h.UpdateRRsTTL(tx, globalConfig.Ttl); err != nil {
				return fmt.Errorf("update rrs's ttl err:%s", err.Error())
			}
			if err := h.UpdateRedirectionsTTL(tx, globalConfig.Ttl); err != nil {
				return fmt.Errorf("update redirections's ttl err:%s", err.Error())
			}
		}

		return SendKafkaMessage(globalConfigid, kafkaconsumer.UpdateGlobalConfig,
			&pb.UpdateGlobalConfigReq{
				Header:       &pb.DDIRequestHead{Method: "Update", Resource: globalConfig.GetType()},
				LogEnable:    globalConfig.LogEnable,
				DnssecEnable: globalConfig.DnssecEnable,
				TtlChanged:   ttlChanged,
				Ttl:          uint32(globalConfig.Ttl),
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("could not update globalConfig resource:%s", err.Error()))
	}

	return globalConfig, nil
}

func (h *GlobalConfigHandler) UpdateZonesTTL(tx restdb.Transaction, ttl int) error {
	if _, err := tx.Exec(updateZonesTTLSQL, ttl); err != nil {
		return fmt.Errorf("update zone' ttl err:%s", err.Error())
	}
	return nil
}

func (h *GlobalConfigHandler) UpdateRRsTTL(tx restdb.Transaction, ttl int) error {
	if _, err := tx.Exec(updateRRsTTLSQL, ttl); err != nil {
		return fmt.Errorf("update rrs's ttl err:%s", err.Error())
	}
	return nil
}

func (h *GlobalConfigHandler) UpdateRedirectionsTTL(tx restdb.Transaction, ttl int) error {
	if _, err := tx.Exec(updateRedirectionsTTLSQL, ttl); err != nil {
		return fmt.Errorf("update redirections's ttl err:%s", err.Error())
	}

	return nil
}

func (h *GlobalConfigHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var globalConfigs []*resource.DnsGlobalConfig
	if err := db.GetResources(nil, &globalConfigs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list globalConfig resource from db failed: %s", err.Error()))
	}
	return globalConfigs, nil
}

func (h *GlobalConfigHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	globalConfigid := ctx.Resource.(*resource.DnsGlobalConfig).GetID()
	var globalConfigs []*resource.DnsGlobalConfig
	globalConfig, err := restdb.GetResourceWithID(db.GetDB(), globalConfigid, &globalConfigs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get globalConfig resource %s failed: %s", globalConfigid, err.Error()))
	}
	return globalConfig.(*resource.DnsGlobalConfig), nil
}
