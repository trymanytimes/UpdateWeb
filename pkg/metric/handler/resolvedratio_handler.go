package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

func getResolvedRatios(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSResolvedRatios
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s resolved ratios failed: %s", ctx.NodeIP, err.Error()))
	}

	var resolvedRatios []resource.ResolvedRatio
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		switch rcode := r.MetricLabels[agentmetric.MetricLabelRcode]; rcode {
		case agentmetric.RcodeSuccess, agentmetric.RcodeReferral, agentmetric.RcodeNxrrset, agentmetric.RcodeSERVFAIL,
			agentmetric.RcodeFORMERR, agentmetric.RcodeNXDOMAIN, agentmetric.RcodeDuplicate, agentmetric.RcodeDropped,
			agentmetric.RcodeFailure:
			resolvedRatios = append(resolvedRatios, resource.ResolvedRatio{
				Rcode:  rcode,
				Ratios: getRatiosWithTimestamp(r.Values, ctx.Period),
			})
		}
	}

	dns := &resource.Dns{ResolvedRatios: resolvedRatios}
	dns.SetID(resource.ResourceIDResolvedRatios)
	return dns, nil
}

func exportResolvedRatios(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSResolvedRatios
	ctx.MetricLabel = agentmetric.MetricLabelRcode
	return exportMultiColunms(ctx)
}
