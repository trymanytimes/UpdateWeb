package handler

import (
	"fmt"
	"strings"
	"sync"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/authorization"
	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var (
	TableUser      = restdb.ResourceDBType(&resource.Ddiuser{})
	Admin          = "admin"
	aesKey         = []byte("linkingthing.com")
	AuditlogIgnore = "auditlogIgnore"
	ActivityUsers  = sync.Map{}
)

type UserHandler struct{}

func NewUserHandler() (*UserHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var users []*resource.Ddiuser
		if err := tx.Fill(nil, &users); err != nil {
			return err
		}

		haveAdmin := false
		for _, user := range users {
			if err := reloadUserAuthority(user, tx); err != nil {
				return err
			}
			ActivityUsers.Store(user.Name, user)
			if user.Name == Admin {
				haveAdmin = true
			}
		}

		if !haveAdmin {
			encryptText, err := util.Encrypt(aesKey, Admin)
			if err != nil {
				return err
			}
			user := &resource.Ddiuser{Name: Admin, Password: encryptText, RoleType: resource.RoleTypeSUPER}
			user.SetID(Admin)
			if _, err = tx.Insert(user); err != nil {
				return err
			}
			ActivityUsers.Store(user.Name, user)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not insert admin and password:%s", err.Error())
	}

	return &UserHandler{}, nil
}

func CheckPassword(userName, password string) error {
	var ddiUsers []*resource.Ddiuser
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: userName}, &ddiUsers); err != nil {
			return err
		} else if len(ddiUsers) != 1 {
			return fmt.Errorf("user or password is incorrect")
		}

		if passwordDecrypt, err := util.Decrypt(aesKey, ddiUsers[0].Password); err != nil {
			return err
		} else if passwordDecrypt != password {
			return fmt.Errorf("user or password is incorrect")
		}

		return nil
	})
}

func GetUserInfo(userName string) (*resource.Ddiuser, error) {
	var ddiUser *resource.Ddiuser
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		out, err := tx.Get(TableUser, map[string]interface{}{restdb.IDField: userName})
		if err != nil {
			return err
		}

		ddiUser = out.([]*resource.Ddiuser)[0]
		err = reloadUserAuthority(ddiUser, tx)
		return err
	}); err != nil {
		return nil, err
	}

	return ddiUser, nil
}

func (h *UserHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	user := ctx.Resource.(*resource.Ddiuser)
	if user.Password == "" {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("password should not be empty"))
	}

	encryptText, err := util.Encrypt(aesKey, user.Password)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get password failed"))
	}
	user.Password = encryptText
	user.SetID(user.Name)
	user.RoleType = resource.RoleTypeNORMAL
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(user); err != nil {
			return err
		}

		return reloadUserAuthority(user, tx)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create user %s failed: %s", user.Name, err.Error()))
	}

	ActivityUsers.Store(user.Name, user)
	user.Password = ""
	return user, nil
}

func recombineSlices(slice1, slice2 []string, deleteSlice2 bool) []string {
	roleMap := make(map[string]bool)
	var out []string
	for _, s := range slice1 {
		roleMap[s] = true
	}

	for _, s := range slice2 {
		if deleteSlice2 {
			delete(roleMap, s)
		} else {
			roleMap[s] = true
		}
	}

	for k := range roleMap {
		out = append(out, k)
	}

	return out
}

func reloadUserAuthority(user *resource.Ddiuser, tx restdb.Transaction) error {
	if user.RoleType == resource.RoleTypeSUPER {
		return nil
	}

	user.RoleAuthority = authorization.CreateBaseAuthority()
	var groupList []*resource.UserGroup
	if err := tx.FillEx(&groupList,
		fmt.Sprintf(`select * from gr_user_group where id in ('%s')`,
			strings.Join(user.UserGroupIds, "','"))); err != nil {
		return err
	}

	for _, group := range groupList {
		user.RoleIds = append(user.RoleIds, group.RoleIds...)
	}
	user.RoleIds = recombineSlices(user.RoleIds, []string{}, false)

	if len(user.RoleIds) == 0 {
		return nil
	}

	var roleList []*resource.Role
	if err := tx.FillEx(&roleList,
		fmt.Sprintf(`select * from gr_role where id in ('%s')`,
			strings.Join(user.RoleIds, "','"))); err != nil {
		return err
	}

	var views []string
	var planIds []string
	for _, role := range roleList {
		views = append(views, role.Views...)
		planIds = append(planIds, role.Plans...)
	}
	views = recombineSlices(views, []string{}, false)
	authorization.CreateViewAuthority(views, user.RoleAuthority)

	var planList []*ipamresource.Plan
	if err := tx.FillEx(&planList,
		fmt.Sprintf(`select * from gr_plan where id in ('%s')`,
			strings.Join(planIds, "','"))); err != nil {
		return err
	}

	var plans []string
	for _, plan := range planList {
		plans = append(plans, plan.Prefix)
	}
	plans = recombineSlices(plans, []string{}, false)
	authorization.CreateDhcpAuthority(plans, user.RoleAuthority)

	return nil
}

func (h *UserHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	user, ok := ctx.Get(resource.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("bad user"))
	}

	ddiUser := ctx.Resource.(*resource.Ddiuser)
	if user != Admin && user != ctx.Resource.GetID() {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("can't update other user's info"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := updateUserToDB(ddiUser.ID, map[string]interface{}{
			"comment":        ddiUser.Comment,
			"user_group_ids": ddiUser.UserGroupIds,
			"role_ids":       ddiUser.RoleIds}, tx); err != nil {
			return err
		}

		return reloadUserAuthority(ddiUser, tx)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update user %s to db failed: %s", ddiUser.Name, err.Error()))
	}

	ddiUser.Password = ""
	ActivityUsers.Store(ddiUser.Name, ddiUser)
	return ddiUser, nil
}

func (h *UserHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	if ctx.Resource.GetID() == Admin {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("can't delete user admin"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(TableUser, map[string]interface{}{restdb.IDField: ctx.Resource.GetID()})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete user %s from db failed: %s", ctx.Resource.GetID(), err.Error()))
	}

	ActivityUsers.Delete(ctx.Resource.GetID())
	return nil
}

func (h *UserHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	userId := ctx.Resource.(*resource.Ddiuser).GetID()
	var users []*resource.Ddiuser
	user, err := restdb.GetResourceWithID(db.GetDB(), userId, &users)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get user %s from db failed: %s", userId, err.Error()))
	}

	user.(*resource.Ddiuser).Password = ""
	return user.(*resource.Ddiuser), nil
}

func (h *UserHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var users []*resource.Ddiuser
	if err := db.GetResources(map[string]interface{}{"orderby": "create_time"}, &users); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list user from db failed: %s", err.Error()))
	}

	var visible []*resource.Ddiuser
	for _, user := range users {
		if user.ID != Admin {
			user.Password = ""
			visible = append(visible, user)
		}
	}

	return visible, nil
}

func (h *UserHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ctx.Set(AuditlogIgnore, nil)
	switch ctx.Resource.GetAction().Name {
	case resource.ActionLogout:
		return h.Logout(ctx)
	case resource.ActionCurrentUser:
		return h.CurrentUser(ctx)
	case resource.ActionChangePassword:
		return h.changePassword(ctx)
	case resource.ActionResetPassword:
		return h.resetPassword(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *UserHandler) Logout(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	user, ok := ctx.Get(resource.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("unknown user"))
	}
	ActivityUsers.Delete(user)
	return resource.LogoutResponse{Result: true, RetMsg: ""}, nil
}

func (h *UserHandler) CurrentUser(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	userId, ok := ctx.Get(resource.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("unknown user"))
	}

	userInter, ok := ActivityUsers.Load(userId)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("unknown user"))
	}
	userInfo := userInter.(*resource.Ddiuser).RoleAuthorityToUserInfo()

	return &userInfo, nil
}

func (h *UserHandler) changePassword(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	user, ok := ctx.Get(resource.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("unknown user"))
	}

	input := ctx.Resource.GetAction().Input.(*resource.LoginRequest)
	if user != Admin && user != input.Username {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("bad option"))
	}

	if input.Password == "" {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("password should not be empty"))
	}

	encryptText, err := util.Encrypt(aesKey, input.Password)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return updateUserToDB(input.Username, map[string]interface{}{
			"password": encryptText}, tx)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update password failed:%s", err.Error()))
	}

	input.Password = ""
	return &input, nil
}

func (h *UserHandler) resetPassword(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	user, ok := ctx.Get(resource.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("bad user"))
	}

	if user != Admin {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("permission denied"))
	}

	input := ctx.Resource.GetAction().Input.(*resource.LoginRequest)
	if input.Password == "" {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("password should not be empty"))
	}

	encryptText, err := util.Encrypt(aesKey, input.Password)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return updateUserToDB(input.Username, map[string]interface{}{
			"password": encryptText}, tx)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update password failed:%s", err.Error()))
	}

	input.Password = ""
	return &input, nil
}

func updateUserToDB(userId string, con map[string]interface{}, tx restdb.Transaction) error {
	_, err := tx.Update(TableUser, con,
		map[string]interface{}{restdb.IDField: userId})
	return err
}
