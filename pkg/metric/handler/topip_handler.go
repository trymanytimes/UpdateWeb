package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

type ElasticsearchRequest struct {
	Size  uint32                       `json:"size"`
	Query ElasticsearchQuery           `json:"query"`
	Aggs  map[string]ElasticsearchAggs `json:"aggs"`
}

type ElasticsearchQuery struct {
	Range ElasticsearchQueryRange `json:"range"`
}

type ElasticsearchQueryRange struct {
	Timestamp ElasticsearchQueryRangeTimestamp `json:"@timestamp"`
}

type ElasticsearchQueryRangeTimestamp struct {
	From string `json:"from"`
}

type ElasticsearchAggs struct {
	Term ElasticsearchAggsTerm `json:"terms"`
}

type ElasticsearchAggsTerm struct {
	Field string            `json:"field"`
	Order map[string]string `json:"order"`
	Size  uint32            `json:"size"`
}

type ElasticsearchResponse struct {
	Timeout      bool                   `json:"time_out"`
	Aggregations map[string]Aggregation `json:"aggregations"`
}

type Aggregation struct {
	Buckets []Bucket `json:"buckets"`
}

type Bucket struct {
	Key      string `json:"key"`
	DocCount uint64 `json:"doc_count"`
}

var (
	ElasticsearchQueryUrl = "http://%s/dns_%s/_search"
	AggsTermOrder         = map[string]string{"_count": "desc"}
	TableHeaderTopIP      = []string{"请求源地址", "请求次数"}
)

const (
	HttpMethodPOST  = "POST"
	TopIpKeyWord    = "src_ip.keyword"
	TopIpAggsName   = "top10Ips"
	MetricNameTopIp = "lx_dns_top10_ips"
)

func getTopTenIps(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.AggsName = TopIpAggsName
	ctx.AggsKeyword = TopIpKeyWord
	resp, err := requestElasticsearch(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s top ten ips from elasticsearch failed: %s", ctx.NodeIP, err.Error()))
	}

	var topIps []resource.TopIp
	if tops, ok := resp.Aggregations[TopIpAggsName]; ok {
		for _, bucket := range tops.Buckets {
			topIps = append(topIps, resource.TopIp{
				Ip:    bucket.Key,
				Count: bucket.DocCount,
			})
		}
	}

	dns := &resource.Dns{TopTenIps: topIps}
	dns.SetID(resource.ResourceIDTopTenIPs)
	return dns, nil
}

func requestElasticsearch(ctx *MetricContext) (ElasticsearchResponse, error) {
	var resp ElasticsearchResponse
	if err := httpRequest(ctx.Client, HttpMethodPOST, fmt.Sprintf(ElasticsearchQueryUrl, ctx.ElasticsearchAddr, ctx.NodeIP),
		&ElasticsearchRequest{
			Size: 0,
			Query: ElasticsearchQuery{
				Range: ElasticsearchQueryRange{
					Timestamp: ElasticsearchQueryRangeTimestamp{
						From: "now-" + strconv.Itoa(int(ctx.Period.Begin)) + "h",
					},
				},
			},
			Aggs: map[string]ElasticsearchAggs{
				ctx.AggsName: ElasticsearchAggs{
					Term: ElasticsearchAggsTerm{
						Field: ctx.AggsKeyword,
						Order: AggsTermOrder,
						Size:  10,
					},
				},
			},
		}, &resp); err != nil {
		return resp, fmt.Errorf("get %s from elasticsearch failed: %s", ctx.AggsKeyword, err.Error())
	}

	if resp.Timeout {
		log.Warnf("get %s from elasticsearch timeout", ctx.AggsKeyword)
		return resp, nil
	}

	return resp, nil
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

func exportTopTenIps(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameTopIp
	ctx.TableHeader = TableHeaderTopIP
	ctx.AggsName = TopIpAggsName
	ctx.AggsKeyword = TopIpKeyWord
	return exportTopMetrics(ctx)
}

func exportTopMetrics(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := requestElasticsearch(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("export node %s %s to csv file failed: %s", ctx.NodeIP, ctx.AggsKeyword, err.Error()))
	}

	var matrix [][]string
	if tops, ok := resp.Aggregations[ctx.AggsName]; ok {
		for _, bucket := range tops.Buckets {
			matrix = append(matrix, []string{bucket.Key, strconv.FormatUint(bucket.DocCount, 10)})
		}
	}

	filepath, err := exportFile(ctx, matrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export node %s %s to csv file failed: %s", ctx.NodeIP, ctx.AggsKeyword, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}
