package handler

import (
	"fmt"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

var (
	TableUserGroup = restdb.ResourceDBType(&resource.UserGroup{})
)

type UserGroupHandler struct{}

func NewUserGroupHandler() *UserGroupHandler {
	return &UserGroupHandler{}
}

func (h *UserGroupHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	userGroup := ctx.Resource.(*resource.UserGroup)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(userGroup); err != nil {
			return fmt.Errorf("insert user group %s into db fail:%s", userGroup.Name, err.Error())
		}

		return updateUserGroupAuthority(userGroup.GetID(), userGroup.UserIds, tx, false)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return userGroup, nil
}

func (h *UserGroupHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	userGroup := ctx.Resource.(*resource.UserGroup)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableUserGroup, map[string]interface{}{
			"comment":  userGroup.Comment,
			"role_ids": userGroup.RoleIds,
		}, map[string]interface{}{restdb.IDField: userGroup.GetID()}); err != nil {
			return fmt.Errorf("update usergroup failed:%s", err.Error())
		}

		return updateUserGroupAuthority(userGroup.GetID(), userGroup.UserIds, tx, false)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return userGroup, nil
}

func (h *UserGroupHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var userGroup *resource.UserGroup
		userGroupRes, err := tx.Get(TableUserGroup, map[string]interface{}{restdb.IDField: ctx.Resource.GetID()})
		if err != nil {
			return err
		}
		userGroup = userGroupRes.([]*resource.UserGroup)[0]
		if _, err = tx.Delete(TableUserGroup,
			map[string]interface{}{restdb.IDField: ctx.Resource.GetID()}); err != nil {
			return err
		}

		return updateUserGroupAuthority(userGroup.GetID(), userGroup.UserIds, tx, true)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete user group %s from db failed: %s", ctx.Resource.GetID(), err.Error()))
	}

	return nil
}

func (h *UserGroupHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	userGroupId := ctx.Resource.(*resource.UserGroup).GetID()
	var userGroup *resource.UserGroup
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		out, err := tx.Get(TableUserGroup, map[string]interface{}{restdb.IDField: userGroupId})
		if err != nil {
			return err
		}
		userGroup = out.([]*resource.UserGroup)[0]
		var users []*resource.Ddiuser
		if err := tx.FillEx(&users, "select * from gr_ddiuser where $1=any(user_group_ids)",
			userGroup.GetID()); err != nil {
			return err
		}

		for _, user := range users {
			userGroup.UserIds = append(userGroup.UserIds, user.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get user group failed:%s", err.Error()))
	}

	return userGroup, nil
}

func (h *UserGroupHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var userGroups []*resource.UserGroup
	if err := db.GetResources(map[string]interface{}{"orderby": "create_time"}, &userGroups); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list userGroup from db failed: %s", err.Error()))
	}

	return userGroups, nil
}

func updateUserGroupAuthority(groupId string, userIds []string, tx restdb.Transaction, isDelete bool) error {
	var users []*resource.Ddiuser
	if isDelete {
		if err := tx.FillEx(&users,
			"select * from gr_ddiuser where $1=any(user_group_ids)", groupId); err != nil {
			return err
		}
	} else {
		if err := tx.FillEx(&users,
			fmt.Sprintf(`select * from gr_ddiuser where id in ('%s')`,
				strings.Join(userIds, "','"))); err != nil {
			return err
		}
	}

	for _, user := range users {
		user.UserGroupIds = recombineSlices(user.UserGroupIds, []string{groupId}, isDelete)
		if err := reloadUserAuthority(user, tx); err != nil {
			return err
		}

		if err := updateUserToDB(user.GetID(), map[string]interface{}{
			"role_ids":       user.RoleIds,
			"user_group_ids": user.UserGroupIds}, tx); err != nil {
			return err
		}
	}

	return nil
}
