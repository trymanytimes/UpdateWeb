package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type ThresholdName string

const (
	ThresholdNameCpuUsedRatio     ThresholdName = "cpuUsedRatio"
	ThresholdNameMemoryUsedRatio  ThresholdName = "memoryUsedRatio"
	ThresholdNameStorageUsedRatio ThresholdName = "storageUsedRatio"
	ThresholdNameQPS              ThresholdName = "qps"
	ThresholdNameLPS              ThresholdName = "lps"
	ThresholdNameSubnetUsedRatio  ThresholdName = "subnetUsedRatio"
	ThresholdNameHATrigger        ThresholdName = "haTrigger"
	ThresholdNameNodeOffline      ThresholdName = "nodeOffline"
	ThresholdNameDNSOffline       ThresholdName = "dnsOffline"
	ThresholdNameDHCPOffline      ThresholdName = "dhcpOffline"
	ThresholdNameIPConflict       ThresholdName = "ipConflict"
)

type ThresholdLevel string

const (
	ThresholdLevelCritical ThresholdLevel = "critical"
	ThresholdLevelMajor    ThresholdLevel = "major"
	ThresholdLevelMinor    ThresholdLevel = "minor"
	ThresholdLevelWarning  ThresholdLevel = "warning"
)

type ThresholdType string

const (
	ThresholdTypeValue   ThresholdType = "value"
	ThresholdTypeRatio   ThresholdType = "ratio"
	ThresholdTypeTrigger ThresholdType = "trigger"
)

type Threshold struct {
	restresource.ResourceBase `json:",inline"`
	Name                      ThresholdName  `json:"name" rest:"required=true" db:"uk"`
	Level                     ThresholdLevel `json:"level" rest:"description=readonly"`
	ThresholdType             ThresholdType  `json:"thresholdType" rest:"description=readonly"`
	Value                     uint64         `json:"value"`
	SendMail                  bool           `json:"sendMail"`
}

var TableThreshold = restdb.ResourceDBType(&Threshold{})
