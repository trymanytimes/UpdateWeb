package grpcclient

import (
	"google.golang.org/grpc"

	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

type GrpcClient struct {
	DHCPClient pb.DHCPManagerClient
}

var grpcClient *GrpcClient

func NewDhcpClient(conn *grpc.ClientConn) {
	grpcClient = &GrpcClient{DHCPClient: pb.NewDHCPManagerClient(conn)}
}

func GetDHCPGrpcClient() pb.DHCPManagerClient {
	return grpcClient.DHCPClient
}
