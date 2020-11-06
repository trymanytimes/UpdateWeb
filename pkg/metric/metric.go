package metric

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/metric/handler"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
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
	apiServer.Schemas.MustImport(&Version, resource.Dns{}, handler.NewDnsHandler(conf, cli))
	apiServer.Schemas.MustImport(&Version, resource.Dhcp{}, handler.NewDhcpHandler(conf, cli))
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
	schemas.MustImport(&Version, resource.Dns{}, &handler.DnsHandler{})
	schemas.MustImport(&Version, resource.Dhcp{}, &handler.DhcpHandler{})
}
