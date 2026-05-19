package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutrition-core/internal/domain"
)

func sampleItems() []domain.ShoppingItem {
	return []domain.ShoppingItem{
		{IngredientName: "Молоко 3.2%", Amount: 1000, Unit: domain.UnitMl},
		{IngredientName: "Гречка", Amount: 500, Unit: domain.UnitG},
	}
}

func TestEstimate_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/estimate" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type")
		}
		if r.Header.Get("X-Request-ID") != "req-1" {
			t.Errorf("missing X-Request-ID")
		}
		var req EstimateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Items) != 2 {
			t.Errorf("got %d items, want 2", len(req.Items))
		}
		resp := EstimateResponse{
			RequestID:        "req-1",
			Status:           "ok",
			MinTotalPrice:    250.0,
			MaxTotalPrice:    310.5,
			PriceRange:       PriceRange{MinPrice: 250.0, MaxPrice: 310.5},
			PricingScope:     "nearest_shops_range",
			Currency:         "RUB",
			PricedItemsCount: 2,
			IsFullyPriced:    true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := c.Estimate(ctx, sampleItems(), "", "req-1")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if r.Status != "ok" || r.MinTotalPrice != 250 || r.MaxTotalPrice != 310.5 {
		t.Errorf("bad response: %+v", r)
	}
}

func TestEstimate_WithShopUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req EstimateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.SelectedShopUUID != "shop-1" {
			t.Errorf("shop uuid passthrough broken: %q", req.SelectedShopUUID)
		}
		_ = json.NewEncoder(w).Encode(EstimateResponse{
			Status: "ok", PricingScope: "selected_shop",
			MinTotalPrice: 300, MaxTotalPrice: 300,
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	r, err := c.Estimate(context.Background(), sampleItems(), "shop-1", "")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if r.PricingScope != "selected_shop" {
		t.Errorf("scope = %q", r.PricingScope)
	}
}

func TestEstimate_Partial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(EstimateResponse{
			Status:             "partial",
			MinTotalPrice:      100,
			MaxTotalPrice:      150,
			PricedItemsCount:   1,
			UnpricedItemsCount: 1,
			UnpricedItems: []UnpricedItem{
				{IngredientName: "Кинза", Reason: "no_products_found"},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	r, err := c.Estimate(context.Background(), sampleItems(), "", "")
	if err != nil {
		t.Fatalf("partial must not be an error, got %v", err)
	}
	if r.Status != "partial" || len(r.UnpricedItems) != 1 {
		t.Errorf("partial parsing broken: %+v", r)
	}
}

func TestEstimate_503(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "edadil down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Estimate(context.Background(), sampleItems(), "", "")
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("expected ErrUpstream, got %v", err)
	}
}

func TestEstimate_400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad json", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Estimate(context.Background(), sampleItems(), "", "")
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestEstimate_Disabled(t *testing.T) {
	c := New("")
	_, err := c.Estimate(context.Background(), sampleItems(), "", "")
	if !errors.Is(err, ErrDisabled) {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestEstimate_EmptyItems(t *testing.T) {
	c := New("http://localhost")
	_, err := c.Estimate(context.Background(), nil, "", "")
	if err == nil {
		t.Error("expected error on empty items")
	}
}
