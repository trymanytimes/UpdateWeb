package ipam

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/ipam/handler"
	"github.com/linkingthing/ddi-controller/pkg/ipam/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/ipam",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	apiServer.Schemas.MustImport(&Version, resource.Plan{}, handler.NewPlanHandler())
	apiServer.Schemas.MustImport(&Version, resource.Layout{}, handler.NewLayoutHandler())
	apiServer.Schemas.MustImport(&Version, resource.NetworkEquipment{}, handler.NewNetworkEquipmentHandler())
	apiServer.Schemas.MustImport(&Version, resource.NetNode{}, handler.NewNetNodeHandler())
	scannedHandler := handler.NewScannedSubnetHandler()
	apiServer.Schemas.MustImport(&Version, resource.ScannedSubnet{}, scannedHandler)
	apiServer.Schemas.MustImport(&Version, resource.NetworkInterface{}, handler.NewNetworkInterfaceHandler(scannedHandler))
	apiServer.Schemas.MustImport(&Version, resource.Asset{}, handler.NewAssetHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Plan{},
		&resource.Layout{},
		&resource.NetworkEquipment{},
		&resource.NetworkTopology{},
		&resource.PlanNode{},
		&resource.Asset{},
	}
}
