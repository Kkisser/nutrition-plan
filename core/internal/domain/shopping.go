package domain

// ShoppingItem — позиция агрегированного списка покупок.
// Формат совпадает с items[] price-service (docs/КОНТРАКТ_API.md §2.2, §3.2).
type ShoppingItem struct {
	IngredientName string  `json:"ingredient_name"`
	Category       string  `json:"category,omitempty"`
	Amount         float64 `json:"amount"`
	Unit           Unit    `json:"unit"`
}
