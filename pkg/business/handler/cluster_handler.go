package handler

import (
	"context"
	"fmt"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/business/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	pbCluster "github.com/trymanytimes/UpdateWeb/pkg/proto/ate_cluster"
)

var (
	DefaultClusterID   = "001"
	DefaultClusterName = "c001"
	OperTypeCreate     = int32(1)
	OperTypeModify     = int32(2)
	OperTypeDelete     = int32(3)
	ClusterType        = "6ATE"
	BalanceTypeHash    = int32(1)
	On                 = "on"
	Off                = "Off"
	SwitchUp           = int32(1)
	SwitchDown         = int32(2)
	Localhost          = "127.0.0.1"
	LogPort            = int32(50000)
)

type ClusterHandler struct{}

func NewClusterHandler() (*ClusterHandler, error) {
	cli := grpcclient.GetGrpcClient()
	//query wether exists a cluster
	clusterIDReq := pbCluster.ClusterIDReq{ClusterId: DefaultClusterID}
	defaultCluster, err := cli.ClusterClient.QryOneCluster(context.Background(), &clusterIDReq)
	if err != nil {
		return nil, log.Errorf("grpc service exec SetCluster failed: %s", err.Error())
	}
	if defaultCluster.SocsInfo.ClusterName == "" {
		//create a new cluster
		var clusterInfo pbCluster.ClusterPublicInfoReq
		clusterInfo.ClusterId = DefaultClusterID
		clusterInfo.OperType = OperTypeCreate
		//Balance info
		clusterInfo.BalanceInfo = &pbCluster.ClusterBalanceInfo{}
		clusterInfo.BalanceInfo.ClusterName = DefaultClusterName
		clusterInfo.BalanceInfo.ClusterType = ClusterType
		//get availavle device
		devRsp, err := cli.ClusterClient.QryFreeDevice(context.Background(), &pbCluster.QryFreeDeviceReq{})
		if err != nil {
			return nil, log.Errorf("grpc service exec SetCluster failed: %s", err.Error())
		}
		for _, v := range devRsp.HostId {
			clusterInfo.BalanceInfo.NodeHost = append(clusterInfo.BalanceInfo.NodeHost, &pbCluster.NodeHost{
				HostId: v,
				NodeId: v,
			})
		}

		clusterInfo.BalanceInfo.BalanceType = BalanceTypeHash
		clusterInfo.BalanceInfo.Ipv6Vip = append(clusterInfo.BalanceInfo.Ipv6Vip, &pbCluster.VipInterval{
			BeginVip: config.GetConfig().VIP.BeginVIP,
			EndVip:   config.GetConfig().VIP.EndVIP,
			Length:   config.GetConfig().VIP.Length,
		})
		//Application info
		clusterInfo.AppInfo = &pbCluster.ClusterAppInfo{RaltRefererDefault: SwitchUp, Redirect: On}
		//cluster info
		clusterInfo.LogInfo = &pbCluster.ClusterLogInfo{IsOn: SwitchUp, NodeLogSize: 10240, RemoteLogIp: Localhost, RemoteLogPort: LogPort}
		//cache info
		clusterInfo.CacheInfo = &pbCluster.ClusterCacheInfo{IsCacheOpen: SwitchUp, CacheDbSize: 4096}
		if _, err := cli.ClusterClient.SetCluster(context.Background(), &clusterInfo); err != nil {
			return nil, log.Errorf("grpc service exec SetCluster failed: %s", err.Error())
		}
	}
	return &ClusterHandler{}, nil
}

func (h *ClusterHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cluster := ctx.Resource.(*resource.Cluster)
	cli := grpcclient.GetGrpcClient()
	var clusterInfo pbCluster.ClusterPublicInfoReq
	clusterInfo.ClusterId = cluster.GetID()
	clusterInfo.OperType = cluster.OperType
	//load balance info
	clusterInfo.BalanceInfo = &pbCluster.ClusterBalanceInfo{ClusterName: cluster.Name, ClusterType: cluster.Balance.ClusterType}
	for _, v := range cluster.Balance.NodeHosts {
		clusterInfo.BalanceInfo.NodeHost = append(clusterInfo.BalanceInfo.NodeHost, &pbCluster.NodeHost{HostId: v.HostID, NodeId: v.NodeID})
	}
	clusterInfo.BalanceInfo.BalanceType = BalanceTypeHash
	for _, v := range cluster.Balance.Ipv6Vips {
		clusterInfo.BalanceInfo.Ipv6Vip = append(clusterInfo.BalanceInfo.Ipv6Vip, &pbCluster.VipInterval{BeginVip: v.BeginVip, EndVip: v.EndVip, Length: v.Length})
	}
	//application info
	clusterInfo.AppInfo = &pbCluster.ClusterAppInfo{RaltRefererDefault: SwitchUp, Redirect: On}
	//Log info
	clusterInfo.LogInfo = &pbCluster.ClusterLogInfo{
		IsOn:          SwitchUp,
		NodeLogSize:   cluster.LogInfo.NodeLogSize,
		RemoteLogIp:   cluster.LogInfo.RemoteLogIp,
		RemoteLogPort: cluster.LogInfo.RemoteLogPort,
	}
	//cache info
	clusterInfo.CacheInfo = &pbCluster.ClusterCacheInfo{IsCacheOpen: cluster.Cache.IsOn, CacheDbSize: cluster.Cache.CacheDBSize}
	if _, err := cli.ClusterClient.SetCluster(context.Background(), &clusterInfo); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec SetCluster failed: %s", err.Error()))
	}
	cluster.Balance.Name = clusterInfo.BalanceInfo.ClusterName
	cluster.Balance.ClusterType = clusterInfo.BalanceInfo.ClusterType
	for _, v := range clusterInfo.BalanceInfo.Ipv6Vip {
		cluster.Balance.Ipv6Vips = append(cluster.Balance.Ipv6Vips, &resource.VipInterval{BeginVip: v.BeginVip, EndVip: v.EndVip, Length: v.Length})
	}

	return cluster, nil
}

func (h *ClusterHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	var clusterInfoDel pbCluster.ClusterPublicInfoReq
	clusterInfoDel.ClusterId = DefaultClusterID
	clusterInfoDel.OperType = OperTypeCreate
	cli := grpcclient.GetGrpcClient()
	if _, err := cli.ClusterClient.SetCluster(context.Background(), &clusterInfoDel); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec SetCluster failed: %s", err.Error()))
	}
	return nil
}

func (h *ClusterHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	Cluster := ctx.Resource.(*resource.Cluster)
	return Cluster, nil
}

func (h *ClusterHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	cli := grpcclient.GetGrpcClient()
	//query wether exists a cluster
	clusterIDReq := pbCluster.ClusterIDReq{ClusterId: DefaultClusterID}
	defaultCluster, err := cli.ClusterClient.QryOneCluster(context.Background(), &clusterIDReq)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("grpc service exec QryOneCluster failed: %s", err.Error()))
	}
	c := &resource.Cluster{Name: defaultCluster.SocsInfo.ClusterName}
	c.SetID(DefaultClusterID)
	c.Balance.Name = defaultCluster.GetSocsInfo().ClusterName
	c.Balance.ClusterType = defaultCluster.SocsInfo.ClusterType
	for _, v := range defaultCluster.SocsInfo.NodeHost {
		c.Balance.NodeHosts = append(c.Balance.NodeHosts, &resource.NodeHost{HostID: v.HostId, NodeID: v.NodeId})
	}
	for _, v := range defaultCluster.SocsInfo.Ipv6Vip {
		c.Balance.Ipv6Vips = append(c.Balance.Ipv6Vips, &resource.VipInterval{BeginVip: v.BeginVip, EndVip: v.EndVip, Length: v.Length})
	}
	return c, nil
}

func (h *ClusterHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var Clusters []*resource.Cluster
	return Clusters, nil
}
