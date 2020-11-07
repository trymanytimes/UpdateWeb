package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest"
	"github.com/zdnscloud/gorest/resource/schema"

	"github.com/trymanytimes/UpdateWeb/pkg/alarm"
	auth "github.com/trymanytimes/UpdateWeb/pkg/auth"
	authhandler "github.com/trymanytimes/UpdateWeb/pkg/auth/handler"
	authresource "github.com/trymanytimes/UpdateWeb/pkg/auth/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/dhcp"
	"github.com/trymanytimes/UpdateWeb/pkg/dns"
	"github.com/trymanytimes/UpdateWeb/pkg/dns/handler"
	dnshandler "github.com/trymanytimes/UpdateWeb/pkg/dns/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/dns/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/ipam"
	auditlog "github.com/trymanytimes/UpdateWeb/pkg/log"
	"github.com/trymanytimes/UpdateWeb/pkg/metric"
	restresource "github.com/zdnscloud/gorest/resource"
)

var (
	ipamInterfaceDocFolder     string
	dhcpInterfaceDocFolder     string
	dnsInterfaceDocFolder      string
	metricInterfaceDocFolder   string
	alarmInterfaceDocFolder    string
	auditlogInterfaceDocFolder string
	authInterfaceDocFolder     string
	Version                    = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/dns",
	}
)

func main() {
	flag.StringVar(&ipamInterfaceDocFolder, "ipam", "", "ipam api file path")
	flag.StringVar(&dhcpInterfaceDocFolder, "dhcp", "", "dhcp api file path")
	flag.StringVar(&dnsInterfaceDocFolder, "dns", "", "dns api file path")
	flag.StringVar(&metricInterfaceDocFolder, "metric", "", "metric api file path")
	flag.StringVar(&alarmInterfaceDocFolder, "alarm", "", "alarm api file path")
	flag.StringVar(&auditlogInterfaceDocFolder, "auditlog", "", "auditlog api file path")
	flag.StringVar(&authInterfaceDocFolder, "auth", "", "auth api file path")
	flag.Parse()

	log.InitLogger(log.Debug)
	apiServer := gorest.NewAPIServer(schema.NewSchemaManager())
	if ipamInterfaceDocFolder != "" {
		ipam.RegisterHandler(apiServer, nil)
		if err := apiServer.Schemas.WriteJsonDocs(&ipam.Version, ipamInterfaceDocFolder); err != nil {
			log.Fatalf("generate ipam resource doc failed. %s", err.Error())
		}
	}

	if dhcpInterfaceDocFolder != "" {
		dhcp.RegisterHandler(apiServer, nil)
		if err := apiServer.Schemas.WriteJsonDocs(&dhcp.Version, dhcpInterfaceDocFolder); err != nil {
			log.Fatalf("generate dhcp resource doc failed. %s", err.Error())
		}
	}

	if dnsInterfaceDocFolder != "" {
		RegisterDNSHandler(apiServer, nil)
		if err := apiServer.Schemas.WriteJsonDocs(&dns.Version, dnsInterfaceDocFolder); err != nil {
			log.Fatalf("generate dns resource doc failed. %s", err.Error())
		}
	}

	if metricInterfaceDocFolder != "" {
		metric.RegistHandler(apiServer.Schemas)
		if err := apiServer.Schemas.WriteJsonDocs(&metric.Version, metricInterfaceDocFolder); err != nil {
			log.Fatalf("generate metric resource doc failed. %s", err.Error())
		}
	}

	if alarmInterfaceDocFolder != "" {
		alarm.RegistHandler(apiServer.Schemas)
		if err := apiServer.Schemas.WriteJsonDocs(&alarm.Version, alarmInterfaceDocFolder); err != nil {
			log.Fatalf("generate alarm resource doc failed. %s", err.Error())
		}
	}

	if auditlogInterfaceDocFolder != "" {
		auditlog.RegisterHandler(apiServer, nil)
		if err := apiServer.Schemas.WriteJsonDocs(&auditlog.Version, auditlogInterfaceDocFolder); err != nil {
			log.Fatalf("generate auditlog resource doc failed. %s", err.Error())
		}
	}

	if authInterfaceDocFolder != "" {
		RegisterAuthHandler(apiServer, nil)
		if err := apiServer.Schemas.WriteJsonDocs(&auth.Version, authInterfaceDocFolder); err != nil {
			log.Fatalf("generate auth resource doc failed. %s", err.Error())
		}
	}
}

func RegisterDNSHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	apiServer.Schemas.MustImport(&Version, resource.Acl{}, &dnshandler.ACLHandler{})
	apiServer.Schemas.MustImport(&Version, resource.View{}, &dnshandler.ViewHandler{})
	apiServer.Schemas.MustImport(&Version, resource.Zone{}, handler.NewZoneHandler())
	apiServer.Schemas.MustImport(&Version, resource.ForwardZone{}, handler.NewForwardZoneHandler())
	apiServer.Schemas.MustImport(&Version, resource.Rr{}, handler.NewRRHandler())
	apiServer.Schemas.MustImport(&Version, resource.Forward{}, handler.NewForwardHandler())
	apiServer.Schemas.MustImport(&Version, resource.IpBlackHole{}, handler.NewIPBlackHoleHandler())
	apiServer.Schemas.MustImport(&Version, resource.RecursiveConcurrent{}, handler.NewRecursiveConcurrentHandler())
	apiServer.Schemas.MustImport(&Version, resource.Redirection{}, handler.NewRedirectionHandler())
	apiServer.Schemas.MustImport(&Version, resource.DnsGlobalConfig{}, &dnshandler.GlobalConfigHandler{})
	apiServer.Schemas.MustImport(&Version, resource.UrlRedirect{}, &dnshandler.UrlRedirectHandler{})
	return nil
}
func RegisterAuthHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	apiServer.Schemas.MustImport(&Version, authresource.Ddiuser{}, &authhandler.UserHandler{})
	apiServer.Schemas.MustImport(&Version, authresource.Role{}, &authhandler.RoleHandler{})
	apiServer.Schemas.MustImport(&Version, authresource.UserGroup{}, authhandler.NewUserGroupHandler())
	return nil
}
