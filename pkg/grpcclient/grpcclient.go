package grpcclient

import (
	"google.golang.org/grpc"

	pbMonitor "github.com/trymanytimes/UpdateWeb/pkg/proto/ateStatsHomePage"
	pbCluster "github.com/trymanytimes/UpdateWeb/pkg/proto/ate_cluster"
	pbWebsite "github.com/trymanytimes/UpdateWeb/pkg/proto/rcs"
)

type GrpcClient struct {
	ClusterClient pbCluster.ClusterManagerClient
	WebsiteClient pbWebsite.RaltConfServClient
	MonitorClient pbMonitor.AteStatsHomePageClient
}

var grpcClient *GrpcClient

func NewGrpcClient(conn *grpc.ClientConn) {
	grpcClient = &GrpcClient{
		ClusterClient: pbCluster.NewClusterManagerClient(conn),
		WebsiteClient: pbWebsite.NewRaltConfServClient(conn),
		MonitorClient: pbMonitor.NewAteStatsHomePageClient(conn),
	}
}

func GetGrpcClient() *GrpcClient {
	return grpcClient
}
