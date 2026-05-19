package estimator

type EstimateRequest struct {
	Items            []ShoppingListItem `json:"items"`
	SelectedShopUUID string             `json:"selected_shop_uuid,omitempty"`
	ShopUUID         string             `json:"shop_uuid,omitempty"`
}

type ShoppingListItem struct {
	IngredientName string  `json:"ingredient_name"`
	Amount         float64 `json:"amount"`
	Unit           string  `json:"unit"`
}

type EstimateResponse struct {
	RequestID          string          `json:"request_id"`
	Status             string          `json:"status"`
	CalculatedAt       string          `json:"calculated_at"`
	IsFullyPriced      bool            `json:"is_fully_priced"`
	PriceType          string          `json:"price_type"`
	PricingScope       string          `json:"pricing_scope"`
	SelectedShopUUID   string          `json:"selected_shop_uuid,omitempty"`
	Retailer           string          `json:"retailer"`
	RetailerSlug       string          `json:"retailer_slug"`
	Currency           string          `json:"currency"`
	TotalPrice         float64         `json:"total_price"`
	MinTotalPrice      float64         `json:"min_total_price"`
	MaxTotalPrice      float64         `json:"max_total_price"`
	PriceRange         PriceRange      `json:"price_range"`
	PricedItemsCount   int             `json:"priced_items_count"`
	UnpricedItemsCount int             `json:"unpriced_items_count"`
	Items              []EstimatedItem `json:"items"`
	UnpricedItems      []UnpricedItem  `json:"unpriced_items"`
}

type PriceRange struct {
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

type EstimatedItem struct {
	IngredientName  string          `json:"ingredient_name"`
	RequestedAmount float64         `json:"requested_amount"`
	RequestedUnit   string          `json:"requested_unit"`
	Status          string          `json:"status"`
	ErrorMessage    *string         `json:"error_message"`
	Query           string          `json:"query"`
	PriceRange      *PriceRange     `json:"price_range,omitempty"`
	SelectedOption  *SelectedOption `json:"selected_option,omitempty"`
	Alternatives    []Alternative   `json:"alternatives,omitempty"`
	Debug           any             `json:"debug,omitempty"`
}

type SelectedOption struct {
	TotalPrice    float64           `json:"total_price"`
	MinTotalPrice float64           `json:"min_total_price"`
	MaxTotalPrice float64           `json:"max_total_price"`
	PriceRange    PriceRange        `json:"price_range"`
	CoveredAmount float64           `json:"covered_amount"`
	CoveredUnit   string            `json:"covered_unit"`
	Packages      []SelectedPackage `json:"packages"`
}

type SelectedPackage struct {
	ProductTitle  string  `json:"product_title"`
	ItemUUID      string  `json:"item_uuid"`
	BaseOfferUUID string  `json:"base_offer_uuid"`
	PackageAmount float64 `json:"package_amount"`
	PackageUnit   string  `json:"package_unit"`
	UnitPrice     float64 `json:"unit_price"`
	UnitPriceMin  float64 `json:"unit_price_min"`
	UnitPriceMax  float64 `json:"unit_price_max"`
	Quantity      int     `json:"quantity"`
	Subtotal      float64 `json:"subtotal"`
	SubtotalMin   float64 `json:"subtotal_min"`
	SubtotalMax   float64 `json:"subtotal_max"`
	ImageURL      string  `json:"image_url"`
	ShopUUID      string  `json:"shop_uuid,omitempty"`
}

type Alternative struct {
	ProductTitle      string     `json:"product_title"`
	ItemUUID          string     `json:"item_uuid"`
	BaseOfferUUID     string     `json:"base_offer_uuid"`
	PackageAmount     float64    `json:"package_amount"`
	PackageUnit       string     `json:"package_unit"`
	UnitPrice         float64    `json:"unit_price"`
	UnitPriceMin      float64    `json:"unit_price_min"`
	UnitPriceMax      float64    `json:"unit_price_max"`
	QuantityToCover   int        `json:"quantity_to_cover"`
	TotalPriceToCover float64    `json:"total_price_to_cover"`
	MinPriceToCover   float64    `json:"min_price_to_cover"`
	MaxPriceToCover   float64    `json:"max_price_to_cover"`
	PriceRange        PriceRange `json:"price_range"`
}

type UnpricedItem struct {
	IngredientName  string  `json:"ingredient_name"`
	RequestedAmount float64 `json:"requested_amount"`
	RequestedUnit   string  `json:"requested_unit"`
	Reason          string  `json:"reason"`
}
