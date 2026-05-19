package edadeal

type RetailerInfo struct {
	UUID string `json:"uuid"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type SearchCandidate struct {
	ItemUUID        string         `json:"item_uuid"`
	ItemType        string         `json:"item_type"`
	Title           string         `json:"title"`
	BaseOfferUUID   string         `json:"base_offer_uuid"`
	PriceData       map[string]any `json:"price_data,omitempty"`
	PackageAmount   int            `json:"package_amount,omitempty"`
	PackageUnit     string         `json:"package_unit,omitempty"`
	ImageURL        string         `json:"image_url,omitempty"`
	DiscountPercent *float64       `json:"discount_percent,omitempty"`
}

type SearchProductsResult struct {
	Candidates []SearchCandidate   `json:"candidates"`
	Rejected   []RejectedCandidate `json:"rejected_candidates,omitempty"`
	ItemsCount int                 `json:"items_count"`
}

type RejectedCandidate struct {
	Reason          string         `json:"reason"`
	Title           string         `json:"title,omitempty"`
	ItemUUID        string         `json:"item_uuid,omitempty"`
	BaseOfferUUID   string         `json:"base_offer_uuid,omitempty"`
	DetailURL       string         `json:"detail_endpoint_url,omitempty"`
	RawFieldSummary map[string]any `json:"raw_field_summary,omitempty"`
}

type ProductOffer struct {
	RetailerSlug     string
	RetailerUUID     string
	Query            string
	ItemUUID         string
	BaseOfferUUID    string
	ItemType         string
	Title            string
	PackageAmount    int
	PackageUnit      string
	ImageURL         string
	Price            float64
	OldPrice         *float64
	PriceSource      string
	SelectedShopUUID string
	NearestShops     []NearestShop
	RawPriceData     map[string]any

	DetailURL       string
	PriceRangeMax   *float64
	RawFieldSummary map[string]any
}

func (o ProductOffer) EffectiveMaxPrice() float64 {
	if o.PriceRangeMax != nil && *o.PriceRangeMax > 0 {
		return *o.PriceRangeMax
	}
	return o.Price
}

type NearestShop struct {
	ShopUUID string   `json:"shop_uuid"`
	Address  string   `json:"address"`
	Distance *float64 `json:"distance,omitempty"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Price    *float64 `json:"price,omitempty"`
	OldPrice *float64 `json:"old_price,omitempty"`
}
