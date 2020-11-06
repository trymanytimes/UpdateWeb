package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

func getQueryTypeRatios(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSQueryTypeRatios
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s querytype ratios failed: %s", ctx.NodeIP, err.Error()))
	}

	var queryTypeRatios []resource.QueryTypeRatio
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if qtype, ok := r.MetricLabels[agentmetric.MetricLabelType]; ok {
			queryTypeRatios = append(queryTypeRatios, resource.QueryTypeRatio{
				Type:   qtype,
				Ratios: getRatiosWithTimestamp(r.Values, ctx.Period),
			})
		}
	}

	dns := &resource.Dns{QueryTypeRatios: queryTypeRatios}
	dns.SetID(resource.ResourceIDQueryTypeRatios)
	return dns, nil
}

func exportQueryTypeRatios(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSQueryTypeRatios
	ctx.MetricLabel = agentmetric.MetricLabelType
	return exportMultiColunms(ctx)
}
