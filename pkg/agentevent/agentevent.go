package agentevent

import (
	"github.com/gin-gonic/gin"
	"github.com/trymanytimes/UpdateWeb/pkg/agentevent/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/agentevent/resource"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/agentevent",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	if h, err := handler.NewAgentEventHandler(); err != nil {
		return err
	} else {
		h.RegisterWSHandler(router)
	}

	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.AgentEvent{},
	}
}
