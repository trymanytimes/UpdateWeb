package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/metric/resource"
)

var (
	TableNode       = restdb.ResourceDBType(&resource.Node{})
	instance        = "instance"
	device          = "device"
	docker          = "docker"
	dockerInterface = "veth"
	schema          = "http://"
)

const DefaultPeriodValue = "6"

type NodeHandler struct {
	prometheusAddr string
	exportPort     int
	httpClient     *http.Client
}
type TimePeriodParams struct {
	Begin int64
	End   int64
	Step  int64
}

func getTimePeriodParamFromFilter(filters []restresource.Filter) *TimePeriodParams {
	periodStr := getPeriodFromFilters(filters)
	period, _ := strconv.Atoi(periodStr)

	return genTimePeriodParams(period)
}
func getPeriodFromFilters(filters []restresource.Filter) string {
	for _, filter := range filters {
		if filter.Name == "period" && filter.Modifier == restresource.Eq {
			for _, value := range filter.Values {
				switch value {
				case "6", "12", "24", "168", "720", "2160":
					return value
				}
			}
		}
	}

	return DefaultPeriodValue
}

func genTimePeriodParams(period int) *TimePeriodParams {
	now := time.Now().Unix()
	return &TimePeriodParams{
		Begin: now - int64(period*3600),
		End:   now,
		Step:  int64(period * 12),
	}
}

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

func NewNodeHandler(conf *config.DDIControllerConfig, cli *http.Client) *NodeHandler {
	h := &NodeHandler{
		prometheusAddr: conf.Prometheus.Addr,
		httpClient:     cli,
		exportPort:     conf.Prometheus.ExportPort,
	}
	//go h.monitor()
	return h
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

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var nodes []*resource.Node
	if err := db.GetResources(map[string]interface{}{"orderby": "roles, master"}, &nodes); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list nodes from db failed: %s", err.Error()))
	}

	if err := h.setNodeMetrics(getNodeAddrsLabel(nodes, h.exportPort), nodes, getTimePeriodParamFromFilter(ctx.GetFilters())); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list nodes metrics from prometheus failed: %s", err.Error()))
	}

	return nodes, nil
}

func (h *NodeHandler) setNodeMetrics(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	if err := h.getMemoryRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes memory metrics from prometheus failed: %s", err.Error())
	}
	if err := h.getCPURatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes cpu ratio metrics from prometheus failed: %s", err.Error())
	}

	if err := h.getDiscRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes disk ratio metric from prometheus failed: %s", err.Error())
	}

	if err := h.getNetworkRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes network metric from prometheus failed: %s", err.Error())
	}
	for i, node := range nodes {
		if !node.NodeIsAlive {
			continue
		}

		if len(node.CpuUsage) >= 1 {
			nodes[i].CpuRatio = node.CpuUsage[len(node.CpuUsage)-1].Ratio
		}
		if len(node.MemoryUsage) >= 1 {
			nodes[i].MemRatio = node.MemoryUsage[len(node.MemoryUsage)-1].Ratio
		}
	}
	return nil
}

func getNodeAddrsLabel(nodes []*resource.Node, port int) string {
	var nodesAddrs []string
	for _, node := range nodes {
		nodesAddrs = append(nodesAddrs, node.GetID()+":"+strconv.Itoa(port))
	}
	return strings.Join(nodesAddrs, "|")
}

func (h *NodeHandler) getMemoryRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "1 - (node_memory_MemFree_bytes{instance=~\"" + nodesAddrLabel + "\"}+node_memory_Cached_bytes{instance=~\"" +
		nodesAddrLabel + "\"}+node_memory_Buffers_bytes{instance=~\"" + nodesAddrLabel + "\"}) / node_memory_MemTotal_bytes"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveMemoryValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getCPURatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "1 - (avg(irate(node_cpu_seconds_total{instance=~\"" + nodesAddrLabel + "\", mode=\"idle\"}[5m])) by (instance))"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveCPUValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getDiscRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pqlFree := "node_filesystem_free_bytes{instance=~\"" + nodesAddrLabel +
		"\", fstype=~\"ext4|xfs\"}"
	respFree, err := h.getPrometheusData(pqlFree, period)
	if err != nil {
		return err
	}

	pqlTotal := "node_filesystem_size_bytes{instance=~\"" + nodesAddrLabel +
		"\", fstype=~\"ext4|xfs\"}"
	respTotal, err := h.getPrometheusData(pqlTotal, period)
	if err != nil {
		return err
	}
	if err := h.resolveDiscValues(nodes, respFree, respTotal, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getNetworkRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "irate(node_network_receive_bytes_total{device!=\"lo\", instance=~\"" + nodesAddrLabel +
		"\"}[5m]) + irate(node_network_transmit_bytes_total{device!=\"lo\", instance=~\"" + nodesAddrLabel + "\"}[5m])"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveNetworkValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getPrometheusData(pql string, period *TimePeriodParams) (*PrometheusResponse, error) {
	param := url.Values{}
	param.Add("query", pql)
	left := fmt.Sprintf("&start=%d&end=%d&step=%d", period.Begin, period.End, period.Step)
	path := schema + h.prometheusAddr + "/api/v1/query_range?" + param.Encode() + left
	httpResp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp PrometheusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal pronmetheus response failed: %s", err.Error())
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("get metric failed with status: %s", resp.Status)
	}

	return &resp, nil
}

func (h *NodeHandler) resolveCPUValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			node.CpuUsage = getRatiosWithTimestamp(r.Values, period)
		}
	}
	return nil
}

func (h *NodeHandler) resolveDiscValues(nodes []*resource.Node, respFree *PrometheusResponse, respTotal *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		var dataFree [][]resource.RatioWithTimestamp
		var dataUsed [][]resource.RatioWithTimestamp
		for _, r := range respFree.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			dataFree = append(dataFree, getRatiosWithTimestamp(r.Values, period))
		}
		for _, r := range respTotal.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			dataUsed = append(dataUsed, getRatiosWithTimestamp(r.Values, period))
		}
		if len(dataUsed) != len(dataFree) {
			return fmt.Errorf("no data got or data not correct")
		}
		if len(dataUsed) == 0 || len(dataFree) == 0 {
			continue
		}
		for j := 0; j < len(dataFree[0]); j++ {
			tmp := resource.RatioWithTimestamp{Timestamp: dataFree[0][j].Timestamp}
			var sumFree float64
			var sumUsed float64
			for i := 0; i < len(dataFree); i++ {
				newValue, err := strconv.ParseFloat(dataFree[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sumFree += newValue
			}
			for i := 0; i < len(dataUsed); i++ {
				newValue, err := strconv.ParseFloat(dataUsed[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sumUsed += newValue
			}
			if sumUsed != 0 {
				tmp.Ratio = fmt.Sprintf("%.2f", (sumUsed-sumFree)/sumUsed)
			} else {
				tmp.Ratio = "0"
			}
			node.DiscUsage = append(node.DiscUsage, tmp)
		}
	}
	return nil
}

func (h *NodeHandler) resolveMemoryValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			node.MemoryUsage = getRatiosWithTimestamp(r.Values, period)
		}
	}
	return nil
}

func (h *NodeHandler) resolveNetworkValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		var data [][]resource.RatioWithTimestamp
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] || strings.Index(r.MetricLabels[device], docker) > 0 ||
				strings.Index(r.MetricLabels[device], dockerInterface) > 0 {
				continue
			}
			data = append(data, getRatiosWithTimestamp(r.Values, period))
		}
		if len(data) == 0 {
			continue
		}
		for j := 0; j < len(data[0]); j++ {
			var tmp resource.RatioWithTimestamp
			tmp.Timestamp = data[0][j].Timestamp
			var sum float64
			for i := 0; i < len(data); i++ {
				newValue, err := strconv.ParseFloat(data[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sum += newValue
			}
			tmp.Ratio = fmt.Sprintf("%.2f", sum)
			node.Network = append(node.Network, tmp)
		}
	}

	return nil
}

func (h *NodeHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ip := ctx.Resource.(*resource.Node).GetID()
	var nodes []*resource.Node
	_, err := restdb.GetResourceWithID(db.GetDB(), ip, &nodes)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s from db failed: %s", ip, err.Error()))
	}

	if err := h.setNodeMetrics(getNodeAddrsLabel(nodes, h.exportPort), nodes, getTimePeriodParamFromFilter(ctx.GetFilters())); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get node %s metrics from prometheus failed: %s", ip, err.Error()))
	}

	return nodes[0], nil
}
