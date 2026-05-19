package estimator

import (
	"errors"
	"math"
	"sort"

	"price-service/internal/edadeal"
)

var ErrNoCandidates = errors.New("no candidates")

func OptimizePackages(required int, unit string, offers []edadeal.ProductOffer) (*SelectedOption, error) {
	if required <= 0 {
		return nil, ErrNoCandidates
	}
	if len(offers) == 0 {
		return nil, ErrNoCandidates
	}

	maxPack := 0
	for _, offer := range offers {
		if offer.PackageAmount > maxPack {
			maxPack = offer.PackageAmount
		}
	}
	if maxPack <= 0 {
		return nil, ErrNoCandidates
	}

	limit := required + maxPack
	const inf = math.MaxFloat64 / 4
	dp := make([]float64, limit+1)
	choice := make([]int, limit+1)
	prev := make([]int, limit+1)
	count := make([]int, limit+1)
	for i := range dp {
		dp[i] = inf
		choice[i] = -1
		prev[i] = -1
		count[i] = math.MaxInt32
	}
	dp[0] = 0
	count[0] = 0

	for amount := 0; amount <= limit; amount++ {
		if dp[amount] >= inf {
			continue
		}
		for idx, offer := range offers {
			next := amount + offer.PackageAmount
			if next > limit {
				next = limit
			}
			candidate := dp[amount] + offer.Price
			candidateCount := count[amount] + 1
			if candidate < dp[next]-1e-9 || (almostEqual(candidate, dp[next]) && candidateCount < count[next]) {
				dp[next] = candidate
				count[next] = candidateCount
				choice[next] = idx
				prev[next] = amount
			}
		}
	}

	bestAmount := -1
	bestPrice := inf
	bestCount := math.MaxInt32
	for amount := required; amount <= limit; amount++ {
		if dp[amount] >= inf {
			continue
		}
		if dp[amount] < bestPrice-1e-9 ||
			(almostEqual(dp[amount], bestPrice) && amount < bestAmount) ||
			(almostEqual(dp[amount], bestPrice) && amount == bestAmount && count[amount] < bestCount) {
			bestAmount = amount
			bestPrice = dp[amount]
			bestCount = count[amount]
		}
	}
	if bestAmount < 0 {
		return nil, ErrNoCandidates
	}

	selectedCounts := make(map[string]int)
	offerByKey := make(map[string]edadeal.ProductOffer)
	for amount := bestAmount; amount > 0; {
		idx := choice[amount]
		if idx < 0 {
			break
		}
		offer := offers[idx]
		key := offer.ItemUUID + "|" + offer.BaseOfferUUID
		selectedCounts[key]++
		offerByKey[key] = offer
		amount = prev[amount]
	}

	packages := make([]SelectedPackage, 0, len(selectedCounts))
	maxTotal := 0.0
	for key, qty := range selectedCounts {
		offer := offerByKey[key]
		minPrice := roundMoney(offer.Price)
		maxPrice := roundMoney(offer.EffectiveMaxPrice())
		subtotalMin := roundMoney(offer.Price * float64(qty))
		subtotalMax := roundMoney(offer.EffectiveMaxPrice() * float64(qty))
		maxTotal += subtotalMax
		packages = append(packages, SelectedPackage{
			ProductTitle:  offer.Title,
			ItemUUID:      offer.ItemUUID,
			BaseOfferUUID: offer.BaseOfferUUID,
			PackageAmount: float64(offer.PackageAmount),
			PackageUnit:   offer.PackageUnit,
			UnitPrice:     minPrice,
			UnitPriceMin:  minPrice,
			UnitPriceMax:  maxPrice,
			Quantity:      qty,
			Subtotal:      subtotalMin,
			SubtotalMin:   subtotalMin,
			SubtotalMax:   subtotalMax,
			ImageURL:      offer.ImageURL,
			ShopUUID:      offer.SelectedShopUUID,
		})
	}
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Subtotal == packages[j].Subtotal {
			return packages[i].ProductTitle < packages[j].ProductTitle
		}
		return packages[i].Subtotal > packages[j].Subtotal
	})

	minTotal := roundMoney(bestPrice)
	maxTotal = roundMoney(maxTotal)
	return &SelectedOption{
		TotalPrice:    minTotal,
		MinTotalPrice: minTotal,
		MaxTotalPrice: maxTotal,
		PriceRange: PriceRange{
			MinPrice: minTotal,
			MaxPrice: maxTotal,
		},
		CoveredAmount: float64(bestAmount),
		CoveredUnit:   unit,
		Packages:      packages,
	}, nil
}

func BuildAlternatives(required int, offers []edadeal.ProductOffer) []Alternative {
	out := make([]Alternative, 0, len(offers))
	for _, offer := range offers {
		if offer.PackageAmount <= 0 || offer.Price <= 0 {
			continue
		}
		qty := int(math.Ceil(float64(required) / float64(offer.PackageAmount)))
		minPrice := roundMoney(offer.Price)
		maxPrice := roundMoney(offer.EffectiveMaxPrice())
		minTotal := roundMoney(float64(qty) * offer.Price)
		maxTotal := roundMoney(float64(qty) * offer.EffectiveMaxPrice())
		out = append(out, Alternative{
			ProductTitle:      offer.Title,
			ItemUUID:          offer.ItemUUID,
			BaseOfferUUID:     offer.BaseOfferUUID,
			PackageAmount:     float64(offer.PackageAmount),
			PackageUnit:       offer.PackageUnit,
			UnitPrice:         minPrice,
			UnitPriceMin:      minPrice,
			UnitPriceMax:      maxPrice,
			QuantityToCover:   qty,
			TotalPriceToCover: minTotal,
			MinPriceToCover:   minTotal,
			MaxPriceToCover:   maxTotal,
			PriceRange: PriceRange{
				MinPrice: minTotal,
				MaxPrice: maxTotal,
			},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if almostEqual(out[i].TotalPriceToCover, out[j].TotalPriceToCover) {
			leftCovered := out[i].PackageAmount * float64(out[i].QuantityToCover)
			rightCovered := out[j].PackageAmount * float64(out[j].QuantityToCover)
			if leftCovered == rightCovered {
				return out[i].QuantityToCover < out[j].QuantityToCover
			}
			return leftCovered < rightCovered
		}
		return out[i].TotalPriceToCover < out[j].TotalPriceToCover
	})
	return out
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
