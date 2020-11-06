package handler

import (
	"fmt"
	"strconv"
	"time"

	monitor "github.com/linkingthing/ddi-monitor/config"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"

	alarm "github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/metric/resource"
)

func (h *NodeHandler) monitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var nodes []*resource.Node
			var thresholds []*alarm.Threshold
			if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
				if err := tx.Fill(nil, &nodes); err != nil {
					return err
				}

				return tx.Fill(nil, &thresholds)
			}); err != nil {
				log.Warnf("dhcp monitor occur error: %s", err.Error())
			}

			h.collectNodesMetric(nodes, thresholds)
		}
	}
}

func (h *NodeHandler) collectNodesMetric(nodes []*resource.Node, thresholds []*alarm.Threshold) error {
	if len(nodes) == 0 || len(thresholds) == 0 {
		return nil
	}

	period := genMonitorTimestamp(300)
	for _, node := range nodes {
		err := h.setNodeMetrics(node.GetID()+":"+strconv.Itoa(h.exportPort), []*resource.Node{node}, period)
		if err != nil {
			log.Warnf("get node %s metrics failed: %s", node.Ip, err.Error())
		}

		for _, threshold := range thresholds {
			switch threshold.Name {
			case alarm.ThresholdNameCpuUsedRatio:
				sendEventWithRatiosIfNeed(node.Ip, threshold, node.CpuUsage)
			case alarm.ThresholdNameMemoryUsedRatio:
				sendEventWithRatiosIfNeed(node.Ip, threshold, node.MemoryUsage)
			case alarm.ThresholdNameStorageUsedRatio:
				sendEventWithRatiosIfNeed(node.Ip, threshold, node.DiscUsage)
			case alarm.ThresholdNameNodeOffline:
				sendEventWithBoolIfNeed(node.Ip, threshold, node.NodeIsAlive)
			case alarm.ThresholdNameDNSOffline:
				sendServiceEventIfNeed(node.Ip, node.Roles, threshold, node.DnsIsAlive)
			case alarm.ThresholdNameDHCPOffline:
				sendServiceEventIfNeed(node.Ip, node.Roles, threshold, node.DhcpIsAlive)
			case alarm.ThresholdNameQPS:
				err = h.collectQPSMetric(node.Ip, node.Roles, threshold, period)
			case alarm.ThresholdNameLPS:
				err = h.collectLPSMetric(node.Ip, node.Roles, threshold, period)
			case alarm.ThresholdNameSubnetUsedRatio:
				err = h.collectSubnetMetric(node.Ip, node.Roles, threshold, period)
			}

			if err != nil {
				log.Warnf("get node %s metrics failed: %s", node.Ip, err.Error())
			}
		}

	}

	return nil
}

func genMonitorTimestamp(period int64) *TimePeriodParams {
	now := time.Now().Unix()
	return &TimePeriodParams{
		Begin: now - period,
		End:   now,
		Step:  period / 30,
	}
}

func sendEventWithRatiosIfNeed(nodeIP string, threshold *alarm.Threshold, ratios []resource.RatioWithTimestamp) {
	if latestTime, latestValue, ok := getLatestValueAndTimestamp(threshold.Value, ratios); ok {
		alarm.NewEvent().Node(nodeIP).Name(threshold.Name).Level(threshold.Level).ThresholdType(threshold.ThresholdType).
			Threshold(threshold.Value).Time(latestTime).Value(latestValue).SendMail(threshold.SendMail).Publish()
	}
}

func getLatestValueAndTimestamp(threshold uint64, ratios []resource.RatioWithTimestamp) (time.Time, uint64, bool) {
	var latestTime time.Time
	var latestValue uint64
	if len(ratios) == 0 {
		return latestTime, latestValue, false
	}

	var exceedThresholdCount int
	for _, ratio := range ratios {
		f, _ := strconv.ParseFloat(ratio.Ratio, 64)
		if value := uint64(f * 100); value >= threshold {
			exceedThresholdCount += 1
			latestTime = time.Time(ratio.Timestamp)
			latestValue = value
		}
	}

	return latestTime, latestValue, float64(exceedThresholdCount)/float64(len(ratios)) > 0.6
}

func sendEventWithBoolIfNeed(nodeIP string, threshold *alarm.Threshold, online bool) {
	if online {
		return
	}

	alarm.NewEvent().Node(nodeIP).Name(threshold.Name).Level(threshold.Level).ThresholdType(threshold.ThresholdType).
		Time(time.Now()).SendMail(threshold.SendMail).Publish()
}

func sendServiceEventIfNeed(nodeIP string, roles []string, threshold *alarm.Threshold, online bool) {
	if online || checkNeedSendServiceEvent(roles, threshold) == false {
		return
	}

	alarm.NewEvent().Node(nodeIP).Name(threshold.Name).Level(threshold.Level).ThresholdType(threshold.ThresholdType).
		Time(time.Now()).SendMail(threshold.SendMail).Publish()
}

func checkNeedSendServiceEvent(roles []string, threshold *alarm.Threshold) bool {
	for _, role := range roles {
		if (monitor.ServiceRole(role) == monitor.ServiceRoleDHCP && threshold.Name == alarm.ThresholdNameDHCPOffline) ||
			(monitor.ServiceRole(role) == monitor.ServiceRoleDNS && threshold.Name == alarm.ThresholdNameDNSOffline) {
			return true
		}
	}

	return false
}

func (h *NodeHandler) collectQPSMetric(nodeIP string, roles []string, threshold *alarm.Threshold, period *TimePeriodParams) error {
	if slice.SliceIndex(roles, string(monitor.ServiceRoleDNS)) == -1 {
		return nil
	}

	dns, err := getQps(&MetricContext{Client: h.httpClient, PrometheusAddr: h.prometheusAddr, NodeIP: nodeIP, Period: period})
	if err != nil {
		return fmt.Errorf("get node %s qps failed: %s", nodeIP, err.Error())
	}

	sendEventWithValuesIfNeed(nodeIP, threshold, dns.Qps.Values)
	return nil
}

func sendEventWithValuesIfNeed(nodeIP string, threshold *alarm.Threshold, values []resource.ValueWithTimestamp) {
	if len(values) == 0 {
		return
	}

	var exceedThresholdCount int
	var latestTime time.Time
	var latestValue uint64
	for _, value := range values {
		if value.Value >= threshold.Value {
			latestTime = time.Time(value.Timestamp)
			latestValue = value.Value
			exceedThresholdCount += 1
		}
	}

	if float64(exceedThresholdCount)/float64(len(values)) > 0.6 {
		alarm.NewEvent().Node(nodeIP).Name(threshold.Name).Level(threshold.Level).ThresholdType(threshold.ThresholdType).
			Threshold(threshold.Value).Time(latestTime).Value(latestValue).SendMail(threshold.SendMail).Publish()
	}
}

func (h *NodeHandler) collectLPSMetric(nodeIP string, roles []string, threshold *alarm.Threshold, period *TimePeriodParams) error {
	if slice.SliceIndex(roles, string(monitor.ServiceRoleDHCP)) == -1 {
		return nil
	}

	dhcp, err := getLps(&MetricContext{Client: h.httpClient, PrometheusAddr: h.prometheusAddr, NodeIP: nodeIP, Period: period})
	if err != nil {
		return fmt.Errorf("get node %s lps failed: %s", nodeIP, err.Error())
	}

	sendEventWithValuesIfNeed(nodeIP, threshold, dhcp.Lps.Values)
	return nil
}

func (h *NodeHandler) collectSubnetMetric(nodeIP string, roles []string, threshold *alarm.Threshold, period *TimePeriodParams) error {
	if slice.SliceIndex(roles, string(monitor.ServiceRoleDHCP)) == -1 {
		return nil
	}

	dhcp, err := getSubnetUsedRatios(&MetricContext{Client: h.httpClient, PrometheusAddr: h.prometheusAddr, NodeIP: nodeIP, Period: period})
	if err != nil {
		return fmt.Errorf("get node %s subnet used ratio failed: %s", nodeIP, err.Error())
	}

	sendEventWithSubnetRatioIfNeed(nodeIP, threshold, dhcp.SubnetUsedRatios)
	return nil
}

func sendEventWithSubnetRatioIfNeed(nodeIP string, threshold *alarm.Threshold, subnetUsedRatios []resource.SubnetUsedRatio) {
	if len(subnetUsedRatios) == 0 {
		return
	}

	for _, subnet := range subnetUsedRatios {
		if latestTime, latestValue, ok := getLatestValueAndTimestamp(threshold.Value, subnet.UsedRatios); ok {
			alarm.NewEvent().Node(nodeIP).Name(threshold.Name).Level(threshold.Level).ThresholdType(threshold.ThresholdType).Subnet(subnet.Ipnet).
				Threshold(threshold.Value).Time(latestTime).Value(latestValue).SendMail(threshold.SendMail).Publish()
		}
	}
}
