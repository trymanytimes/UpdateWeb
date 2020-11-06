package resource

const ResourceIDLease = "lease"

type Lease struct {
	Values []ValueWithTimestamp `json:"values"`
}
