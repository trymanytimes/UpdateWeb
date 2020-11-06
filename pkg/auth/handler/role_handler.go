package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

var (
	TableRole = restdb.ResourceDBType(&resource.Role{})
)

type RoleHandler struct{}

func NewRoleHandler() (*RoleHandler, error) {
	h := &RoleHandler{}
	return h, nil
}

func (h *RoleHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	role := ctx.Resource.(*resource.Role)
	role.SetID(role.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(role)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create role %s to db failed: %s", role.Name, err.Error()))
	}

	return role, nil
}

func (h *RoleHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	role := ctx.Resource.(*resource.Role)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(TableRole, map[string]interface{}{
			"comment": role.Comment,
			"views":   role.Views,
			"plans":   role.Plans,
		}, map[string]interface{}{restdb.IDField: role.GetID()})
		if err != nil {
			return fmt.Errorf("update role %s err:%s", role.Name, err.Error())
		}

		return updateRoleAuthority(role.GetID(), tx)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update role %s to db failed: %s", role.Name, err.Error()))
	}

	return role, nil
}

func (h *RoleHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if c, err := tx.CountEx(TableUser, "select count(1) from gr_ddiuser where $1=any(role_ids)",
			ctx.Resource.GetID()); err != nil {
			return err
		} else if c > 0 {
			return fmt.Errorf("the role has bind other users")
		}

		_, err := tx.Delete(TableRole, map[string]interface{}{restdb.IDField: ctx.Resource.GetID()})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete role %s from db failed: %s", ctx.Resource.GetID(), err.Error()))
	}

	return nil
}

func (h *RoleHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	roleId := ctx.Resource.GetID()
	role, err := restdb.GetResourceWithID(db.GetDB(), roleId, &[]*resource.Role{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get role %s from db failed: %s", roleId, err.Error()))
	}

	return role.(*resource.Role), nil
}

func (h *RoleHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var roles []*resource.Role
	if err := db.GetResources(map[string]interface{}{"orderby": "create_time"}, &roles); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list role from db failed: %s", err.Error()))
	}

	return roles, nil
}

func updateRoleAuthority(roleID string, tx restdb.Transaction) error {
	var users []*resource.Ddiuser
	err := tx.FillEx(&users, "select * from gr_ddiuser where $1=any(role_ids)", roleID)
	if err != nil {
		return err
	}

	for _, user := range users {
		if err = reloadUserAuthority(user, tx); err != nil {
			return err
		}

		if err = updateUserToDB(user.GetID(), map[string]interface{}{
			"role_ids":       user.RoleIds,
			"user_group_ids": user.UserGroupIds}, tx); err != nil {
			return err
		}
	}

	return nil
}
