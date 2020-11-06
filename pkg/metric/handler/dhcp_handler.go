package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

const DefaultPeriodValue = "6"

type MetricContext struct {
	Client            *http.Client
	NodeIP            string
	ElasticsearchAddr string
	PrometheusAddr    string
	MetricName        string
	MetricLabel       string
	TableHeader       []string
	PeriodBegin       string
	Period            *TimePeriodParams
	AggsName          string
	AggsKeyword       string
}

type TimePeriodParams struct {
	Begin int64
	End   int64
	Step  int64
}

type DhcpHandler struct {
	prometheusAddr string
	httpClient     *http.Client
}

func NewDhcpHandler(conf *config.DDIControllerConfig, cli *http.Client) *DhcpHandler {
	return &DhcpHandler{
		prometheusAddr: conf.Prometheus.Addr,
		httpClient:     cli,
	}
}

func (h *DhcpHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	context := h.genDHCPMetricContext(ctx.Resource.GetParent().GetID(), getTimePeriodParamFromFilter(ctx.GetFilters()))
	lps, err := getLps(context)
	if err != nil {
		return nil, err
	}

	lease, err := getLease(context)
	if err != nil {
		return nil, err
	}

	packets, err := getPackets(context)
	if err != nil {
		return nil, err
	}

	subnetUsedRatios, err := getSubnetUsedRatios(context)
	if err != nil {
		return nil, err
	}

	return []*resource.Dhcp{lps, lease, packets, subnetUsedRatios}, nil
}

func (h *DhcpHandler) genDHCPMetricContext(nodeIP string, period *TimePeriodParams) *MetricContext {
	return &MetricContext{
		Client:         h.httpClient,
		PrometheusAddr: h.prometheusAddr,
		NodeIP:         nodeIP,
		Period:         period,
	}
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

func (h *DhcpHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpID := ctx.Resource.GetID()
	context := h.genDHCPMetricContext(ctx.Resource.GetParent().GetID(), getTimePeriodParamFromFilter(ctx.GetFilters()))
	switch dhcpID {
	case resource.ResourceIDLPS:
		return getLps(context)
	case resource.ResourceIDLease:
		return getLease(context)
	case resource.ResourceIDPackets:
		return getPackets(context)
	case resource.ResourceIDSubnetUsedRatios:
		return getSubnetUsedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dhcp resource %s", dhcpID))
	}
}

func (h *DhcpHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *DhcpHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	period, err := getTimePeriodParamFromActionInput(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("parse action input failed: %s", err.Error()))
	}

	context := h.genDHCPMetricContext(ctx.Resource.GetParent().GetID(), period)
	dhcpID := ctx.Resource.GetID()
	switch dhcpID {
	case resource.ResourceIDLPS:
		return exportLps(context)
	case resource.ResourceIDLease:
		return exportLease(context)
	case resource.ResourceIDPackets:
		return exportPackets(context)
	case resource.ResourceIDSubnetUsedRatios:
		return exportSubnetUsedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dhcp resource %s", dhcpID))
	}
}

func getTimePeriodParamFromActionInput(ctx *restresource.Context) (*TimePeriodParams, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if ok == false {
		return nil, fmt.Errorf("action exportcsv input invalid")
	}

	switch filter.Period {
	case 6, 12, 24, 168, 720, 2160:
	default:
		return nil, fmt.Errorf("action exportcsv param period not in [6, 12, 24, 168, 720, 2160]")
	}

	return genTimePeriodParams(filter.Period), nil
}
