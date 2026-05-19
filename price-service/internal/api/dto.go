package api

type errorResponse struct {
	Error string `json:"error"`
}

type reloadResponse struct {
	Status       string `json:"status"`
	CacheCleared bool   `json:"cache_cleared"`
}

type healthResponse struct {
	Status       string `json:"status"`
	Retailer     string `json:"retailer"`
	RetailerSlug string `json:"retailer_slug"`
}

type lastRequestsResponse struct {
	Requests any `json:"requests"`
}
