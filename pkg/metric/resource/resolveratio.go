package resource

const ResourceIDResolvedRatios = "resolvedratios"

type ResolvedRatio struct {
	Rcode  string               `json:"rcode"`
	Ratios []RatioWithTimestamp `json:"ratios"`
}
