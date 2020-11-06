package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

const (
	HttpMethodGET = "GET"
)

var (
	PromQueryUrl   = "http://%s/api/v1/query_range?query=%s{node='%s'}&start=%d&end=%d&step=%d"
	TableHeaderQPS = []string{"日期", "QPS"}
)

func getQps(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSQPS
	qpsValues, err := getValuesFromPrometheus(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s qps failed: %s", ctx.NodeIP, err.Error()))
	}

	dns := &resource.Dns{Qps: resource.Qps{Values: qpsValues}}
	dns.SetID(resource.ResourceIDQPS)
	return dns, nil
}

func exportQps(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSQPS
	ctx.TableHeader = TableHeaderQPS
	return exportTwoColumns(ctx)
}
