package authentification

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/handler"
	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var (
	signingKey = []byte("Linking-ddi-cooper")
	ViewKey    = "authViewList"
	PrefixKey  = "authPlanList"
	AuthKey    = "authorization"
	AuthUser   = resource.AuthUser
)

type LinkingClaims struct {
	*jwt.StandardClaims
	UserName string
}

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		return authentification(c)
	}
}

func Login(ctx *gin.Context) {
	if errorCode, err := checkLogin(ctx); err != nil {
		ctx.JSON(errorCode.Status,
			resource.LoginResponse{Code: errorCode.Status, Message: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resource.LoginResponse{Code: http.StatusOK})
}

func checkLogin(ctx *gin.Context) (resterror.ErrorCode, error) {
	if err := checkWhiteList(ctx.ClientIP()); err != nil {
		return resterror.Unauthorized, err
	}

	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return resterror.ServerError, fmt.Errorf("bad data")
	}

	login := &resource.LoginRequest{}
	if err = json.Unmarshal(body, login); err != nil {
		return resterror.ServerError, fmt.Errorf("bad data")
	}

	if err := handler.CheckPassword(login.Username, login.Password); err != nil {
		return resterror.ServerError, err
	}

	tokenString, err := createToken(login.Username)
	if err != nil {
		return resterror.ServerError, fmt.Errorf("authorization error")
	}

	user, err := handler.GetUserInfo(login.Username)
	if err != nil {
		return resterror.ServerError, fmt.Errorf("authorization error")
	}
	handler.ActivityUsers.Store(login.Username, user)
	ctx.Header(AuthKey, tokenString)

	return resterror.ErrorCode{}, nil
}

func createToken(userName string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256,
		&LinkingClaims{
			StandardClaims: &jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour * 24).Unix()},
			UserName: userName,
		}).SignedString(signingKey)
}

func authentification(ctx *restresource.Context) *resterror.APIError {
	if err := checkWhiteList(getClientIP(ctx.Request.RemoteAddr)); err != nil {
		return resterror.NewAPIError(resterror.Unauthorized, fmt.Sprintf("forbidden:%s", err.Error()))
	}
	tokenString := ctx.Request.Header.Get(AuthKey)
	if tokenString == "" {
		return resterror.NewAPIError(resterror.Unauthorized, fmt.Sprintf("forbidden:token not exists"))
	}
	tokenOrigin, err := jwt.ParseWithClaims(tokenString, &LinkingClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		})
	if err != nil || !tokenOrigin.Valid {
		return resterror.NewAPIError(resterror.Unauthorized, fmt.Sprintf("forbidden:token invalid"))
	}

	claims, ok := tokenOrigin.Claims.(*LinkingClaims)
	if !ok {
		return resterror.NewAPIError(resterror.Unauthorized, fmt.Sprintf("forbidden:token invalid"))
	}

	user, ok := handler.ActivityUsers.Load(claims.UserName)
	if !ok {
		return resterror.NewAPIError(resterror.Unauthorized, fmt.Sprintf("forbidden:token not exists"))
	}

	ctx.Set(AuthUser, claims.UserName)
	if claims.UserName != handler.Admin {
		if err := checkAuthority(ctx, user.(*resource.Ddiuser)); err != nil {
			return resterror.NewAPIError(resterror.PermissionDenied, err.Error())
		}
	}

	return nil
}

func checkAuthority(ctx *restresource.Context, user *resource.Ddiuser) error {
	haveAuthority := false
	for _, roleAuthority := range user.RoleAuthority {
		if roleAuthority.Resource == ctx.Resource.GetType() {
			for _, operation := range roleAuthority.Operations {
				if string(operation) == ctx.Method {
					haveAuthority = true
					break
				}
			}

			if haveAuthority {
				ctx.Set(PrefixKey, roleAuthority.Plans)
				haveAuthority = checkView(ctx, roleAuthority)
				if haveAuthority {
					ctx.Set(ViewKey, roleAuthority.Views)
					break
				}
			}
		}
	}

	if !haveAuthority {
		return fmt.Errorf("permission denied")
	}

	return nil
}

func checkView(ctx *restresource.Context, roleAuthority resource.RoleAuthority) bool {
	if ctx.Resource.GetID() == "" || len(roleAuthority.Views) == 0 {
		return true
	}

	ancestors := restresource.GetAncestors(ctx.Resource)
	for _, view := range roleAuthority.Views {
		if view == ctx.Resource.GetID() {
			return true
		}

		for _, ancestor := range ancestors {
			if ancestor.GetID() == view {
				return true
			}
		}
	}

	return false
}

func ViewFilter(ctx *restresource.Context, viewId string) bool {
	user, ok := ctx.Get(AuthUser)
	if !ok {
		return false
	}

	if user == handler.Admin {
		return true
	}

	checkViews, ok := ctx.Get(ViewKey)
	if !ok || len(checkViews.([]string)) == 0 {
		return false
	}

	for _, v := range checkViews.([]string) {
		if v == viewId {
			return true
		}
	}

	return false
}

func PrefixFilter(ctx *restresource.Context, subnetOrIps ...string) bool {
	user, ok := ctx.Get(AuthUser)
	if !ok {
		return false
	}

	if user == handler.Admin {
		return true
	}

	authPrefixs, ok := ctx.Get(PrefixKey)
	if !ok || len(authPrefixs.([]string)) == 0 || len(subnetOrIps) == 0 {
		return false
	}

	for _, subnetOrIp := range subnetOrIps {
		for _, authPlan := range authPrefixs.([]string) {
			if authPlan != subnetOrIp {
				return false
			}

			if ok, _ := util.PrefixContainsSubnetOrIP(authPlan, subnetOrIp); !ok {
				return false
			}
		}
	}

	return true
}

func getClientIP(remoteAddr string) string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil {
		return ip
	}

	return ""
}

func checkWhiteList(remoteIP string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		wls, err := tx.Get(handler.TableWhiteList, map[string]interface{}{restdb.IDField: handler.WhiteListID})
		if err != nil {
			return fmt.Errorf("get whitelist failed:%s", err.Error())
		}
		if len(wls.([]*resource.WhiteList)) != 1 {
			return fmt.Errorf("get whitelist failed:record is not 1")
		}
		whiteList := wls.([]*resource.WhiteList)[0]
		if !whiteList.Enabled || whiteList.Privilege == remoteIP {
			return nil
		}

		for _, ip := range whiteList.Ips {
			if net.ParseIP(ip) != nil && ip == remoteIP {
				return nil
			}

			if ok, _ := util.PrefixContainsSubnetOrIP(ip, remoteIP); ok {
				return nil
			}
		}

		return fmt.Errorf("ip not in whitelist:%s", remoteIP)
	})
}
