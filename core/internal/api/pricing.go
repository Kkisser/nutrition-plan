package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"nutrition-core/internal/domain"
	"nutrition-core/internal/pricing"
)

// PricingRequest — POST /pricing.
//
// shopping_list передаётся явно (а не через user_id+week_ref), потому что
// price-service stateless и работает только с готовым списком. Источник
// списка на стороне фронта: ответ POST /plan (`shopping_list`).
type PricingRequest struct {
	ShopUUID     string            `json:"shop_uuid,omitempty"`
	ShoppingList []ShoppingItemDTO `json:"shopping_list"`
}

// PricingResponse — упрощённый выход для UI (полная диагностика
// price-service не нужна вне сервиса).
type PricingResponse struct {
	Status             string                  `json:"status"`
	PricingScope       string                  `json:"pricing_scope"`
	Currency           string                  `json:"currency"`
	MinTotalPrice      float64                 `json:"min_total_price"`
	MaxTotalPrice      float64                 `json:"max_total_price"`
	PricedItemsCount   int                     `json:"priced_items_count"`
	UnpricedItemsCount int                     `json:"unpriced_items_count"`
	UnpricedItems      []pricing.UnpricedItem  `json:"unpriced_items,omitempty"`
}

func (h *Handler) PostPricing(w http.ResponseWriter, r *http.Request) {
	if h.pricing == nil {
		writeError(w, http.StatusServiceUnavailable,
			fmt.Errorf("pricing disabled (PRICE_SERVICE_URL not set)"))
		return
	}
	var req PricingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	if len(req.ShoppingList) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("shopping_list is empty"))
		return
	}

	items := make([]domain.ShoppingItem, len(req.ShoppingList))
	for i, x := range req.ShoppingList {
		items[i] = domain.ShoppingItem{
			IngredientName: x.IngredientName,
			Amount:         x.Amount,
			Unit:           domain.Unit(x.Unit),
		}
	}

	reqID := r.Header.Get("X-Request-ID")
	out, err := h.pricing.Estimate(r.Context(), items, req.ShopUUID, reqID)
	if err != nil {
		switch {
		case errors.Is(err, pricing.ErrDisabled):
			writeError(w, http.StatusServiceUnavailable, err)
		case errors.Is(err, pricing.ErrUpstream):
			writeError(w, http.StatusBadGateway, err)
		case errors.Is(err, pricing.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, PricingResponse{
		Status:             out.Status,
		PricingScope:       out.PricingScope,
		Currency:           out.Currency,
		MinTotalPrice:      out.MinTotalPrice,
		MaxTotalPrice:      out.MaxTotalPrice,
		PricedItemsCount:   out.PricedItemsCount,
		UnpricedItemsCount: out.UnpricedItemsCount,
		UnpricedItems:      out.UnpricedItems,
	})
}
