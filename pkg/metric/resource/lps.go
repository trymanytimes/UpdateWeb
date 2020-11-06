package resource

const ResourceIDLPS = "lps"

type Lps struct {
	Values []ValueWithTimestamp `json:"values"`
}
