package handler

import (
	"fmt"
	"strconv"
	"time"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var TableHeaderLPS = []string{"日期", "LPS"}

type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}

type PrometheusData struct {
	Results []PrometheusDataResult `json:"result"`
}

type PrometheusDataResult struct {
	MetricLabels map[string]string `json:"metric"`
	Values       [][]interface{}   `json:"values"`
}

func getLps(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPLPS
	lpsValues, err := getValuesFromPrometheus(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get lps with node %s failed: %s", ctx.NodeIP, err.Error()))
	}

	dhcp := &resource.Dhcp{Lps: resource.Lps{Values: lpsValues}}
	dhcp.SetID(resource.ResourceIDLPS)
	return dhcp, nil
}

func getValuesFromPrometheus(ctx *MetricContext) ([]resource.ValueWithTimestamp, error) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok && nodeIp == ctx.NodeIP {
			return getValuesWithTimestamp(r.Values, ctx.Period), nil
		}
	}

	return nil, nil
}

func prometheusRequest(ctx *MetricContext) (*PrometheusResponse, error) {
	var resp PrometheusResponse
	if err := httpRequest(ctx.Client, HttpMethodGET, genPrometheusUrl(ctx), nil, &resp); err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("get node %s %s failed with status: %s", ctx.NodeIP, ctx.MetricName, resp.Status)
	}

	return &resp, nil
}

func genPrometheusUrl(ctx *MetricContext) string {
	return fmt.Sprintf(PromQueryUrl, ctx.PrometheusAddr, ctx.MetricName, ctx.NodeIP, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
}

func getValuesWithTimestamp(values [][]interface{}, period *TimePeriodParams) []resource.ValueWithTimestamp {
	var valueWithTimestamps []resource.ValueWithTimestamp
	for i := period.Begin; i <= period.End; i += period.Step {
		valueWithTimestamps = append(valueWithTimestamps, resource.ValueWithTimestamp{
			Timestamp: restresource.ISOTime(time.Unix(i, 0)),
			Value:     0,
		})
	}

	for _, vs := range values {
		if t, s := getTimestampAndValue(vs); t != 0 && t >= period.Begin {
			if value, err := strconv.ParseUint(s, 10, 64); err == nil {
				valueWithTimestamps[(t-period.Begin)/period.Step].Value = value
			}
		}
	}

	return valueWithTimestamps
}

func exportLps(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDHCPLPS
	ctx.TableHeader = TableHeaderLPS
	return exportTwoColumns(ctx)
}

func exportTwoColumns(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("get node %s %s from prometheus failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	var strMatrix [][]string
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok && nodeIp == ctx.NodeIP {
			strMatrix = genStrMatrix(r.Values, ctx.Period)
			break
		}
	}

	filepath, err := exportFile(ctx, strMatrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("export node %s %s failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genStrMatrix(values [][]interface{}, period *TimePeriodParams) [][]string {
	var strMatrix [][]string
	for i := period.Begin; i <= period.End; i += period.Step {
		strMatrix = append(strMatrix, append([]string{time.Unix(int64(i), 0).Format(util.TimeFormat)}, "0"))
	}

	for _, vs := range values {
		if timestamp, value := getTimestampAndValue(vs); timestamp != 0 && timestamp >= period.Begin {
			strMatrix[(timestamp-period.Begin)/period.Step][1] = value
		}
	}

	return strMatrix
}
