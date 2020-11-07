package log

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/log/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/log/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/log",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	auditLogHandler := handler.NewAuditLogHandler()
	apiServer.Schemas.MustImport(&Version, resource.AuditLog{}, auditLogHandler)
	conf := config.GetConfig()
	cli := &http.Client{Timeout: 10 * time.Second}
	apiServer.Schemas.MustImport(&Version, resource.DNSLog{}, handler.NewDNSLogHandler(conf, cli))
	apiServer.EndUse(auditLogHandler.AuditLoggerHandler())

	apiServer.Schemas.MustImport(&Version, resource.UploadLog{}, handler.NewUploadLogHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.AuditLog{},
		&resource.UploadLog{},
	}
}
