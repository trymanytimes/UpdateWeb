package metric

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/metric/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/metric/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/metric",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	conf := config.GetConfig()
	cli := &http.Client{Timeout: 10 * time.Second}
	apiServer.Schemas.MustImport(&Version, resource.Node{}, handler.NewNodeHandler(conf, cli))
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Node{},
	}
}

//for gen api doc
func RegistHandler(schemas restresource.SchemaManager) {
	schemas.MustImport(&Version, resource.Node{}, &handler.NodeHandler{})
}
