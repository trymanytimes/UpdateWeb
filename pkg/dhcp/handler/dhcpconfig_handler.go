package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
)

var (
	TableDhcpConfig = restdb.ResourceDBType(&resource.DhcpConfig{})
)

const (
	DefaultMinValidLifetime uint32 = 10800
	DefaultMaxValidLifetime uint32 = 14400
	DefaultValidLifetime    uint32 = 14400
	DefaultIdentify                = "dhcpglobalconfig"
)

type DhcpConfigHandler struct {
}

func NewDhcpConfigHandler() *DhcpConfigHandler {
	return &DhcpConfigHandler{}
}

func (s *DhcpConfigHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(nil, &configs); err != nil {
			return err
		}

		if len(configs) == 0 {
			config := &resource.DhcpConfig{
				Identify:         DefaultIdentify,
				MinValidLifetime: DefaultMinValidLifetime,
				MaxValidLifetime: DefaultMaxValidLifetime,
				ValidLifetime:    DefaultValidLifetime,
			}
			tx.Insert(config)
			configs = append(configs, config)
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list global config from db failed: %s", err.Error()))
	}

	return configs, nil
}

func (s *DhcpConfigHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	configID := ctx.Resource.(*resource.DhcpConfig).GetID()
	var configs []*resource.DhcpConfig
	config, err := restdb.GetResourceWithID(db.GetDB(), configID, &configs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get global config %s from db failed: %s", configID, err.Error()))
	}

	return config.(*resource.DhcpConfig), nil
}

func (s *DhcpConfigHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	config := ctx.Resource.(*resource.DhcpConfig)
	if err := checkDhcpConfigValid(config); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update global config params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(TableDhcpConfig, map[string]interface{}{
			"validLifetime":    config.ValidLifetime,
			"maxValidLifetime": config.MaxValidLifetime,
			"minValidLifetime": config.MinValidLifetime,
			"domainServers":    config.DomainServers,
		}, map[string]interface{}{restdb.IDField: config.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update global config %s failed: %s", config.GetID(), err.Error()))
	}

	return config, nil
}

func checkDhcpConfigValid(config *resource.DhcpConfig) error {
	var version resource.Version
	if err := checkIPsValid(config.DomainServers, version); err != nil {
		return fmt.Errorf("domain servers invalid: %s", err.Error())
	}

	return checkLifetimeValid(config.ValidLifetime, config.MinValidLifetime, config.MaxValidLifetime)
}

func checkLifetimeValid(validLifetime, minValidLifetime, maxValidLifetime uint32) error {
	if minValidLifetime < 3600 {
		return fmt.Errorf("min-lifetime %d must not less than 3600", minValidLifetime)
	}

	if minValidLifetime > maxValidLifetime {
		return fmt.Errorf("min-lifetime must less than max-lifetime")
	}

	if validLifetime < minValidLifetime || validLifetime > maxValidLifetime {
		return fmt.Errorf("default lifetime %d is not between min-lifttime %d and max-lifetime %d",
			validLifetime, minValidLifetime, maxValidLifetime)
	}

	return nil
}
