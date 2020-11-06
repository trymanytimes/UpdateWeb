package resource

const ResourceIDPackets = "packets"

type Packet struct {
	Version string               `json:"version"`
	Type    string               `json:"type"`
	Values  []ValueWithTimestamp `json:"values"`
}
