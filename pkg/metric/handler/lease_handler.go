package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

var TableHeaderLease = []string{"日期", "租赁总数"}

func getLease(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPLeasesTotal
	leaseValues, err := getValuesFromPrometheus(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s leases failed: %s", ctx.NodeIP, err.Error()))
	}

	dhcp := &resource.Dhcp{Lease: resource.Lease{Values: leaseValues}}
	dhcp.SetID(resource.ResourceIDLease)
	return dhcp, nil
}

func exportLease(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPLeasesTotal
	ctx.TableHeader = TableHeaderLease
	return exportTwoColumns(ctx)
}
