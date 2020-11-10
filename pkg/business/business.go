package web

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "trymanytimes/business",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	clusterHandler, err := handler.NewClusterHandler()
	if err != nil {
		return fmt.Errorf("new cluster handler err:%s", err.Error())
	}

	apiServer.Schemas.MustImport(&Version, resource.Cluster{}, clusterHandler)
	apiServer.Schemas.MustImport(&Version, resource.HomePage{}, handler.NewHomePageHandler())
	apiServer.Schemas.MustImport(&Version, resource.Host{}, handler.NewHostHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{}
}
