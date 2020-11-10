package handler

import (
	"context"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbWeb "github.com/trymanytimes/UpdateWeb/pkg/proto/rcs"
)

var (
	TransformModType64 = int32(1)
)

type WebGroupHandler struct{}

func NewWebGroupHandler() (*WebGroupHandler, error) {
	return &WebGroupHandler{}, nil
}

func (h *WebGroupHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	webGroup := ctx.Resource.(*resource.WebGroup)
	cli := grpcclient.GetGrpcClient()
	webGroupIDReq := &pbWeb.OptRaltGroupReq{
		Iopt:               OperTypeCreate,
		StrgroupId:         webGroup.ID,
		StrgroupName:       webGroup.Name,
		StrgroupHrefDomain: webGroup.HrefDomain,
		ItransformMod:      TransformModType64,
		StrclusterId:       DefaultClusterID,
	}
	webGroupIDReq.FuncSwitcher.BreplaceHref = webGroup.UpdateSwithcher.IsReplaceHrefOn
	webGroupIDReq.FuncSwitcher.BhttpsToHttp = webGroup.UpdateSwithcher.IsHttpsToHttpOn
	webGroupIDReq.FuncSwitcher.Binet6Cache = webGroup.UpdateSwithcher.IsCacheOn
	for _, v := range webGroup.Rules {
		webGroupIDReq.Rule = append(webGroupIDReq.Rule, &pbWeb.RuleInfo{
			StrruleId:  v.ID,
			IruleType:  v.RuleType,
			Strsearch:  v.SearchString,
			Strreplace: v.ReplaceString,
		})
	}
	if _, err := cli.WebsiteClient.OptRaltGroup(context.Background(), webGroupIDReq); err != nil {
		log.Errorf("grpc service exec OptRaltGroup failed: %s", err.Error())
	}

	return webGroup, nil
}

func (h *WebGroupHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	webGroup := ctx.Resource.(*resource.WebGroup)
	cli := grpcclient.GetGrpcClient()
	webGroupIDReq := &pbWeb.OptRaltGroupReq{
		Iopt:         OperTypeDelete,
		StrgroupId:   webGroup.ID,
		StrclusterId: DefaultClusterID,
	}
	if _, err := cli.WebsiteClient.OptRaltGroup(context.Background(), webGroupIDReq); err != nil {
		log.Errorf("grpc service exec OptRaltGroup failed: %s", err.Error())
	}
	return nil
}

func (h *WebGroupHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	webGroup := ctx.Resource.(*resource.WebGroup)
	cli := grpcclient.GetGrpcClient()
	webGroupIDReq := &pbWeb.OptRaltGroupReq{
		Iopt:               OperTypeModify,
		StrgroupId:         webGroup.ID,
		StrgroupName:       webGroup.Name,
		StrgroupHrefDomain: webGroup.HrefDomain,
		ItransformMod:      TransformModType64,
		StrclusterId:       DefaultClusterID,
	}
	webGroupIDReq.FuncSwitcher.BreplaceHref = webGroup.UpdateSwithcher.IsReplaceHrefOn
	webGroupIDReq.FuncSwitcher.BhttpsToHttp = webGroup.UpdateSwithcher.IsHttpsToHttpOn
	webGroupIDReq.FuncSwitcher.Binet6Cache = webGroup.UpdateSwithcher.IsCacheOn
	for _, v := range webGroup.Rules {
		webGroupIDReq.Rule = append(webGroupIDReq.Rule, &pbWeb.RuleInfo{
			StrruleId:  v.ID,
			IruleType:  v.RuleType,
			Strsearch:  v.SearchString,
			Strreplace: v.ReplaceString,
		})
	}
	if _, err := cli.WebsiteClient.OptRaltGroup(context.Background(), webGroupIDReq); err != nil {
		log.Errorf("grpc service exec OptRaltGroup failed: %s", err.Error())
	}
	return webGroup, nil
}

func (h *WebGroupHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	webGroup := ctx.Resource.(*resource.WebGroup)
	cli := grpcclient.GetGrpcClient()
	//query wether exists a WebGroup
	WebGroupIDReq := pbWeb.GetRaltGroupReq{StrgroupId: webGroup.GetID()}
	defaultWebGroup, err := cli.WebsiteClient.GetRaltGroup(context.Background(), &WebGroupIDReq)
	if err != nil {
		log.Errorf("grpc service exec GetRaltGroup failed: %s", err.Error())
	}
	if len(defaultWebGroup.GroupList) == 0 {
		log.Errorf("grpc service exec GetRaltGroup failed: group for id %s is not exists", webGroup.GetID())
	}
	c := &resource.WebGroup{
		Name:         defaultWebGroup.GroupList[0].StrgroupName,
		HrefDomain:   defaultWebGroup.GroupList[0].StrgroupHrefDomain,
		TransformMod: defaultWebGroup.GroupList[0].ItransformMod,
		ClusterID:    defaultWebGroup.GroupList[0].StrclusterId,
	}
	c.SetID(defaultWebGroup.GroupList[0].StrgroupId)
	c.UpdateSwithcher.IsCacheOn = defaultWebGroup.GroupList[0].FuncSwitcher.Binet6Cache
	c.UpdateSwithcher.IsHttpsToHttpOn = defaultWebGroup.GroupList[0].FuncSwitcher.BhttpsToHttp
	c.UpdateSwithcher.IsReplaceHrefOn = defaultWebGroup.GroupList[0].FuncSwitcher.BreplaceHref
	for _, v := range defaultWebGroup.GroupList[0].Rule {
		c.Rules = append(c.Rules, &resource.RuleInfo{
			ID:            v.StrruleId,
			RuleType:      v.IruleType,
			SearchString:  v.Strsearch,
			ReplaceString: v.Strreplace,
		})
	}

	return c, nil
}

func (h *WebGroupHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var webGroups []*resource.WebGroup
	cli := grpcclient.GetGrpcClient()
	//query wether exists a WebGroup
	defaultWebGroups, err := cli.WebsiteClient.GetRaltGroup(context.Background(), &pbWeb.GetRaltGroupReq{})
	if err != nil {
		log.Errorf("grpc service exec GetRaltGroup failed: %s", err.Error())
	}
	if len(defaultWebGroups.GroupList) == 0 {
		return webGroups, nil
	}
	for _, v := range defaultWebGroups.GroupList {
		c := &resource.WebGroup{
			Name:         v.StrgroupName,
			HrefDomain:   v.StrgroupHrefDomain,
			TransformMod: v.ItransformMod,
			ClusterID:    v.StrclusterId,
		}
		c.SetID(defaultWebGroups.GroupList[0].StrgroupId)
		c.UpdateSwithcher.IsCacheOn = defaultWebGroups.GroupList[0].FuncSwitcher.Binet6Cache
		c.UpdateSwithcher.IsHttpsToHttpOn = defaultWebGroups.GroupList[0].FuncSwitcher.BhttpsToHttp
		c.UpdateSwithcher.IsReplaceHrefOn = defaultWebGroups.GroupList[0].FuncSwitcher.BreplaceHref
		for _, rule := range v.Rule {
			c.Rules = append(c.Rules, &resource.RuleInfo{
				ID:            rule.StrruleId,
				RuleType:      rule.IruleType,
				SearchString:  rule.Strsearch,
				ReplaceString: rule.Strreplace,
			})
		}
		webGroups = append(webGroups, c)
	}

	return webGroups, nil
}
