package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/trymanytimes/UpdateWeb/pkg/auth/authentification"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	authhandler "github.com/trymanytimes/UpdateWeb/pkg/auth/handler"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/log/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

const (
	DefaultAuditLogValidPeriod = 180 //day
)

var AuditLogFilterNames = []string{"source_ip"}

type AuditLogHandler struct {
	defaultAuditLogValidPeriod int
}

func NewAuditLogHandler() *AuditLogHandler {
	defaultAuditLogValidPeriod := DefaultAuditLogValidPeriod

	h := &AuditLogHandler{
		defaultAuditLogValidPeriod: defaultAuditLogValidPeriod,
	}
	go h.run()
	return h
}

func (h *AuditLogHandler) run() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
				_, err := tx.Exec("delete from gr_audit_log where expire < now()")
				return err
			}); err != nil {
				log.Warnf("delete expired audit log failed: %s", err.Error())
			}
		}
	}
}

func (h *AuditLogHandler) AuditLoggerHandler() gorest.EndHandlerFunc {
	return func(ctx *restresource.Context, respErr *resterror.APIError) *resterror.APIError {
		if _, ok := ctx.Get(authhandler.AuditlogIgnore); ok {
			return nil
		}

		var params interface{} = ctx.Resource
		method := ctx.Request.Method
		switch method {
		case http.MethodPost:
			if action := ctx.Resource.GetAction(); action != nil {
				method = action.Name
				params = action.Input
			}
		case http.MethodPut:
		case http.MethodDelete:
			params = nil
		default:
			return nil
		}

		data, err := json.Marshal(params)
		if err != nil {
			return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("marshal %s %s auditlog failed %s",
				method, ctx.Resource.GetType(), err.Error()))
		}

		var errMsg string
		succeed := true
		if respErr != nil {
			succeed = false
			errMsg = respErr.Error()
		}

		sourceIp := ctx.Request.RemoteAddr
		if strings.Contains(sourceIp, ":") {
			sourceIp = strings.Split(sourceIp, ":")[0]
		}

		currentUser, _ := ctx.Get(authentification.AuthUser)
		auditLog := &resource.AuditLog{
			Username:     currentUser.(string),
			SourceIp:     sourceIp,
			Method:       method,
			ResourceKind: restresource.DefaultKindName(ctx.Resource),
			ResourcePath: ctx.Request.URL.Path,
			ResourceId:   ctx.Resource.GetID(),
			Parameters:   string(data),
			Succeed:      succeed,
			ErrMessage:   errMsg,
			Expire:       time.Now().AddDate(0, 0, h.defaultAuditLogValidPeriod),
		}

		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			_, err := tx.Insert(auditLog)
			return err
		}); err != nil {
			return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("insert auditlog %s to db failed %s",
				auditLog.ResourceKind, err.Error()))
		}

		return nil
	}
}

func (h *AuditLogHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var auditLogs []*resource.AuditLog
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		sql, args := util.GenSqlAndArgsByFileters(resource.TableAuditLog, AuditLogFilterNames, h.defaultAuditLogValidPeriod, ctx.GetFilters())
		return tx.FillEx(&auditLogs, sql, args...)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list auditlog faield: %s", err.Error()))
	}

	return auditLogs, nil
}
