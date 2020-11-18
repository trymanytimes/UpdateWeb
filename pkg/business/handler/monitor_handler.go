package handler

import (
	"context"
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbHomePage "github.com/trymanytimes/UpdateWeb/pkg/proto/ateStatsHomePage"
	pbWeb "github.com/trymanytimes/UpdateWeb/pkg/proto/rcs"
)

var MonitorID = "m001"

type HomePageHandler struct{}

func NewHomePageHandler() *HomePageHandler {
	return &HomePageHandler{}
}

func (h *HomePageHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cli := grpcclient.GetGrpcClient()
	//query wether exists a cluster
	req := pbHomePage.ShowHomePageDataReq{ClusterId: DefaultClusterID}
	resp, err := cli.MonitorClient.ShowHomePageData(context.Background(), &req)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec ShowHomePageData failed: %s", err.Error()))
	}
	monitor := &resource.HomePage{
		NormalDomains:       resp.GetDomainIsNormal(),
		AbnormalDomains:     resp.GetDomainIsAbnormal(),
		DomainsCount:        resp.GetDomainIsNormal() + resp.GetDomainIsAbnormal(),
		NormalIPv6Address:   resp.GetNormalIpv6IpaddrNumber(),
		AbnormalIPv6Address: resp.GetAbnormalIpv6IpaddrNumber(),
		IPv6AddressCount:    resp.GetNormalIpv6IpaddrNumber() + resp.GetAbnormalIpv6IpaddrNumber(),
		NormalNode:          resp.GetNormalNodeNumber(),
		AbnormalNode:        resp.GetAbnormalNodeNumber(),
		NodesCount:          resp.GetNormalNodeNumber() + resp.GetAbnormalNodeNumber(),
	}
	monitor.SetID(MonitorID)
	//get all the website visit data
	reqVisit := pbHomePage.ShowDomainVisitorDataReq{ClusterId: DefaultClusterID}
	allVisit, err := cli.MonitorClient.ShowDomainVisitorData(context.Background(), &reqVisit)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec ShowDomainVisitorData failed: %s", err.Error()))
	}
	type websiteVisit struct {
		MemberDomain string
		Count        uint64
	}
	var allVisitData []websiteVisit
	for _, v := range allVisit.GetDomainVisitNum() {
		allVisitData = append(allVisitData, websiteVisit{MemberDomain: v.MemberDomain, Count: v.VisitHostNumber[len(v.VisitHostNumber)-1].VisitDomainNum})
	}

	fmt.Println("allVisitData:", allVisitData)
	//get all the group id
	//query wether exists a WebGroup
	webGroups, err := cli.WebsiteClient.GetRaltGroup(context.Background(), &pbWeb.GetRaltGroupReq{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec GetRaltGroup failed: %s", err.Error()))
	}
	for _, group := range webGroups.GroupList {
		fmt.Println("group:", group)
		req := pbWeb.GetRaltGroupWebsiteReq{StrgroupId: group.GetStrgroupId()}
		rsp, err := cli.WebsiteClient.GetRaltGroupWebsite(context.Background(), &req)
		if err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec GetRaltGroupWebsite failed: %s", err.Error()))
		}
		var groupVisit resource.GroupVisit
		for _, website := range rsp.Website {
			fmt.Println("website:", website)
			for _, domainVisit := range allVisitData {
				fmt.Println("domainVisit:", domainVisit)
				if domainVisit.MemberDomain == website.StrsrcDomain {
					groupVisit.Count += domainVisit.Count
					groupVisit.WebsiteVisits = append(groupVisit.WebsiteVisits, &resource.WebsiteVisit{
						Domain: domainVisit.MemberDomain,
						Count:  domainVisit.Count,
					})
				}
			}
		}
		monitor.GroupVisit = append(monitor.GroupVisit, &groupVisit)
	}
	return monitor, nil
}
