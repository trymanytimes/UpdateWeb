package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type NetworkTopology struct {
	restresource.ResourceBase  `json:",inline"`
	NetworkEquipment           string            `json:"networkEquipment" db:"ownby,uk"`
	NetworkEquipmentPort       string            `json:"networkEquipmentPort" db:"uk"`
	NetworkEquipmentPortType   EquipmentPortType `json:"networkEquipmentPortType"`
	LinkedNetworkEquipmentIp   string            `json:"linkedNetworkEquipmentIp"`
	LinkedNetworkEquipmentPort string            `json:"linkedNetworkEquipmentPort"`
}
