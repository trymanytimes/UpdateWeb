package handler

import (
	"fmt"
	"strconv"
	"time"

	resterror "github.com/zdnscloud/gorest/error"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	dhcpresource "github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	PacketLabelPrefixVersion4 = "pkt4-"
	PacketLabelPrefixVersion6 = "pkt6-"
)

func getPackets(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPPacketsStats
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s packet stats failed: %s", ctx.NodeIP, err.Error()))
	}

	var packets []resource.Packet
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if ptype, ok := r.MetricLabels[agentmetric.MetricLabelType]; ok {
			if version, ok := r.MetricLabels[agentmetric.MetricLabelVersion]; ok {
				packets = append(packets, resource.Packet{
					Version: version,
					Type:    ptype,
					Values:  getValuesWithTimestamp(r.Values, ctx.Period),
				})
			}
		}
	}

	dhcp := &resource.Dhcp{Packets: packets}
	dhcp.SetID(resource.ResourceIDPackets)
	return dhcp, nil
}

func exportPackets(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPPacketsStats
	ctx.MetricLabel = agentmetric.MetricLabelType
	return exportMultiColunms(ctx)
}

func exportMultiColunms(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get node %s %s from prometheus failed: %s", ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	strMatrix, err := genHeaderAndStrMatrix(ctx, resp.Data.Results)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s %s from prometheus failed: %s", ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	filepath, err := exportFile(ctx, strMatrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("export node %s %s failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genHeaderAndStrMatrix(ctx *MetricContext, results []PrometheusDataResult) ([][]string, error) {
	headers := []string{"日期"}
	var subnets []*dhcpresource.Subnet
	if ctx.MetricLabel == agentmetric.MetricLabelSubnetId {
		ss, err := getSubnetsFromDB()
		if err != nil {
			return nil, fmt.Errorf("list subnets failed: %s", err.Error())
		}

		subnets = ss
	}

	var validResults []PrometheusDataResult
	for _, r := range results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if label, ok := r.MetricLabels[ctx.MetricLabel]; ok {
			if ctx.MetricName == agentmetric.MetricNameDHCPUsages {
				found := false
				for _, subnet := range subnets {
					if subnet.GetID() == label {
						label = subnet.Ipnet
						found = true
						break
					}
				}

				if found == false {
					continue
				}
			} else if ctx.MetricName == agentmetric.MetricNameDHCPPacketsStats {
				if version, ok := r.MetricLabels[agentmetric.MetricLabelVersion]; ok {
					if version == agentmetric.DHCPVersion4 {
						label = PacketLabelPrefixVersion4 + label
					} else {
						label = PacketLabelPrefixVersion6 + label
					}
				}
			}

			headers = append(headers, label)
			validResults = append(validResults, r)
		}
	}

	var values []string
	for i := 0; i < len(validResults); i++ {
		values = append(values, "0")
	}

	var matrix [][]string
	for i := ctx.Period.Begin; i <= ctx.Period.End; i += ctx.Period.Step {
		matrix = append(matrix, append([]string{time.Unix(int64(i), 0).Format(util.TimeFormat)}, values...))
	}

	for i, r := range validResults {
		for _, vs := range r.Values {
			if timestamp, value := getTimestampAndValue(vs); timestamp != 0 && timestamp >= ctx.Period.Begin {
				if f, err := strconv.ParseFloat(value, 64); err == nil {
					value = fmt.Sprintf("%.4f", f)
				}
				matrix[(timestamp-ctx.Period.Begin)/ctx.Period.Step][i+1] = value
			}
		}
	}

	ctx.TableHeader = headers
	return matrix, nil
}

func getTimestampAndValue(values []interface{}) (int64, string) {
	var timestamp int64
	var value string
	for _, v := range values {
		if i, ok := v.(float64); ok {
			timestamp = int64(i)
		}

		if s, ok := v.(string); ok {
			value = s
		}
	}

	return timestamp, value
}
