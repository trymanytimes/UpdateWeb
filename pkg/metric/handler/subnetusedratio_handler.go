package handler

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/db"
	dhcpresource "github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

func getSubnetUsedRatios(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPUsages
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s subnet used ratio failed: %s", ctx.NodeIP, err.Error()))
	}

	subnets, err := getSubnetsFromDB()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	var subnetUsedRatios resource.SubnetUsedRatios
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if subnetId, ok := r.MetricLabels[agentmetric.MetricLabelSubnetId]; ok {
			for _, subnet := range subnets {
				if subnet.GetID() == subnetId {
					subnetUsedRatios = append(subnetUsedRatios, resource.SubnetUsedRatio{
						Ipnet:      subnet.Ipnet,
						UsedRatios: getRatiosWithTimestamp(r.Values, ctx.Period),
					})
					break
				}
			}
		}
	}

	sort.Sort(subnetUsedRatios)
	dhcp := &resource.Dhcp{SubnetUsedRatios: subnetUsedRatios}
	dhcp.SetID(resource.ResourceIDSubnetUsedRatios)
	return dhcp, nil
}

func getSubnetsFromDB() ([]*dhcpresource.Subnet, error) {
	var subnets []*dhcpresource.Subnet
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, err
	}

	return subnets, nil
}

func getRatiosWithTimestamp(values [][]interface{}, period *TimePeriodParams) []resource.RatioWithTimestamp {
	var ratioWithTimestamps []resource.RatioWithTimestamp
	for i := period.Begin; i <= period.End; i += period.Step {
		ratioWithTimestamps = append(ratioWithTimestamps, resource.RatioWithTimestamp{
			Timestamp: restresource.ISOTime(time.Unix(i, 0)),
			Ratio:     "0",
		})
	}

	for _, vs := range values {
		if t, s := getTimestampAndValue(vs); t != 0 {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				ratioWithTimestamps[(t-period.Begin)/period.Step].Ratio = fmt.Sprintf("%.4f", f)
			}
		}
	}

	return ratioWithTimestamps
}

func exportSubnetUsedRatios(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPUsages
	ctx.MetricLabel = agentmetric.MetricLabelSubnetId
	return exportMultiColunms(ctx)
}
