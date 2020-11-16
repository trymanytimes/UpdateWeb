package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbWeb "github.com/trymanytimes/UpdateWeb/pkg/proto/rcs"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

type WebsiteHandler struct{}

func NewWebsiteHandler() *WebsiteHandler {
	return &WebsiteHandler{}
}

func (h *WebsiteHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	website := ctx.Resource.(*resource.Website)
	if err := h.OptRaltWebsite(website, OperTypeCreate); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("OptRaltWebsite error: %s", err.Error()))
	}
	return website, nil
}

func (h *WebsiteHandler) OptRaltWebsite(website *resource.Website, operType int32) *resterror.APIError {
	cli := grpcclient.GetGrpcClient()
	websiteReq := &pbWeb.OptRaltWebsiteReq{Iopt: operType}
	addrSrcDomain, err := net.ResolveIPAddr("ip", website.SourceDomain)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("Resolution %s for IP error: %s", website.SourceDomain, err.Error()))
	}
	web := &pbWeb.WebsiteReqInfo{
		StrdomainId:          website.GetID(),
		StrgroupId:           website.GroupID,
		StrsrcDomain:         website.SourceDomain,
		StrdstDomain:         website.DestDomain,
		StrsrcIpAddr:         addrSrcDomain.IP.String(),
		StripAddr:            website.VirtualIP,
		StrwebsiteHrefDomain: website.HrefDomain,
		I64Mod:               TransformModType64,
	}
	for _, v := range website.ProtocolPorts {
		web.ProtocolMap = append(web.ProtocolMap, &pbWeb.ProtocolMap{
			StrprotocolMapId: util.CreateRandomString(4),
			StrsrcProtocol:   v.SourceProtocol,
			IsrcPort:         v.SourcePort,
			StrdstProtocol:   v.DestProtocol,
			IdstPort:         v.DestPort,
		})
	}
	websiteReq.Website = append(websiteReq.Website, web)
	if _, err := cli.WebsiteClient.OptRaltWebsite(context.Background(), websiteReq); err != nil {
		log.Errorf("", err.Error())
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec OptRaltWebsite failed: %s", err.Error()))
	}
	return nil
}

func (h *WebsiteHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	website := ctx.Resource.(*resource.Website)
	cli := grpcclient.GetGrpcClient()
	websiteReq := &pbWeb.OptRaltWebsiteReq{Iopt: OperTypeDelete}
	web := &pbWeb.WebsiteReqInfo{
		StrdomainId: website.GetID(),
		StrgroupId:  website.GroupID,
	}
	websiteReq.Website = append(websiteReq.Website, web)
	if _, err := cli.WebsiteClient.OptRaltWebsite(context.Background(), websiteReq); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec OptRaltWebsite failed: %s", err.Error()))
	}
	return nil
}

func (h *WebsiteHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	website := ctx.Resource.(*resource.Website)
	if err := h.OptRaltWebsite(website, OperTypeModify); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("OptRaltWebsite error: %s", err.Error()))
	}
	return website, nil
}

func (h *WebsiteHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	website := ctx.Resource.(*resource.Website)
	cli := grpcclient.GetGrpcClient()
	//query wether exists a Website
	WebsiteReq := pbWeb.GetRaltSpecWebsiteReq{StrdomainId: website.GetID()}
	rsp, err := cli.WebsiteClient.GetRaltSpecWebsite(context.Background(), &WebsiteReq)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec GetRaltSpecWebsite failed: %s", err.Error()))
	}
	if len(rsp.Website) == 0 {
		return website, nil
	}
	website.GroupID = rsp.GetWebsite()[0].GetStrgroupId()
	website.SourceDomain = rsp.GetWebsite()[0].GetStrsrcDomain()
	website.DestDomain = rsp.GetWebsite()[0].GetStrdstDomain()
	website.VirtualIP = rsp.GetWebsite()[0].GetStripAddr()
	website.HrefDomain = rsp.GetWebsite()[0].GetStrwebsiteHrefDomain()
	website.TransferMod = rsp.GetWebsite()[0].GetI64Mod()
	for _, v := range rsp.Website[0].ProtocolMap {
		website.ProtocolPorts = append(website.ProtocolPorts, &resource.ProtocolPort{
			SourceProtocol: v.StrsrcProtocol,
			SourcePort:     v.IsrcPort,
			DestProtocol:   v.StrdstProtocol,
			DestPort:       v.IdstPort,
		})
	}

	return website, nil
}

func (h *WebsiteHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var websites []*resource.Website
	req := pbWeb.GetRaltGroupWebsiteReq{StrgroupId: ctx.Resource.GetParent().GetID()}
	cli := grpcclient.GetGrpcClient()
	rsp, err := cli.WebsiteClient.GetRaltGroupWebsite(context.Background(), &req)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec GetRaltGroupWebsite failed: %s", err.Error()))
	}
	for _, v := range rsp.Website {
		var protocolPorts []*resource.ProtocolPort
		for _, p := range v.ProtocolMap {
			protocolPorts = append(protocolPorts, &resource.ProtocolPort{
				SourceProtocol: p.StrsrcProtocol,
				SourcePort:     p.IsrcPort,
				DestProtocol:   p.StrdstProtocol,
				DestPort:       p.IdstPort,
			})
		}
		web := &resource.Website{
			GroupID:       req.StrgroupId,
			SourceDomain:  v.StrsrcDomain,
			DestDomain:    v.StrdstDomain,
			VirtualIP:     v.StrsrcIpAddr,
			ProtocolPorts: protocolPorts,
		}
		web.SetID(v.StrdomainId)
		websites = append(websites, web)
	}
	return websites, nil
}
