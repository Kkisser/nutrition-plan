package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"price-service/internal/cache"
	"price-service/internal/config"
	debugrec "price-service/internal/debug"
	"price-service/internal/edadeal"
	"price-service/internal/estimator"
	"price-service/internal/logger"
)

func TestEstimateInvalidUnit(t *testing.T) {
	rt := testRuntime(t, mockClient{})
	body := `{"items":[{"ingredient_name":"молоко","amount":1,"unit":"литр"}]}`
	req := httptest.NewRequest(http.MethodPost, "/estimate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestEstimateEmptyItems(t *testing.T) {
	rt := testRuntime(t, mockClient{})
	req := httptest.NewRequest(http.MethodPost, "/estimate", bytes.NewBufferString(`{"items":[]}`))
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestEstimateFoundProductWithMockClient(t *testing.T) {
	rt := testRuntime(t, mockClient{
		search: edadeal.SearchProductsResult{Candidates: []edadeal.SearchCandidate{
			{ItemUUID: "milk-1l", BaseOfferUUID: "base-milk", Title: "Молоко 1 л", ItemType: "meta_offer"},
		}, ItemsCount: 1},
		detail: edadeal.ProductOffer{
			RetailerSlug:  "5ka",
			RetailerUUID:  "retailer",
			Query:         "молоко",
			ItemUUID:      "milk-1l",
			BaseOfferUUID: "base-milk",
			Title:         "Молоко 1 л",
			PackageAmount: 1000,
			PackageUnit:   "ml",
			Price:         89.99,
			PriceSource:   "price_data_new",
		},
	})
	body := `{"items":[{"ingredient_name":"молоко","amount":1000,"unit":"ml"}]}`
	req := httptest.NewRequest(http.MethodPost, "/estimate", bytes.NewBufferString(body))
	req.Header.Set("X-Request-ID", "test-request-id")
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp estimator.EstimateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.TotalPrice != 89.99 || resp.Items[0].Status != "priced" {
		t.Fatalf("response = %#v, want priced total 89.99", resp)
	}
	if resp.MinTotalPrice != 89.99 || resp.MaxTotalPrice != 89.99 || resp.PriceRange.MinPrice != 89.99 || resp.PriceRange.MaxPrice != 89.99 {
		t.Fatalf("price range fields = %#v, want exact 89.99", resp)
	}
	if resp.RequestID != "test-request-id" {
		t.Fatalf("request_id = %q, want test-request-id", resp.RequestID)
	}
	if resp.Status != "ok" || !resp.IsFullyPriced || resp.PriceType != "estimated_reference_price" || resp.CalculatedAt == "" {
		t.Fatalf("new response fields not populated correctly: %#v", resp)
	}
	if resp.Items[0].ErrorMessage != nil {
		t.Fatalf("error_message = %v, want nil", *resp.Items[0].ErrorMessage)
	}
	if len(resp.Items[0].Alternatives) != 0 {
		t.Fatalf("alternatives len = %d, want hidden by default", len(resp.Items[0].Alternatives))
	}
}

func TestEstimateIncludesAlternativesWhenRequested(t *testing.T) {
	rt := testRuntime(t, mockClient{
		search: edadeal.SearchProductsResult{Candidates: []edadeal.SearchCandidate{
			{ItemUUID: "milk-1l", BaseOfferUUID: "base-milk", Title: "Молоко 1 л", ItemType: "meta_offer"},
		}, ItemsCount: 1},
		detail: edadeal.ProductOffer{
			RetailerSlug:  "5ka",
			RetailerUUID:  "retailer",
			Query:         "молоко",
			ItemUUID:      "milk-1l",
			BaseOfferUUID: "base-milk",
			Title:         "Молоко 1 л",
			PackageAmount: 1000,
			PackageUnit:   "ml",
			Price:         89.99,
			PriceSource:   "price_data_new",
		},
	})
	body := `{"items":[{"ingredient_name":"молоко","amount":1000,"unit":"ml"}]}`
	req := httptest.NewRequest(http.MethodPost, "/estimate?include_alternatives=true", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp estimator.EstimateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items[0].Alternatives) == 0 {
		t.Fatal("alternatives hidden, want present when include_alternatives=true")
	}
}

func TestEstimateSelectedShopUUIDUsesOnlyThatShop(t *testing.T) {
	rt := testRuntime(t, mockClient{
		search: edadeal.SearchProductsResult{Candidates: []edadeal.SearchCandidate{
			{ItemUUID: "milk-1l", BaseOfferUUID: "base-milk", Title: "Молоко 1 л", ItemType: "meta_offer"},
		}, ItemsCount: 1},
		detail: edadeal.ProductOffer{
			RetailerSlug:  "5ka",
			RetailerUUID:  "retailer",
			Query:         "молоко",
			ItemUUID:      "milk-1l",
			BaseOfferUUID: "base-milk",
			Title:         "Молоко 1 л",
			PackageAmount: 1000,
			PackageUnit:   "ml",
			Price:         110,
			PriceRangeMax: ptrFloat(140),
			PriceSource:   "min_nearest_price",
			NearestShops: []edadeal.NearestShop{
				{ShopUUID: "shop-a", Price: ptrFloat(110)},
				{ShopUUID: "shop-b", Price: ptrFloat(140)},
			},
		},
	})
	body := `{"selected_shop_uuid":"shop-b","items":[{"ingredient_name":"молоко","amount":1000,"unit":"ml"}]}`
	req := httptest.NewRequest(http.MethodPost, "/estimate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp estimator.EstimateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.PricingScope != "selected_shop" || resp.SelectedShopUUID != "shop-b" {
		t.Fatalf("scope/shop = %q/%q, want selected_shop/shop-b", resp.PricingScope, resp.SelectedShopUUID)
	}
	if resp.MinTotalPrice != 140 || resp.MaxTotalPrice != 140 || resp.TotalPrice != 140 {
		t.Fatalf("totals = %.2f/%.2f/%.2f, want 140 exact", resp.MinTotalPrice, resp.MaxTotalPrice, resp.TotalPrice)
	}
	if got := resp.Items[0].SelectedOption.Packages[0].ShopUUID; got != "shop-b" {
		t.Fatalf("package shop_uuid = %q, want shop-b", got)
	}
}

func TestEstimateMixedInvalidItemReturnsItemError(t *testing.T) {
	rt := testRuntime(t, mockClient{
		search: edadeal.SearchProductsResult{Candidates: []edadeal.SearchCandidate{
			{ItemUUID: "milk-1l", BaseOfferUUID: "base-milk", Title: "Молоко 1 л", ItemType: "meta_offer"},
		}, ItemsCount: 1},
		detail: edadeal.ProductOffer{
			RetailerSlug:  "5ka",
			RetailerUUID:  "retailer",
			Query:         "молоко",
			ItemUUID:      "milk-1l",
			BaseOfferUUID: "base-milk",
			Title:         "Молоко 1 л",
			PackageAmount: 1000,
			PackageUnit:   "ml",
			Price:         89.99,
			PriceSource:   "price_data_new",
		},
	})
	body := `{"items":[{"ingredient_name":"молоко","amount":1000,"unit":"ml"},{"ingredient_name":"рис","amount":1,"unit":"литр"}]}`
	req := httptest.NewRequest(http.MethodPost, "/estimate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	rt.Estimate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp estimator.EstimateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "partial" || resp.Items[1].Status != "invalid_unit" || resp.Items[1].ErrorMessage == nil {
		t.Fatalf("mixed response = %#v, want partial with item error", resp)
	}
	if len(resp.UnpricedItems) != 1 || resp.UnpricedItems[0].RequestedAmount != 1 || resp.UnpricedItems[0].RequestedUnit != "литр" {
		t.Fatalf("unpriced_items = %#v, want requested amount/unit", resp.UnpricedItems)
	}
}

func TestReloadClearsCache(t *testing.T) {
	cacheStore := cache.NewMemoryCache()
	cacheStore.Set("key", "value", 0)
	rt := testRuntimeWithCache(t, mockClient{}, cacheStore)
	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	rec := httptest.NewRecorder()

	rt.Reload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if cacheStore.Len() != 0 {
		t.Fatalf("cache len = %d, want 0", cacheStore.Len())
	}
}

func testRuntime(t *testing.T, client mockClient) *Runtime {
	t.Helper()
	return testRuntimeWithCache(t, client, cache.NewMemoryCache())
}

func testRuntimeWithCache(t *testing.T, client mockClient, cacheStore *cache.MemoryCache) *Runtime {
	t.Helper()
	cfg := config.Default()
	retailer := edadeal.RetailerInfo{UUID: "retailer", Slug: "5ka", Name: "Пятёрочка"}
	est := estimator.New(cfg, client, retailer, cacheStore, logger.New(false))
	return NewRuntime(Dependencies{
		Config:    cfg,
		Estimator: est,
		Retailer:  retailer,
		Cache:     cacheStore,
		Recorder:  debugrec.NewRecorder(10),
	}, nil)
}

type mockClient struct {
	search edadeal.SearchProductsResult
	detail edadeal.ProductOffer
}

func (m mockClient) GetRetailerInfo(ctx context.Context) (edadeal.RetailerInfo, error) {
	return edadeal.RetailerInfo{UUID: "retailer", Slug: "5ka", Name: "Пятёрочка"}, nil
}

func (m mockClient) SearchProducts(ctx context.Context, retailerUUID string, query string) (edadeal.SearchProductsResult, error) {
	if len(m.search.Candidates) == 0 {
		return edadeal.SearchProductsResult{}, edadeal.ErrNoProductsFound
	}
	return m.search, nil
}

func (m mockClient) GetItemDetail(ctx context.Context, retailer edadeal.RetailerInfo, query string, candidate edadeal.SearchCandidate) (edadeal.ProductOffer, edadeal.RejectedCandidate, error) {
	return m.detail, edadeal.RejectedCandidate{}, nil
}

func ptrFloat(v float64) *float64 {
	return &v
}
