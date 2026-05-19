package edadeal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"price-service/internal/normalizer"
)

func ParseRetailerInfo(raw []byte) (RetailerInfo, error) {
	root, err := decodeObject(raw)
	if err != nil {
		return RetailerInfo{}, ErrInvalidJSON
	}
	if info, ok := retailerFromObject(root); ok {
		return info, nil
	}
	for _, key := range []string{"retailer", "partner", "data", "item"} {
		if child, ok := mapField(root, key); ok {
			if info, ok := retailerFromObject(child); ok {
				return info, nil
			}
		}
	}
	for _, key := range []string{"items", "retailers", "results"} {
		if arr, ok := arrayField(root, key); ok {
			for _, item := range arr {
				if child, ok := item.(map[string]any); ok {
					if info, ok := retailerFromObject(child); ok {
						return info, nil
					}
				}
			}
		}
	}
	return RetailerInfo{}, ErrRetailerNotFound
}

func ParseSearchProducts(raw []byte) (SearchProductsResult, error) {
	root, err := decodeObject(raw)
	if err != nil {
		return SearchProductsResult{}, ErrInvalidJSON
	}
	items, ok := arrayField(root, "items")
	if !ok || len(items) == 0 {
		return SearchProductsResult{ItemsCount: 0}, ErrNoProductsFound
	}

	result := SearchProductsResult{ItemsCount: len(items)}
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			result.Rejected = append(result.Rejected, RejectedCandidate{Reason: "invalid_item"})
			continue
		}
		candidate := SearchCandidate{
			ItemUUID:      stringField(item, "uuid", "itemUuid", "item_uuid"),
			ItemType:      stringField(item, "itemType", "item_type", "type"),
			Title:         stringField(item, "title", "name"),
			BaseOfferUUID: stringField(item, "baseOfferUuid", "base_offer_uuid"),
			ImageURL:      stringField(item, "imageUrl", "image_url"),
			PriceData:     mapFieldOrNil(item, "priceData", "price_data"),
		}
		if discount, ok := numberField(item, "discountPercent", "discount_percent"); ok {
			candidate.DiscountPercent = &discount
		}
		if amount, unit, err := normalizer.ParsePackage(item["quantity"], item["quantityUnit"], candidate.Title); err == nil {
			candidate.PackageAmount = amount
			candidate.PackageUnit = unit
		}
		switch {
		case candidate.ItemUUID == "":
			result.Rejected = append(result.Rejected, rejectFromMap("missing_uuid", item, ""))
		case candidate.BaseOfferUUID == "" && candidate.Title != "":
			result.Rejected = append(result.Rejected, rejectFromMap("missing_base_offer_uuid", item, ""))
		default:
			result.Candidates = append(result.Candidates, candidate)
		}
	}

	sort.SliceStable(result.Candidates, func(i, j int) bool {
		left := result.Candidates[i].ItemType == "meta_offer"
		right := result.Candidates[j].ItemType == "meta_offer"
		return left && !right
	})
	if len(result.Candidates) == 0 {
		return result, ErrNoProductsFound
	}
	return result, nil
}

func ParseItemDetail(raw []byte, candidate SearchCandidate, retailer RetailerInfo, query string, detailURL string) (ProductOffer, RejectedCandidate, error) {
	root, err := decodeObject(raw)
	if err != nil {
		return ProductOffer{}, rejectedFromCandidate(candidate, "invalid_json", detailURL), ErrInvalidJSON
	}
	item := unwrapDetail(root)
	title := stringField(item, "title", "name")
	if title == "" {
		title = candidate.Title
	}
	imageURL := stringField(item, "imageUrl", "image_url")
	if imageURL == "" {
		imageURL = candidate.ImageURL
	}
	baseOfferUUID := candidate.BaseOfferUUID
	if baseOfferUUID == "" {
		baseOfferUUID = stringField(item, "baseOfferUuid", "base_offer_uuid")
	}
	if baseOfferUUID == "" {
		baseOfferUUID = firstStringFromArray(item, "offerUuids", "offer_uuids")
	}
	priceData := mapFieldOrNil(item, "priceData", "price_data")
	if len(priceData) == 0 {
		priceData = candidate.PriceData
	}

	partner, _ := mapField(item, "partner")
	retailerSlug := retailer.Slug
	retailerName := retailer.Name
	if slug := stringField(partner, "slug"); slug != "" {
		retailerSlug = slug
	}
	if name := stringField(partner, "name", "title"); name != "" {
		retailerName = name
		_ = retailerName
	}

	nearest := parseNearest(partner)
	price, oldPrice, source, priceMax := choosePrice(nearest, priceData)
	if price <= 0 {
		return ProductOffer{}, rejectedFromCandidate(candidate, "reject_missing_price", detailURL), ErrMissingPrice
	}

	amount, unit, err := normalizer.ParsePackage(item["quantity"], item["quantityUnit"], title)
	if err != nil || amount <= 0 {
		if candidate.PackageAmount > 0 && candidate.PackageUnit != "" {
			amount = candidate.PackageAmount
			unit = candidate.PackageUnit
		} else {
			return ProductOffer{}, rejectedFromCandidate(candidate, "reject_missing_amount", detailURL), ErrMissingAmount
		}
	}

	offer := ProductOffer{
		RetailerSlug:    retailerSlug,
		RetailerUUID:    retailer.UUID,
		Query:           query,
		ItemUUID:        candidate.ItemUUID,
		BaseOfferUUID:   baseOfferUUID,
		ItemType:        candidate.ItemType,
		Title:           title,
		PackageAmount:   amount,
		PackageUnit:     unit,
		ImageURL:        imageURL,
		Price:           price,
		OldPrice:        oldPrice,
		PriceSource:     source,
		NearestShops:    nearest,
		RawPriceData:    priceData,
		DetailURL:       detailURL,
		PriceRangeMax:   priceMax,
		RawFieldSummary: rawFieldSummary(item),
	}
	return offer, RejectedCandidate{}, nil
}

func decodeObject(raw []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	return root, nil
}

func retailerFromObject(obj map[string]any) (RetailerInfo, bool) {
	info := RetailerInfo{
		UUID: stringField(obj, "uuid", "retailerUuid", "retailer_uuid"),
		Slug: stringField(obj, "slug", "retailerSlug", "retailer_slug"),
		Name: stringField(obj, "name", "title"),
	}
	if info.UUID == "" {
		return RetailerInfo{}, false
	}
	return info, true
}

func unwrapDetail(root map[string]any) map[string]any {
	for _, key := range []string{"item", "data", "offer"} {
		if child, ok := mapField(root, key); ok {
			return child
		}
	}
	return root
}

func parseNearest(partner map[string]any) []NearestShop {
	arr, ok := arrayField(partner, "nearest")
	if !ok {
		return nil
	}
	shops := make([]NearestShop, 0, len(arr))
	for _, rawShop := range arr {
		shopMap, ok := rawShop.(map[string]any)
		if !ok {
			continue
		}
		priceMap, _ := mapField(shopMap, "price")
		price, hasPrice := numberField(priceMap, "new")
		oldPrice, hasOld := numberField(priceMap, "old")
		distance, hasDistance := numberField(shopMap, "distance")
		lat, hasLat := numberField(shopMap, "lat", "latitude")
		lng, hasLng := numberField(shopMap, "lng", "lon", "longitude")
		shop := NearestShop{
			ShopUUID: stringField(shopMap, "shopUuid", "shop_uuid", "uuid"),
			Address:  stringField(shopMap, "address"),
		}
		if hasPrice {
			price = normalizeMoney(price)
			shop.Price = &price
		}
		if hasOld {
			oldPrice = normalizeMoney(oldPrice)
			shop.OldPrice = &oldPrice
		}
		if hasDistance {
			shop.Distance = &distance
		}
		if hasLat {
			shop.Lat = &lat
		}
		if hasLng {
			shop.Lng = &lng
		}
		shops = append(shops, shop)
	}
	return shops
}

func choosePrice(nearest []NearestShop, priceData map[string]any) (float64, *float64, string, *float64) {
	best := 0.0
	maxPrice := 0.0
	var old *float64
	for _, shop := range nearest {
		if shop.Price == nil || *shop.Price <= 0 {
			continue
		}
		price := normalizeMoney(*shop.Price)
		if best == 0 || price < best {
			best = price
			old = shop.OldPrice
			if old != nil {
				normalizedOld := normalizeMoney(*old)
				old = &normalizedOld
			}
		}
		if price > maxPrice {
			maxPrice = price
		}
	}
	if best > 0 {
		return best, old, "min_nearest_price", &maxPrice
	}

	if newPrice, ok := mapField(priceData, "new"); ok {
		if price, ok := numberField(newPrice, "value"); ok && price > 0 {
			return normalizeMoney(price), nil, "price_data_new", nil
		}
		if price, ok := numberField(newPrice, "from"); ok && price > 0 {
			var max *float64
			if to, hasTo := numberField(newPrice, "to"); hasTo {
				normalizedTo := normalizeMoney(to)
				max = &normalizedTo
			}
			return normalizeMoney(price), nil, "price_data_from", max
		}
	}
	if price, ok := numberField(priceData, "new"); ok && price > 0 {
		if oldPrice, hasOld := numberField(priceData, "old"); hasOld {
			normalizedOld := normalizeMoney(oldPrice)
			old = &normalizedOld
		}
		return normalizeMoney(price), old, "price_data_new", nil
	}
	if price, ok := numberField(priceData, "from"); ok && price > 0 {
		var max *float64
		if to, hasTo := numberField(priceData, "to"); hasTo {
			normalizedTo := normalizeMoney(to)
			max = &normalizedTo
		}
		return normalizeMoney(price), nil, "price_data_from", max
	}
	if price, ok := numberField(priceData, "to"); ok && price > 0 {
		normalizedPrice := normalizeMoney(price)
		return normalizedPrice, nil, "price_data_to", &normalizedPrice
	}
	return 0, nil, "", nil
}

func mapField(obj map[string]any, keys ...string) (map[string]any, bool) {
	if obj == nil {
		return nil, false
	}
	for _, key := range keys {
		if v, ok := obj[key]; ok {
			if m, ok := v.(map[string]any); ok {
				return m, true
			}
		}
	}
	return nil, false
}

func mapFieldOrNil(obj map[string]any, keys ...string) map[string]any {
	m, ok := mapField(obj, keys...)
	if !ok {
		return nil
	}
	return m
}

func arrayField(obj map[string]any, keys ...string) ([]any, bool) {
	if obj == nil {
		return nil, false
	}
	for _, key := range keys {
		if v, ok := obj[key]; ok {
			if arr, ok := v.([]any); ok {
				return arr, true
			}
		}
	}
	return nil, false
}

func stringField(obj map[string]any, keys ...string) string {
	if obj == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := obj[key]; ok {
			switch x := v.(type) {
			case string:
				return strings.TrimSpace(x)
			case json.Number:
				return x.String()
			case float64:
				return fmt.Sprintf("%g", x)
			}
		}
	}
	return ""
}

func numberField(obj map[string]any, keys ...string) (float64, bool) {
	if obj == nil {
		return 0, false
	}
	for _, key := range keys {
		v, ok := obj[key]
		if !ok {
			continue
		}
		switch x := v.(type) {
		case map[string]any:
			if nested, ok := numberField(x, "value"); ok {
				return nested, true
			}
		case json.Number:
			f, err := x.Float64()
			return f, err == nil
		case float64:
			return x, true
		case int:
			return float64(x), true
		case string:
			f, err := parseNumericString(x)
			return f, err == nil
		}
	}
	return 0, false
}

func normalizeMoney(v float64) float64 {
	if v >= 1000 && v == float64(int64(v)) {
		return v / 100
	}
	return v
}

func firstStringFromArray(obj map[string]any, keys ...string) string {
	arr, ok := arrayField(obj, keys...)
	if !ok {
		return ""
	}
	for _, item := range arr {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func parseNumericString(value string) (float64, error) {
	var n json.Number = json.Number(strings.ReplaceAll(strings.TrimSpace(value), ",", "."))
	return n.Float64()
}

func rejectFromMap(reason string, item map[string]any, detailURL string) RejectedCandidate {
	return RejectedCandidate{
		Reason:          reason,
		Title:           stringField(item, "title", "name"),
		ItemUUID:        stringField(item, "uuid", "itemUuid", "item_uuid"),
		BaseOfferUUID:   stringField(item, "baseOfferUuid", "base_offer_uuid"),
		DetailURL:       detailURL,
		RawFieldSummary: rawFieldSummary(item),
	}
}

func rejectedFromCandidate(candidate SearchCandidate, reason string, detailURL string) RejectedCandidate {
	return RejectedCandidate{
		Reason:        reason,
		Title:         candidate.Title,
		ItemUUID:      candidate.ItemUUID,
		BaseOfferUUID: candidate.BaseOfferUUID,
		DetailURL:     detailURL,
	}
}

func rawFieldSummary(item map[string]any) map[string]any {
	if item == nil {
		return nil
	}
	summary := make(map[string]any)
	for _, key := range []string{"uuid", "itemType", "title", "baseOfferUuid", "quantity", "quantityUnit", "priceData", "imageUrl", "discountPercent"} {
		if v, ok := item[key]; ok {
			summary[key] = v
		}
	}
	return summary
}

func IsNoProducts(err error) bool {
	return errors.Is(err, ErrNoProductsFound)
}
