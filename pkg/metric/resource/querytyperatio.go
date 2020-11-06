package resource

const ResourceIDQueryTypeRatios = "querytyperatios"

type QueryTypeRatio struct {
	Type   string               `json:"type"`
	Ratios []RatioWithTimestamp `json:"ratios"`
}
