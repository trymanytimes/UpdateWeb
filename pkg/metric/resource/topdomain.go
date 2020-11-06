package resource

const ResourceIDTopTenDomains = "toptendomains"

type TopDomain struct {
	Domain string `json:"domain"`
	Count  uint64 `json:"count"`
}
