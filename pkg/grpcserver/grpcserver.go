package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/trymanytimes/UpdateWeb/config"
	metricsrv "github.com/trymanytimes/UpdateWeb/pkg/metric/service"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
	pghapb "github.com/linkingthing/pg-ha/pkg/proto"
)

type GRPCServer struct {
	MonitorService *metricsrv.NodeMonitorService
	server         *grpc.Server
	listener       net.Listener
}

func New(conf *config.DDIControllerConfig) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", conf.Server.GrpcAddr)
	if err != nil {
		return nil, fmt.Errorf("create listener with addr %s failed: %s", conf.Server.GrpcAddr, err.Error())
	}

	grpcServer := &GRPCServer{
		server:   grpc.NewServer(),
		listener: listener,
	}

	MonitorService, err := metricsrv.New(conf)
	if err != nil {
		return nil, err
	}

	grpcServer.MonitorService = MonitorService
	monitorpb.RegisterMonitorManagerServer(grpcServer.server, MonitorService)
	pghapb.RegisterDDICtrlManagerServer(grpcServer.server, MonitorService)

	return grpcServer, nil
}

func (s *GRPCServer) Run() error {
	defer s.Stop()
	return s.server.Serve(s.listener)
}

func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	s.MonitorService.Close()
	return nil
}
