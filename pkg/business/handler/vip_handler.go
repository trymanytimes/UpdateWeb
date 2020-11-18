package handler

import (
	"context"
	"fmt"
	"strconv"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbWeb "github.com/trymanytimes/UpdateWeb/pkg/proto/rcs"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

var (
	FilterAvailableIPCount = "availableipcount"
)

type VipHandler struct{}

func NewVipHandler() *VipHandler {
	return &VipHandler{}
}

func (h *VipHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	cli := grpcclient.GetGrpcClient()
	var rspips []*resource.VipInterval
	for _, filter := range ctx.GetFilters() {
		switch filter.Name {
		case FilterAvailableIPCount:
			if count, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				icount, err := strconv.Atoi(count)
				if err != nil {
					return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("count is not correct, err:%s", err.Error()))
				}
				req := pbWeb.GetAvailIPReq{StrclusterId: DefaultClusterID, Network: 1, Count: int32(icount)}
				iprsp, err := cli.WebsiteClient.GetRaltAvailIP(context.Background(), &req)
				if err != nil {
					return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec GetRaltAvailIP failed: %s", err.Error()))
				}
				for _, v := range iprsp.Ip {
					rspips = append(rspips, &resource.VipInterval{BeginVip: v, EndVip: v, Length: 96})
				}
			}
		}
	}
	return rspips, nil
}
