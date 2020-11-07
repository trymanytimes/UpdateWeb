package log

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

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
	apiServer.EndUse(auditLogHandler.AuditLoggerHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.AuditLog{},
	}
}
