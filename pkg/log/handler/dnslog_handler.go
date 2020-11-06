package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/log/resource"
	metricHandler "github.com/linkingthing/ddi-controller/pkg/metric/handler"
	metricRes "github.com/linkingthing/ddi-controller/pkg/metric/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
	monitorConf "github.com/linkingthing/ddi-monitor/config"
)

const (
	DefaultDNSLogValidPeriod = 180 //day
	DNSLogTimeFormat         = "yyyy-MM-dd HH:mm:ss"
	FilterTimeFrom           = "from"
	FilterTimeTo             = "to"
	DNSLogESQuery            = "http://%s/%s/query/_search"
	postMethod               = "POST"
	TimeHeader               = "时间"
	NodeIPHeader             = "节点ip"
	ContentHeader            = "内容"
	CSVFilePathWithNodeID    = "/opt/website/dns-log-%s-%s.csv"
	CSVFilePathWithoutNodeID = "/opt/website/dns-log-%s.csv"
	UTF8BOM                  = "\xEF\xBB\xBF"
	timeLayout               = "2006-01-02" //format must be 612345
	timeLayoutUTC            = "2006-01-02T15:04:05.000Z"
	ESQueryTimeFormat        = "2006-01-02 15:04:05"
	ListTimeFormat           = "2006-01-02T15:04:05+08:00"
)

type ElasticSearchRequest struct {
	Size  int                 `json:"size"`
	Query ElasticSearchQuery  `json:"query"`
	Sort  []ElasticSearchSort `json:"sort"`
}

type ElasticSearchQuery struct {
	BoolCondition ElasticSearchBool `json:"bool"`
}

type ElasticSearchBool struct {
	Should []ElasticSearchShould `json:"should"`
	Must   *ElasticSearchMust    `json:"must,omitempty"`
}

type ElasticSearchMust struct {
	Match MatchDestIP `json:"match"`
}

type MatchDestIP struct {
	DestIP string `json:"dest_ip"`
}

type ElasticSearchShould struct {
	Range ElasticSearchRange `json:"range"`
}

type ElasticSearchRange struct {
	Timestamp TimestampKeyword `json:"@timestamp"`
}

type TimestampKeyword struct {
	GTE    string `json:"gte"`
	LTE    string `json:"lte"`
	Format string `json:"format"`
}

type ElasticSearchSort struct {
	TimestampOrder ESTimestampOrder `json:"@timestamp"`
}

type ESTimestampOrder struct {
	Order string `json:"order"`
}

type ElasticSearchResponse struct {
	Timeout bool                 `json:"time_out"`
	Hits    ElasticsearchHistory `json:"hits"`
}

type ElasticsearchHistory struct {
	Hits []HistoryData `json:"hits"`
}

type HistoryData struct {
	Source SourceData `json:"_source"`
}

type SourceData struct {
	TimeStamp   string `json:"@timestamp"`
	DestIP      string `json:"dest_ip"`
	Domain      string
	SourceIP    string `json:"src_ip"`
	SourcePort  string `json:"src_port"`
	ResolveType string `json:"type"`
	View        string `json:"view"`
}

type TimePeriodParams struct {
	Begin int64
	End   int64
	Step  int64
}

var DNSLogFilterNames = []string{"node_ip"}

type DNSLogHandler struct {
	elasticsearchAddr string
	prometheusAddr    string
	httpClient        *http.Client
}

type DNSLogContext struct {
	Client            *http.Client
	NodeIPs           []string
	ElasticsearchAddr string
	TableHeader       []string
	InputIP           string
}

func NewDNSLogHandler(conf *config.DDIControllerConfig, cli *http.Client) *DNSLogHandler {
	return &DNSLogHandler{
		elasticsearchAddr: conf.Elasticsearch.Addr,
		prometheusAddr:    conf.Prometheus.Addr,
		httpClient:        cli,
	}
}

func (h *DNSLogHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	reqBody, err := h.GenPostBodyJsonByFileters(DefaultAuditLogValidPeriod, ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list dnslog faield: %s", err.Error()))
	}
	context, err := h.CreateDNSLogContext()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("CreateDNSLogContext faield: %s", err.Error()))
	}
	resp, err := h.RequestElasticSearch(*context, reqBody)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("RequestElasticSearch faield: %s", err.Error()))
	}

	var dnsLogs []*resource.DNSLog
	for _, his := range resp.(*ElasticSearchResponse).Hits.Hits {
		dnsLogs = append(dnsLogs, &resource.DNSLog{
			Time:    his.Source.TimeStamp,
			NodeIP:  his.Source.DestIP,
			Content: his.Source.SourceIP + " " + his.Source.SourcePort + " " + his.Source.View + " " + his.Source.Domain + " " + his.Source.ResolveType,
		})
	}
	return dnsLogs, nil
}

func (h *DNSLogHandler) CreateDNSLogContext() (*DNSLogContext, error) {
	context := DNSLogContext{
		Client:            h.httpClient,
		ElasticsearchAddr: h.elasticsearchAddr,
		TableHeader:       []string{TimeHeader, NodeIPHeader, ContentHeader},
	}
	var nodes interface{}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var err error
		nodes, err = tx.Get(metricHandler.TableNode, nil)
		if err != nil {
			return fmt.Errorf("Get All Nodes from db Error:%s", err.Error())
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for _, node := range nodes.([]*metricRes.Node) {
		if node.Master == "" {
			for _, r := range node.Roles {
				if r == string(monitorConf.ServiceRoleDNS) {
					context.NodeIPs = append(context.NodeIPs, node.Ip)
				}
			}
		}
	}
	return &context, nil
}

func (h *DNSLogHandler) GenPostBodyJsonByFileters(from int, filters []restresource.Filter) (*ElasticSearchRequest, error) {
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -from).Format(ESQueryTimeFormat)
	timeTo := now.Format(ESQueryTimeFormat)
	for _, filter := range filters {
		switch filter.Name {
		case FilterTimeFrom:
			if from, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				timeIput, err := time.Parse(timeLayout, from)
				if err != nil {
					return nil, fmt.Errorf("from %s time format is not correct, format should be like %s, err:%s", from, timeLayout, err.Error())
				}
				timeFrom = timeIput.Format(ESQueryTimeFormat)
			}
		case FilterTimeTo:
			if to, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				timeIput, err := time.Parse(timeLayout, to)
				if err != nil {
					return nil, fmt.Errorf("to %s time format is not correct, format should be like %s, err:%s", to, timeLayout, err.Error())
				}
				timeIput = timeIput.AddDate(0, 0, 1)
				timeTo = timeIput.Format(ESQueryTimeFormat)
			}
		}
	}

	var nodeip string
	for _, filter := range filters {
		if slice.SliceIndex(DNSLogFilterNames, filter.Name) != -1 {
			if value, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				nodeip = value
			}
		}
	}

	req := ElasticSearchRequest{Size: 10000}
	if nodeip != "" {
		req.Query.BoolCondition.Should = append(req.Query.BoolCondition.Should,
			ElasticSearchShould{Range: ElasticSearchRange{
				Timestamp: TimestampKeyword{GTE: timeFrom, LTE: timeTo, Format: DNSLogTimeFormat}}})
		req.Query.BoolCondition.Must = &ElasticSearchMust{Match: MatchDestIP{nodeip}}
	} else {
		req.Query.BoolCondition.Should = append(req.Query.BoolCondition.Should,
			ElasticSearchShould{Range: ElasticSearchRange{
				Timestamp: TimestampKeyword{GTE: timeFrom, LTE: timeTo, Format: DNSLogTimeFormat}}})
	}
	req.Sort = append(req.Sort, ElasticSearchSort{TimestampOrder: ESTimestampOrder{Order: "desc"}})

	return &req, nil
}

func (h *DNSLogHandler) RequestElasticSearch(context DNSLogContext, reqBody interface{}) (interface{}, error) {
	var nodeList string
	for i, ip := range context.NodeIPs {
		nodeList += "dns_" + ip
		if i != len(context.NodeIPs)-1 {
			nodeList += ","
		}
	}

	var rsp ElasticSearchResponse
	err := httpRequest(context.Client, postMethod, fmt.Sprintf(DNSLogESQuery, context.ElasticsearchAddr, nodeList), reqBody, &rsp)
	if err != nil {
		return "", fmt.Errorf("http request %s to elasticsearch error:%s", fmt.Sprintf(DNSLogESQuery, context.ElasticsearchAddr, nodeList), err.Error())
	}
	if rsp.Timeout == true {
		return "", fmt.Errorf("request %s: get the elasticsearch data timeout", fmt.Sprintf(DNSLogESQuery, context.ElasticsearchAddr, nodeList))
	}
	return &rsp, nil
}

func httpRequest(cli *http.Client, httpMethod, url string, req, resp interface{}) error {
	var httpReqBody io.Reader
	if req != nil {
		reqBody, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request failed: %s", err.Error())
		}

		httpReqBody = bytes.NewBuffer(reqBody)
	}

	httpReq, err := http.NewRequest(httpMethod, url, httpReqBody)
	if err != nil {
		return fmt.Errorf("new http request failed: %s", err.Error())
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := cli.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send http request failed: %s", err.Error())
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read http response body failed: %s", err.Error())
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("unmarshal http response failed: %s", err.Error())
	}

	return nil
}

func (h *DNSLogHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *DNSLogHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	//get the input
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.ServerError, "action exportcsv input invalid")
	}
	//get the response data
	reqBody, err := h.GenPostBodyJsonByFileters(DefaultAuditLogValidPeriod, ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list dnslog faield: %s", err.Error()))
	}
	context, err := h.CreateDNSLogContext()
	context.InputIP = filter.NodeIP
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("CreateDNSLogContext faield: %s", err.Error()))
	}
	resp, err := h.RequestElasticSearch(*context, reqBody)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("RequestElasticSearch faield: %s", err.Error()))
	}
	//write file
	var contents [][]string
	for _, his := range resp.(*ElasticSearchResponse).Hits.Hits {
		timeIput, err := time.Parse(timeLayoutUTC, his.Source.TimeStamp)
		var startTime string
		if err == nil {
			startTime = timeIput.Local().Format(ESQueryTimeFormat)
		}
		contents = append(contents, []string{
			startTime,
			his.Source.DestIP,
			his.Source.SourceIP + " " + his.Source.SourcePort + " " + his.Source.View + " " + his.Source.Domain + " " + his.Source.ResolveType,
		})
	}
	filePath, err := exportFile(context, contents)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list dnslog exportFile faield: %s", err.Error()))
	}
	return &resource.FileInfo{Path: filePath}, nil
}

func (h *DNSLogHandler) GenPostBodyJsonByActionInput(from int, filters *resource.ExportFilter) (*ElasticSearchRequest, error) {
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -from).Format(ESQueryTimeFormat)
	timeTo := now.Format(ESQueryTimeFormat)
	if filters.From != "" {
		timeIput, err := time.Parse(timeLayout, filters.From)
		if err != nil {
			return nil, fmt.Errorf("from %s time format is not correct, format should be like %s, err:%s", from, timeLayout, err.Error())
		}
		timeFrom = timeIput.Format(ESQueryTimeFormat)
	}
	if filters.To != "" {
		timeIput, err := time.Parse(timeLayout, filters.To)
		if err != nil {
			return nil, fmt.Errorf("to %s time format is not correct, format should be like %s, err:%s", filters.To, timeLayout, err.Error())
		}
		timeIput = timeIput.AddDate(0, 0, 1)
		timeTo = timeIput.Format(ESQueryTimeFormat)
	}

	req := ElasticSearchRequest{Size: 10000}
	req.Query.BoolCondition.Should = append(req.Query.BoolCondition.Should,
		ElasticSearchShould{Range: ElasticSearchRange{Timestamp: TimestampKeyword{GTE: timeFrom, LTE: timeTo, Format: DNSLogTimeFormat}}})
	req.Sort = append(req.Sort, ElasticSearchSort{TimestampOrder: ESTimestampOrder{Order: "desc"}})
	return &req, nil
}

func exportFile(ctx *DNSLogContext, contents [][]string) (string, error) {
	var filepath string
	if ctx.InputIP == "" {
		filepath = fmt.Sprintf(CSVFilePathWithoutNodeID, time.Now().Format(time.RFC3339))
	} else {
		filepath = fmt.Sprintf(CSVFilePathWithNodeID, ctx.InputIP, time.Now().Format(time.RFC3339))
	}
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("create dns log csv file failed: %s", err.Error())
	}

	defer file.Close()
	file.WriteString(UTF8BOM)
	w := csv.NewWriter(file)
	if err := w.Write(ctx.TableHeader); err != nil {
		return "", fmt.Errorf("write dns log's header %s to csv file failed: %s", ctx.TableHeader, err.Error())
	}

	if err := w.WriteAll(contents); err != nil {
		return "", fmt.Errorf("write dns log's content data to csv file failed: %s", err.Error())
	}

	w.Flush()
	return filepath, nil
}
