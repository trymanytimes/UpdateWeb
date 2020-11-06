package handler

import (
	"bytes"
	"fmt"
	"time"

	ha "github.com/linkingthing/pg-ha/pkg/rpcserver"

	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var Localization = map[string]string{
	string(resource.ThresholdNameCpuUsedRatio):     "CPU使用率",
	string(resource.ThresholdNameMemoryUsedRatio):  "内存使用率",
	string(resource.ThresholdNameStorageUsedRatio): "硬盘使用率",
	string(resource.ThresholdNameHATrigger):        "HA切换",
	string(resource.ThresholdNameNodeOffline):      "节点离线",
	string(resource.ThresholdNameDNSOffline):       "DNS服务离线",
	string(resource.ThresholdNameDHCPOffline):      "DHCP服务离线",
	string(resource.ThresholdNameQPS):              "QPS",
	string(resource.ThresholdNameLPS):              "LPS",
	string(resource.ThresholdNameSubnetUsedRatio):  "地址池使用率",

	string(resource.ThresholdLevelCritical): "严重告警",
	string(resource.ThresholdLevelMajor):    "一般告警",
	string(resource.ThresholdLevelMinor):    "次要告警",
	string(resource.ThresholdLevelWarning):  "警告告警",

	string(resource.AlarmStateUntreated): "未处理",
	string(resource.AlarmStateSolved):    "已处理",
	string(resource.AlarmStateIgnored):   "已忽略",
}

func genLocalizationMessage(alarm *resource.Alarm) string {
	var buf bytes.Buffer
	buf.WriteString("告警项:   ")
	buf.WriteString(Localization[alarm.Name])
	buf.WriteString("\r\n")
	buf.WriteString("告警时间: ")
	buf.WriteString(time.Time(alarm.GetCreationTimestamp()).Format(util.TimeFormat))
	buf.WriteString("\r\n")
	buf.WriteString("告警级别: ")
	buf.WriteString(Localization[alarm.Level])
	buf.WriteString("\r\n")
	buf.WriteString("告警信息: ")
	switch resource.ThresholdName(alarm.Name) {
	case resource.ThresholdNameHATrigger:
		if ha.PGHACmd(alarm.HaCmd) == ha.PGHACmdMasterUp {
			buf.WriteString(fmt.Sprintf("辅节点 %s 切换到主节点 %s", alarm.SlaveIp, alarm.MasterIp))
		} else if ha.PGHACmd(alarm.HaCmd) == ha.PGHACmdMasterDown {
			buf.WriteString(fmt.Sprintf("主节点 %s 切换到辅节点 %s", alarm.MasterIp, alarm.SlaveIp))
		}
	case resource.ThresholdNameIPConflict:
		buf.WriteString(fmt.Sprintf("IP %s 冲突", alarm.ConflictIp))
	default:
		buf.WriteString("节点")
		buf.WriteString(alarm.NodeIp)
		buf.WriteString("的")
		if alarm.Subnet != "" {
			buf.WriteString("子网")
			buf.WriteString(Localization[alarm.Subnet])
		}
		buf.WriteString(Localization[alarm.Name])
		if resource.ThresholdType(alarm.ThresholdType) != resource.ThresholdTypeTrigger {
			buf.WriteString("超过")
			buf.WriteString(fmt.Sprintf("%d", alarm.Threshold))
			if resource.ThresholdType(alarm.ThresholdType) == resource.ThresholdTypeRatio {
				buf.WriteString("%")
			}
		}
	}

	buf.WriteString("\r\n")
	return buf.String()
}
