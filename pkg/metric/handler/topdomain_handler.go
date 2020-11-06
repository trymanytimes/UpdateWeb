package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

const (
	TopDomainKeyWord    = "domain.keyword"
	TopDomainAggsName   = "top10Domains"
	MetricNameTopDomain = "lx_dns_top10_domains"
)

var TableHeaderTopDomain = []string{"请求源域名", "请求次数"}

func getTopTenDomains(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.AggsName = TopDomainAggsName
	ctx.AggsKeyword = TopDomainKeyWord
	resp, err := requestElasticsearch(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s top ten domains from elasticsearch failed: %s", ctx.NodeIP, err.Error()))
	}

	var topDomains []resource.TopDomain
	if tops, ok := resp.Aggregations[TopDomainAggsName]; ok {
		for _, bucket := range tops.Buckets {
			topDomains = append(topDomains, resource.TopDomain{
				Domain: bucket.Key,
				Count:  bucket.DocCount,
			})
		}
	}

	dns := &resource.Dns{TopTenDomains: topDomains}
	dns.SetID(resource.ResourceIDTopTenDomains)
	return dns, nil
}

func exportTopTenDomains(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameTopDomain
	ctx.TableHeader = TableHeaderTopDomain
	ctx.AggsName = TopDomainAggsName
	ctx.AggsKeyword = TopDomainKeyWord
	return exportTopMetrics(ctx)
}
