package service

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	yaml "gopkg.in/yaml.v2"

	monitorconf "github.com/linkingthing/ddi-monitor/config"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
	pghapb "github.com/linkingthing/pg-ha/pkg/proto"
	pgha "github.com/linkingthing/pg-ha/pkg/rpcserver"

	"github.com/linkingthing/ddi-controller/config"
	alarm "github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
	metrichandler "github.com/linkingthing/ddi-controller/pkg/metric/handler"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

type NodeMonitorHandler struct {
	Conf       *config.DDIControllerConfig
	LastTime   map[string]int64
	timeout    int64
	startTime  time.Time
	configFile string
	lock       sync.RWMutex
	masterIp   string
	localIp    string
}

func NewNodeMonitorHandler(conf *config.DDIControllerConfig) *NodeMonitorHandler {
	return &NodeMonitorHandler{
		Conf:       conf,
		LastTime:   make(map[string]int64, 5),
		timeout:    conf.MonitorNode.TimeOut,
		startTime:  time.Now(),
		configFile: conf.Path,
		masterIp:   conf.Server.Master,
		localIp:    conf.Server.IP,
	}
}

func (handler *NodeMonitorHandler) Register(req monitorpb.RegisterReq) error {
	handler.lock.RLock()
	defer handler.lock.RUnlock()
	if handler.Conf.Server.Master == "" {
		node := &resource.Node{
			Ip:           req.IP,
			HostName:     req.HostName,
			Roles:        monitorPbRolesToNodeRoles(req.Roles),
			StartTime:    handler.startTime,
			ControllerIp: handler.Conf.Server.IP,
		}
		node.SetID(req.IP)
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			if exists, err := tx.Exists(metrichandler.TableNode, map[string]interface{}{restdb.IDField: req.IP}); err != nil {
				return err
			} else if exists {
				_, err := tx.Update(metrichandler.TableNode, map[string]interface{}{
					"roles":         node.Roles,
					"host_name":     node.HostName,
					"master":        req.Master,
					"controller_ip": handler.Conf.Server.IP,
					"start_time":    node.StartTime},
					map[string]interface{}{restdb.IDField: req.IP})
				return err
			} else {
				_, err := tx.Insert(node)
				return err
			}
		}); err != nil {
			return fmt.Errorf("create node %s failed: %s", node.HostName, err.Error())
		}
	}
	return nil
}

func monitorPbRolesToNodeRoles(pbRoles []monitorpb.ServiceRole) []string {
	var roles []string
	for _, pbRole := range pbRoles {
		switch pbRole {
		case monitorpb.ServiceRole_ServiceRoleController:
			roles = append(roles, string(monitorconf.ServiceRoleController))
		case monitorpb.ServiceRole_ServiceRoleDNS:
			roles = append(roles, string(monitorconf.ServiceRoleDNS))
		case monitorpb.ServiceRole_ServiceRoleDHCP:
			roles = append(roles, string(monitorconf.ServiceRoleDHCP))
		}
	}
	return roles
}

func (handler *NodeMonitorHandler) KeepAlive(req monitorpb.KeepAliveReq) error {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	if handler.Conf.Server.Master == "" {
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			_, err := tx.Update(metrichandler.TableNode, map[string]interface{}{
				"roles":         monitorPbRolesToNodeRoles(req.Roles),
				"node_is_alive": true,
				"cpu_ratio":     req.CpuUsage,
				"mem_ratio":     req.MemUsage,
				"dns_is_alive":  req.DnsAlive,
				"dhcp_is_alive": req.DhcpAlive,
				"master":        req.Master,
				"controller_ip": handler.Conf.Server.IP,
				"start_time":    handler.startTime,
				"vip":           req.Vip,
			},
				map[string]interface{}{restdb.IDField: req.IP})
			handler.LastTime[req.IP] = time.Now().Unix()
			return err
		}); err != nil {
			return fmt.Errorf("create node %s failed: %s", req.IP, err.Error())
		}
	}

	return nil
}

func (handler *NodeMonitorHandler) CheckAlive() {
	ticker := time.NewTicker(time.Duration(handler.timeout) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			handler.lock.RLock()
			if handler.Conf.Server.Master == "" {
				var nodes []*resource.Node
				if err := db.GetResources(map[string]interface{}{}, &nodes); err != nil {
					log.Warnf("list nodes failed: %s", err.Error())
				}

				for _, node := range nodes {
					if time.Now().Unix()-handler.LastTime[node.ID] > handler.timeout {
						if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
							_, err := tx.Update(metrichandler.TableNode, map[string]interface{}{
								"node_is_alive": false,
								"cpu_ratio":     "0",
								"mem_ratio":     "0",
								"dhcp_is_alive": false,
								"dns_is_alive":  false,
								"start_time":    handler.startTime,
								"vip":           "",
							}, map[string]interface{}{restdb.IDField: node.ID})
							return err
						}); err != nil {
							log.Warnf("update node %s state failed: %s", node.Ip, err.Error())
						}
					}
				}
			}
			handler.lock.RUnlock()
		}
	}
}

func (handler *NodeMonitorHandler) Close() {
}

func (handler *NodeMonitorHandler) MasterUp(req pghapb.DDICtrlRequest) error {
	handler.sendHAEventIfNeed(pgha.PGHACmdMasterUp, req)
	if req.GetMasterIp() == handler.masterIp {
		return handler.MasterOper(req.GetMasterIp())
	}

	return nil
}

func (handler *NodeMonitorHandler) MasterDown(req pghapb.DDICtrlRequest) error {
	handler.sendHAEventIfNeed(pgha.PGHACmdMasterDown, req)
	if req.GetMasterIp() == handler.masterIp {
		return handler.MasterOper("")
	}

	return nil
}

func (handler *NodeMonitorHandler) MasterOper(masterIp string) error {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	file, err := ioutil.ReadFile(handler.configFile)
	if err != nil {
		return fmt.Errorf("open file %s fail: %s", handler.configFile, err.Error())
	}

	handler.Conf.Server.Master = masterIp
	file, err = yaml.Marshal(handler.Conf)
	if err != nil {
		return fmt.Errorf("marshal ddi-controller config err: %s", err.Error())
	}

	ioutil.WriteFile(handler.configFile, file, 0666)
	handler.Conf.Reload()
	return nil
}

func (handler *NodeMonitorHandler) sendHAEventIfNeed(cmd pgha.PGHACmd, req pghapb.DDICtrlRequest) {
	var thresholds []*alarm.Threshold
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: strings.ToLower(string(alarm.ThresholdNameHATrigger))}, &thresholds)
		return err
	}); err != nil {
		log.Warnf("get threshold failed: %s", err.Error())
		return
	}

	if len(thresholds) != 1 {
		return
	}

	alarm.NewEvent().Name(thresholds[0].Name).Level(thresholds[0].Level).ThresholdType(thresholds[0].ThresholdType).Time(time.Now()).
		SendMail(thresholds[0].SendMail).HaCmd(string(cmd)).MasterIp(req.GetMasterIp()).SlaveIp(req.GetSlaveIp()).Publish()

}
