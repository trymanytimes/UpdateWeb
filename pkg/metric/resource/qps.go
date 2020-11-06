package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

const ResourceIDQPS = "qps"

type Qps struct {
	Values []ValueWithTimestamp `json:"values"`
}

type ValueWithTimestamp struct {
	Timestamp restresource.ISOTime `json:"timestamp"`
	Value     uint64               `json:"value"`
}
