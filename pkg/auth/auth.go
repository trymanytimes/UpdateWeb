package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/auth/authorization"
	"github.com/trymanytimes/UpdateWeb/pkg/auth/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/auth/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/auth",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	if err := authorization.InitAuthorization(); err != nil {
		return err
	}

	roleHandler, err := handler.NewRoleHandler()
	if err != nil {
		return err
	}

	userHandler, err := handler.NewUserHandler()
	if err != nil {
		return err
	}

	whiteListHandler, err := handler.NewWhiteListHandler()
	if err != nil {
		return err
	}

	apiServer.Schemas.MustImport(&Version, resource.Ddiuser{}, userHandler)
	apiServer.Schemas.MustImport(&Version, resource.Role{}, roleHandler)
	apiServer.Schemas.MustImport(&Version, resource.UserGroup{}, handler.NewUserGroupHandler())
	apiServer.Schemas.MustImport(&Version, resource.WhiteList{}, whiteListHandler)
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Ddiuser{},
		&resource.Role{},
		&resource.UserGroup{},
		&resource.UserRole{},
		&resource.WhiteList{},
	}
}
