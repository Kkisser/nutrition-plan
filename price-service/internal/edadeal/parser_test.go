package edadeal

import (
	"errors"
	"testing"
)

func TestParseSearchProductsNoItems(t *testing.T) {
	_, err := ParseSearchProducts([]byte(`{}`))
	if !errors.Is(err, ErrNoProductsFound) {
		t.Fatalf("ParseSearchProducts() error = %v, want %v", err, ErrNoProductsFound)
	}
}

func TestParseSearchProductsRejectsMissingUUID(t *testing.T) {
	result, err := ParseSearchProducts([]byte(`{"items":[{"title":"Молоко 1 л","baseOfferUuid":"base"}]}`))
	if !errors.Is(err, ErrNoProductsFound) {
		t.Fatalf("ParseSearchProducts() error = %v, want %v", err, ErrNoProductsFound)
	}
	if len(result.Rejected) != 1 || result.Rejected[0].Reason != "missing_uuid" {
		t.Fatalf("rejected = %#v, want missing_uuid", result.Rejected)
	}
}

func TestParseSearchProductsRejectsMissingBaseOfferUUID(t *testing.T) {
	result, err := ParseSearchProducts([]byte(`{"items":[{"uuid":"item","title":"Молоко 1 л"}]}`))
	if !errors.Is(err, ErrNoProductsFound) {
		t.Fatalf("ParseSearchProducts() error = %v, want %v", err, ErrNoProductsFound)
	}
	if len(result.Rejected) != 1 || result.Rejected[0].Reason != "missing_base_offer_uuid" {
		t.Fatalf("rejected = %#v, want missing_base_offer_uuid", result.Rejected)
	}
}

func TestParseItemDetailRejectsMissingPrice(t *testing.T) {
	candidate := SearchCandidate{ItemUUID: "item", BaseOfferUUID: "base", Title: "Молоко 1 л"}
	_, rejected, err := ParseItemDetail([]byte(`{"title":"Молоко 1 л","quantity":1,"quantityUnit":"л"}`), candidate, retailer(), "молоко", "detail-url")
	if !errors.Is(err, ErrMissingPrice) {
		t.Fatalf("ParseItemDetail() error = %v, want %v", err, ErrMissingPrice)
	}
	if rejected.Reason != "reject_missing_price" {
		t.Fatalf("rejected reason = %q, want reject_missing_price", rejected.Reason)
	}
}

func TestParseItemDetailRejectsMissingPackage(t *testing.T) {
	candidate := SearchCandidate{ItemUUID: "item", BaseOfferUUID: "base", Title: "Молоко"}
	_, rejected, err := ParseItemDetail([]byte(`{"title":"Молоко","priceData":{"new":100}}`), candidate, retailer(), "молоко", "detail-url")
	if !errors.Is(err, ErrMissingAmount) {
		t.Fatalf("ParseItemDetail() error = %v, want %v", err, ErrMissingAmount)
	}
	if rejected.Reason != "reject_missing_amount" {
		t.Fatalf("rejected reason = %q, want reject_missing_amount", rejected.Reason)
	}
}

func TestParseItemDetailUsesNearestPriceRange(t *testing.T) {
	candidate := SearchCandidate{ItemUUID: "item", BaseOfferUUID: "base", Title: "Молоко 1 л"}
	offer, _, err := ParseItemDetail([]byte(`{
		"title":"Молоко 1 л",
		"quantity":1,
		"quantityUnit":"л",
		"partner":{
			"slug":"5ka",
			"nearest":[
				{"shopUuid":"shop-a","price":{"new":11000}},
				{"shopUuid":"shop-b","price":{"new":14000}}
			]
		}
	}`), candidate, retailer(), "молоко", "detail-url")
	if err != nil {
		t.Fatalf("ParseItemDetail() error = %v", err)
	}
	if offer.Price != 110 || offer.EffectiveMaxPrice() != 140 {
		t.Fatalf("price range = %.2f/%.2f, want 110/140", offer.Price, offer.EffectiveMaxPrice())
	}
}

func retailer() RetailerInfo {
	return RetailerInfo{UUID: "retailer", Slug: "5ka", Name: "Пятёрочка"}
}
