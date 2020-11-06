package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

var TableHeaderCacheHits = []string{"日期", "缓存命中率"}

func getCacheHitRatio(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatio
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s cache hit ratios failed: %s", ctx.NodeIP, err.Error()))
	}

	var cacheHitRatios []resource.RatioWithTimestamp
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok || nodeIp == ctx.NodeIP {
			cacheHitRatios = getRatiosWithTimestamp(r.Values, ctx.Period)
			break
		}
	}

	dns := &resource.Dns{CacheHitRatio: resource.CacheHitRatio{Ratios: cacheHitRatios}}
	dns.SetID(resource.ResourceIDCacheHitRatio)
	return dns, nil
}

func exportCacheHitRatio(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatio
	ctx.TableHeader = TableHeaderCacheHits
	return exportTwoColumns(ctx)
}
