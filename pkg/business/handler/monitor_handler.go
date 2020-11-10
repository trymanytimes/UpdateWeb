package handler

import (
	"context"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbHomePage "github.com/trymanytimes/UpdateWeb/pkg/proto/ateStatsHomePage"
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
		log.Errorf("grpc service exec ShowHomePageData failed: %s", err.Error())
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
	return monitor, nil
}
