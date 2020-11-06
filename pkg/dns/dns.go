package dns

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/dns/handler"
	"github.com/linkingthing/ddi-controller/pkg/dns/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/dns",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	aclHandler, err := handler.NewACLHandler()
	if err != nil {
		return fmt.Errorf("new acl handler err:%s", err.Error())
	}

	viewHandler, err := handler.NewViewHandler()
	if err != nil {
		return fmt.Errorf("new view handler err:%s", err.Error())
	}
	apiServer.Schemas.MustImport(&Version, resource.Acl{}, aclHandler)
	apiServer.Schemas.MustImport(&Version, resource.View{}, viewHandler)
	apiServer.Schemas.MustImport(&Version, resource.Zone{}, handler.NewZoneHandler())
	apiServer.Schemas.MustImport(&Version, resource.ForwardZone{}, handler.NewForwardZoneHandler())
	apiServer.Schemas.MustImport(&Version, resource.Rr{}, handler.NewRRHandler())
	apiServer.Schemas.MustImport(&Version, resource.Forward{}, handler.NewForwardHandler())
	apiServer.Schemas.MustImport(&Version, resource.IpBlackHole{}, handler.NewIPBlackHoleHandler())
	apiServer.Schemas.MustImport(&Version, resource.RecursiveConcurrent{}, handler.NewRecursiveConcurrentHandler())
	apiServer.Schemas.MustImport(&Version, resource.Redirection{}, handler.NewRedirectionHandler())
	dnsGlobalHandler, err := handler.NewDNSGlobalConfigHandler()
	if err != nil {
		return fmt.Errorf("new dns global config handler err:%s", err.Error())
	}
	apiServer.Schemas.MustImport(&Version, resource.DnsGlobalConfig{}, dnsGlobalHandler)
	apiServer.Schemas.MustImport(&Version, resource.UrlRedirect{}, handler.NewUrlRedirectHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Acl{},
		&resource.View{},
		&resource.ViewAcl{},
		&resource.Zone{},
		&resource.ForwardZone{},
		&resource.Rr{},
		&resource.Forward{},
		&resource.ZoneForward{},
		&resource.IpBlackHole{},
		&resource.RecursiveConcurrent{},
		&resource.Redirection{},
		&resource.DnsGlobalConfig{},
		&resource.UrlRedirect{},
	}
}
