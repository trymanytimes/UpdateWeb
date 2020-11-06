package resource

import (
	"github.com/zdnscloud/gorest/resource"
	restresource "github.com/zdnscloud/gorest/resource"
)

type Ddiuser struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string                   `json:"username" rest:"required=true,minLen=1,maxLen=20" db:"uk"`
	Password                  string                   `json:"password" rest:"required=true,minLen=0,maxLen=20"`
	Comment                   string                   `json:"comment"`
	RoleType                  RoleType                 `json:"roleType"`
	UserGroupIds              []string                 `json:"userGroupIDs"`
	RoleIds                   []string                 `json:"roleIDs"`
	RoleAuthority             map[string]RoleAuthority `json:"-" db:"-"`
}

var AuthUser = "user"

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type UserRole struct {
	resource.ResourceBase `json:",inline"`
	Ddiuser               string `db:"ownby"`
	Role                  string `db:"referto"`
}

func (u Ddiuser) GetActions() []resource.Action {
	return UserAction
}

func (u Ddiuser) RoleAuthorityToUserInfo() UserInfo {
	var roleAuthority []RoleAuthority
	for _, value := range u.RoleAuthority {
		roleAuthority = append(roleAuthority, value)
	}

	return UserInfo{
		UserName: u.Name,
		UserType: string(u.RoleType),
		MenuList: roleAuthority}
}

const (
	ActionLogout         = "logout"
	ActionCurrentUser    = "currentUser"
	ActionChangePassword = "changePassword"
	ActionResetPassword  = "resetPassword"
)

type LogoutResponse struct {
	Result bool   `json:"result"`
	RetMsg string `json:"returnMessage"`
}

type UserInfo struct {
	UserName string          `json:"username"`
	UserType string          `json:"userType"`
	MenuList []RoleAuthority `json:"menuList"`
}

var UserAction = []resource.Action{
	resource.Action{
		Name:   ActionLogout,
		Output: &LogoutResponse{},
	},
	resource.Action{
		Name:   ActionCurrentUser,
		Output: &UserInfo{},
	},
	resource.Action{
		Name:   ActionChangePassword,
		Input:  &LoginRequest{},
		Output: &LoginResponse{},
	},
	resource.Action{
		Name:   ActionResetPassword,
		Input:  &LoginRequest{},
		Output: &LoginResponse{},
	},
}
