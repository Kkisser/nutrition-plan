// Package pricing — HTTP-клиент к внешнему price-service.
// Контракт: docs/КОНТРАКТ_API.md §3, ../price-service/docs/backend-*.md.
package pricing

import "nutrition-core/internal/domain"

// EstimateRequest — POST /estimate (price-service).
type EstimateRequest struct {
	SelectedShopUUID string         `json:"selected_shop_uuid,omitempty"`
	Items            []EstimateItem `json:"items"`
}

// EstimateItem совпадает с domain.ShoppingItem по полям.
type EstimateItem struct {
	IngredientName string  `json:"ingredient_name"`
	Amount         float64 `json:"amount"`
	Unit           string  `json:"unit"`
}

func itemsFromShopping(in []domain.ShoppingItem) []EstimateItem {
	out := make([]EstimateItem, len(in))
	for i, x := range in {
		out[i] = EstimateItem{
			IngredientName: x.IngredientName,
			Amount:         x.Amount,
			Unit:           string(x.Unit),
		}
	}
	return out
}

// EstimateResponse — выход /estimate, отражает фактический формат
// price-service (КОНТРАКТ_API.md §3.3a).
type EstimateResponse struct {
	RequestID          string       `json:"request_id"`
	Status             string       `json:"status"`            // ok | partial | failed
	CalculatedAt       string       `json:"calculated_at"`
	IsFullyPriced      bool         `json:"is_fully_priced"`
	PriceType          string       `json:"price_type"`
	PricingScope       string       `json:"pricing_scope"`     // nearest_shops_range | selected_shop
	SelectedShopUUID   string       `json:"selected_shop_uuid,omitempty"`
	Retailer           string       `json:"retailer"`
	RetailerSlug       string       `json:"retailer_slug"`
	Currency           string       `json:"currency"`
	TotalPrice         float64      `json:"total_price"`       // = MinTotalPrice (back-compat)
	MinTotalPrice      float64      `json:"min_total_price"`
	MaxTotalPrice      float64      `json:"max_total_price"`
	PriceRange         PriceRange   `json:"price_range"`
	PricedItemsCount   int          `json:"priced_items_count"`
	UnpricedItemsCount int          `json:"unpriced_items_count"`
	UnpricedItems      []UnpricedItem `json:"unpriced_items"`
	// Items []ItemDetail — не разбираем поэлементно; бэкенд хранит только
	// то, что нужно для UI: min/max total + список unpriced. Для текущей
	// задачи детальный разбор items[] не нужен.
}

type PriceRange struct {
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

type UnpricedItem struct {
	IngredientName string  `json:"ingredient_name"`
	RequestedAmount float64 `json:"requested_amount"`
	RequestedUnit  string  `json:"requested_unit"`
	Reason         string  `json:"reason"`
}
