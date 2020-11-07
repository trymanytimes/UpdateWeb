package db

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/config"
)

const ConnStr string = "user=%s password=%s host=%s port=%d database=%s sslmode=disable pool_max_conns=10"

var globalResources []resource.Resource

func RegisterResources(resources ...resource.Resource) {
	globalResources = append(globalResources, resources...)
}

var globalDB restdb.ResourceStore

func GetDB() restdb.ResourceStore {
	return globalDB
}

func Init(conf *config.DDIControllerConfig) error {
	meta, err := restdb.NewResourceMeta(globalResources)
	if err != nil {
		return err
	}

	globalDB, err = restdb.NewRStore(fmt.Sprintf(ConnStr, conf.DB.User, conf.DB.Password, conf.DB.Host, conf.DB.Port, conf.DB.Name), meta)
	return err
}

func GetResources(conditions map[string]interface{}, resources interface{}) error {
	return restdb.WithTx(globalDB, func(tx restdb.Transaction) error {
		return tx.Fill(conditions, resources)
	})
}
