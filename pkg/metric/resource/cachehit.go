package resource

const ResourceIDCacheHitRatio = "cachehitratio"

type CacheHitRatio struct {
	Ratios []RatioWithTimestamp `json:"ratios"`
}
