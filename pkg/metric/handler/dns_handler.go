package handler

import (
	"fmt"
	"net/http"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

type DnsHandler struct {
	elasticsearchAddr string
	prometheusAddr    string
	httpClient        *http.Client
}

func NewDnsHandler(conf *config.DDIControllerConfig, cli *http.Client) *DnsHandler {
	return &DnsHandler{
		elasticsearchAddr: conf.Elasticsearch.Addr,
		prometheusAddr:    conf.Prometheus.Addr,
		httpClient:        cli,
	}
}

func (h *DnsHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	context := h.genDNSMetricContext(ctx.Resource.GetParent().GetID(), getTimePeriodParamFromFilter(ctx.GetFilters()))
	topTenIps, err := getTopTenIps(context)
	if err != nil {
		return nil, err
	}

	topTenDomains, err := getTopTenDomains(context)
	if err != nil {
		return nil, err
	}

	qps, err := getQps(context)
	if err != nil {
		return nil, err
	}

	cachehit, err := getCacheHitRatio(context)
	if err != nil {
		return nil, err
	}

	queryTypeRatios, err := getQueryTypeRatios(context)
	if err != nil {
		return nil, err
	}

	resolvedRatios, err := getResolvedRatios(context)
	if err != nil {
		return nil, err
	}

	return []*resource.Dns{topTenIps, topTenDomains, qps, cachehit, queryTypeRatios, resolvedRatios}, nil
}

func (h *DnsHandler) genDNSMetricContext(nodeIP string, period *TimePeriodParams) *MetricContext {
	return &MetricContext{
		Client:            h.httpClient,
		PrometheusAddr:    h.prometheusAddr,
		ElasticsearchAddr: h.elasticsearchAddr,
		NodeIP:            nodeIP,
		Period:            period,
	}
}

func (h *DnsHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dnsID := ctx.Resource.GetID()
	context := h.genDNSMetricContext(ctx.Resource.GetParent().GetID(), getTimePeriodParamFromFilter(ctx.GetFilters()))
	switch dnsID {
	case resource.ResourceIDTopTenIPs:
		return getTopTenIps(context)
	case resource.ResourceIDTopTenDomains:
		return getTopTenDomains(context)
	case resource.ResourceIDQPS:
		return getQps(context)
	case resource.ResourceIDCacheHitRatio:
		return getCacheHitRatio(context)
	case resource.ResourceIDQueryTypeRatios:
		return getQueryTypeRatios(context)
	case resource.ResourceIDResolvedRatios:
		return getResolvedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dns resource %s", dnsID))
	}
}

func (h *DnsHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *DnsHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	period, err := getTimePeriodParamFromActionInput(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("parse action input failed: %s", err.Error()))
	}

	dnsID := ctx.Resource.GetID()
	context := h.genDNSMetricContext(ctx.Resource.GetParent().GetID(), period)
	switch dnsID {
	case resource.ResourceIDTopTenIPs:
		return exportTopTenIps(context)
	case resource.ResourceIDTopTenDomains:
		return exportTopTenDomains(context)
	case resource.ResourceIDQPS:
		return exportQps(context)
	case resource.ResourceIDCacheHitRatio:
		return exportCacheHitRatio(context)
	case resource.ResourceIDQueryTypeRatios:
		return exportQueryTypeRatios(context)
	case resource.ResourceIDResolvedRatios:
		return exportResolvedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dns resource %s", dnsID))
	}
}
