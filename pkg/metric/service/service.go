package service

import (
	"context"
	"fmt"

	"github.com/trymanytimes/UpdateWeb/config"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
	pghapb "github.com/linkingthing/pg-ha/pkg/proto"
)

const (
	opSuccess = 0
	opFail    = 1
)

type NodeMonitorService struct {
	handler *NodeMonitorHandler
}

func New(conf *config.DDIControllerConfig) (*NodeMonitorService, error) {
	handler := NewNodeMonitorHandler(conf)
	go handler.CheckAlive()
	return &NodeMonitorService{handler: handler}, nil
}

func (service *NodeMonitorService) Register(content context.Context, req *monitorpb.RegisterReq) (*monitorpb.RegisterResp, error) {
	err := service.handler.Register(*req)
	if err != nil {
		return &monitorpb.RegisterResp{Success: false, Msg: fmt.Sprintf("%s", err)}, err
	} else {
		return &monitorpb.RegisterResp{Success: true, Msg: ""}, nil
	}
}

func (service *NodeMonitorService) KeepAlive(context context.Context, req *monitorpb.KeepAliveReq) (*monitorpb.KeepAliveResp, error) {
	err := service.handler.KeepAlive(*req)
	if err != nil {
		return &monitorpb.KeepAliveResp{Success: false, Msg: fmt.Sprintf("%s", err)}, err
	} else {
		return &monitorpb.KeepAliveResp{Success: true, Msg: ""}, nil
	}
}

func (service *NodeMonitorService) MasterUp(content context.Context, req *pghapb.DDICtrlRequest) (*pghapb.DDICtrlResponse, error) {
	err := service.handler.MasterUp(*req)
	if err != nil {
		return &pghapb.DDICtrlResponse{Succeed: false}, err
	} else {
		return &pghapb.DDICtrlResponse{Succeed: true}, nil
	}
}

func (service *NodeMonitorService) MasterDown(context context.Context, req *pghapb.DDICtrlRequest) (*pghapb.DDICtrlResponse, error) {
	err := service.handler.MasterDown(*req)
	if err != nil {
		return &pghapb.DDICtrlResponse{Succeed: false}, err
	} else {
		return &pghapb.DDICtrlResponse{Succeed: true}, nil
	}
}

func (service *NodeMonitorService) Close() {
	service.handler.Close()
}
