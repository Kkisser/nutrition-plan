package estimator

import (
	"testing"

	"price-service/internal/edadeal"
)

func TestOptimizePackages(t *testing.T) {
	tests := []struct {
		name        string
		required    int
		unit        string
		offers      []edadeal.ProductOffer
		wantItem    string
		wantQty     int
		wantCovered float64
		wantPrice   float64
	}{
		{
			name:     "1000 ml chooses one 1000 ml",
			required: 1000,
			unit:     "ml",
			offers: []edadeal.ProductOffer{
				offer("950", 950, "ml", 84.99),
				offer("1000", 1000, "ml", 89.99),
			},
			wantItem:    "1000",
			wantQty:     1,
			wantCovered: 1000,
			wantPrice:   89.99,
		},
		{
			name:     "1900 ml chooses two 950 ml",
			required: 1900,
			unit:     "ml",
			offers: []edadeal.ProductOffer{
				offer("950", 950, "ml", 84.99),
				offer("1000", 1000, "ml", 89.99),
			},
			wantItem:    "950",
			wantQty:     2,
			wantCovered: 1900,
			wantPrice:   169.98,
		},
		{
			name:     "500 g chooses 900 g over two 450 g",
			required: 500,
			unit:     "g",
			offers: []edadeal.ProductOffer{
				offer("900", 900, "g", 120),
				offer("450", 450, "g", 70),
			},
			wantItem:    "900",
			wantQty:     1,
			wantCovered: 900,
			wantPrice:   120,
		},
		{
			name:     "10 pcs chooses 10 pcs",
			required: 10,
			unit:     "pcs",
			offers: []edadeal.ProductOffer{
				offer("10", 10, "pcs", 99),
				offer("20", 20, "pcs", 160),
			},
			wantItem:    "10",
			wantQty:     1,
			wantCovered: 10,
			wantPrice:   99,
		},
		{
			name:     "tie chooses smaller covered amount",
			required: 1000,
			unit:     "g",
			offers: []edadeal.ProductOffer{
				offer("1000", 1000, "g", 100),
				offer("1200", 1200, "g", 100),
			},
			wantItem:    "1000",
			wantQty:     1,
			wantCovered: 1000,
			wantPrice:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OptimizePackages(tt.required, tt.unit, tt.offers)
			if err != nil {
				t.Fatalf("OptimizePackages() error = %v", err)
			}
			if got.TotalPrice != tt.wantPrice || got.CoveredAmount != tt.wantCovered {
				t.Fatalf("OptimizePackages() total/covered = %.2f/%.0f, want %.2f/%.0f", got.TotalPrice, got.CoveredAmount, tt.wantPrice, tt.wantCovered)
			}
			if len(got.Packages) != 1 {
				t.Fatalf("len(Packages) = %d, want 1", len(got.Packages))
			}
			pkg := got.Packages[0]
			if pkg.ItemUUID != tt.wantItem || pkg.Quantity != tt.wantQty {
				t.Fatalf("selected = %s x%d, want %s x%d", pkg.ItemUUID, pkg.Quantity, tt.wantItem, tt.wantQty)
			}
		})
	}
}

func TestOptimizePackagesEmptyOffers(t *testing.T) {
	if _, err := OptimizePackages(100, "g", nil); err == nil {
		t.Fatal("OptimizePackages() error = nil, want error")
	}
}

func TestOptimizePackagesReturnsPriceRange(t *testing.T) {
	maxPrice := 140.0
	got, err := OptimizePackages(1000, "ml", []edadeal.ProductOffer{
		{
			ItemUUID:         "milk",
			BaseOfferUUID:    "base-milk",
			Title:            "Молоко 1 л",
			PackageAmount:    1000,
			PackageUnit:      "ml",
			Price:            110,
			PriceRangeMax:    &maxPrice,
			SelectedShopUUID: "shop-a",
		},
	})
	if err != nil {
		t.Fatalf("OptimizePackages() error = %v", err)
	}
	if got.MinTotalPrice != 110 || got.MaxTotalPrice != 140 {
		t.Fatalf("range = %.2f/%.2f, want 110/140", got.MinTotalPrice, got.MaxTotalPrice)
	}
	if got.PriceRange.MinPrice != 110 || got.PriceRange.MaxPrice != 140 {
		t.Fatalf("price_range = %#v, want 110/140", got.PriceRange)
	}
	if got.Packages[0].UnitPriceMin != 110 || got.Packages[0].UnitPriceMax != 140 || got.Packages[0].ShopUUID != "shop-a" {
		t.Fatalf("package range/shop = %#v, want min/max and shop uuid", got.Packages[0])
	}
}

func offer(id string, amount int, unit string, price float64) edadeal.ProductOffer {
	return edadeal.ProductOffer{
		ItemUUID:      id,
		BaseOfferUUID: "base-" + id,
		Title:         "product " + id,
		PackageAmount: amount,
		PackageUnit:   unit,
		Price:         price,
	}
}
