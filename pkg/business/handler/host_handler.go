package handler

import (
	"context"
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbHost "github.com/trymanytimes/UpdateWeb/pkg/proto/ateStatsHomePage"
	pbCluster "github.com/trymanytimes/UpdateWeb/pkg/proto/ate_cluster"
)

type HostHandler struct{}

func NewHostHandler() *HostHandler {
	return &HostHandler{}
}

func (h *HostHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	device := ctx.Resource.(*resource.Host)
	cli := grpcclient.GetGrpcClient()
	//query wether exists a cluster
	req := pbHost.ShowHomePageFlowDataReq{DeviceId: device.GetID()}
	resp, err := cli.MonitorClient.ShowHomePageFlowData(context.Background(), &req)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec ShowHomePageFlowData failed: %s", err.Error()))
	}
	device.V4UpFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV4UpBytes
	device.V4DownFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV4DownBytes
	device.V6UpFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV6UpBytes
	device.V6DownFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV6DownBytes
	device.TimeStamp = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].Timestamp

	return device, nil
}
func (h *HostHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	cli := grpcclient.GetGrpcClient()
	clusterIDReq := pbCluster.ClusterIDReq{ClusterId: DefaultClusterID}
	defaultCluster, err := cli.ClusterClient.QryOneCluster(context.Background(), &clusterIDReq)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec SetCluster failed: %s", err.Error()))
	}
	var devices []*resource.Host
	for _, v := range defaultCluster.SocsInfo.GetNodeHost() {
		req := pbHost.ShowHomePageFlowDataReq{DeviceId: v.GetHostId()}
		resp, err := cli.MonitorClient.ShowHomePageFlowData(context.Background(), &req)
		if err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec ShowHomePageFlowData failed: %s", err.Error()))
		}
		var device resource.Host
		device.V4UpFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV4UpBytes
		device.V4DownFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV4DownBytes
		device.V6UpFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV6UpBytes
		device.V6DownFlow = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].CurrentDeviceV6DownBytes
		device.TimeStamp = resp.GetUpDowmFlow()[len(resp.UpDowmFlow)-1].Timestamp
		devices = append(devices, &device)
		device.SetID(v.GetHostId())
	}
	return devices, nil
}
