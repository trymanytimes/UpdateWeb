package handler

import (
	"fmt"
	"net"
	"strings"

	"github.com/linkingthing/ddi-controller/config"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

var (
	TableWhiteList = restdb.ResourceDBType(&resource.WhiteList{})
	WhiteListID    = "whitelist"
)

type WhiteListHandler struct{}

func NewWhiteListHandler() (*WhiteListHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableWhiteList, map[string]interface{}{restdb.IDField: WhiteListID}); err != nil {
			return err
		} else if !exists {
			whitelist := resource.WhiteList{Enabled: false}
			whitelist.SetID(WhiteListID)
			whitelist.Privilege = config.GetConfig().Server.IP
			if _, err = tx.Insert(&whitelist); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("insert whitelist data into db failed:%s", err.Error())
	}
	return &WhiteListHandler{}, nil
}

func (h *WhiteListHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	whiteList := ctx.Resource.(*resource.WhiteList)
	if whiteList.Enabled && len(whiteList.Ips) == 0 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("ips should not be empty"))
	}

	for _, ip := range whiteList.Ips {
		if err := checkIP(ip); err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
		}
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableWhiteList, map[string]interface{}{
			"ips":     whiteList.Ips,
			"enabled": whiteList.Enabled,
		}, map[string]interface{}{restdb.IDField: whiteList.GetID()}); err != nil {
			return fmt.Errorf("update WhiteList rr:%s", err.Error())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return whiteList, nil
}

func checkIP(subnetOrIp string) error {
	if strings.Contains(subnetOrIp, "/") {
		if _, subnet, err := net.ParseCIDR(subnetOrIp); err != nil {
			return err
		} else if subnet.String() != subnetOrIp {
			return fmt.Errorf("bad subnet:%s", subnetOrIp)
		}
	} else if err := net.ParseIP(subnetOrIp); err == nil {
		return fmt.Errorf("bad ip:%s", subnetOrIp)
	}

	return nil
}

func (h *WhiteListHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	whiteList, err := restdb.GetResourceWithID(db.GetDB(), ctx.Resource.GetID(), &[]*resource.WhiteList{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get WhiteList %s failed: %s", ctx.Resource.GetID(), err.Error()))
	}

	return whiteList.(*resource.WhiteList), nil
}

func (h *WhiteListHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var WhiteLists []*resource.WhiteList
	if err := db.GetResources(map[string]interface{}{"orderby": "create_time"}, &WhiteLists); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list WhiteList failed: %s", err.Error()))
	}

	return WhiteLists, nil
}
