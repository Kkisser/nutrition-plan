package estimator

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"price-service/internal/cache"
	"price-service/internal/config"
	"price-service/internal/edadeal"
	"price-service/internal/logger"
	"price-service/internal/normalizer"
)

var ErrEmptyItems = errors.New("items must not be empty")

type Estimator struct {
	cfg      config.Config
	client   edadeal.ProductClient
	retailer edadeal.RetailerInfo
	cache    *cache.MemoryCache
	log      logger.Logger
}

type offerLookupResult struct {
	Offers   []edadeal.ProductOffer
	Rejected []edadeal.RejectedCandidate
	Reason   string
}

type EstimateOptions struct {
	RequestID           string
	IncludeDebug        bool
	IncludeAlternatives bool
	SelectedShopUUID    string
}

func New(cfg config.Config, client edadeal.ProductClient, retailer edadeal.RetailerInfo, cacheStore *cache.MemoryCache, log logger.Logger) *Estimator {
	return &Estimator{cfg: cfg, client: client, retailer: retailer, cache: cacheStore, log: log}
}

func (e *Estimator) Estimate(ctx context.Context, req EstimateRequest, options EstimateOptions) (EstimateResponse, error) {
	if len(req.Items) == 0 {
		return EstimateResponse{}, ErrEmptyItems
	}
	selectedShopUUID := req.effectiveShopUUID()
	if override := strings.TrimSpace(options.SelectedShopUUID); override != "" {
		selectedShopUUID = override
	}
	options.SelectedShopUUID = selectedShopUUID
	resp := EstimateResponse{
		RequestID:        options.RequestID,
		PriceType:        "estimated_reference_price",
		PricingScope:     "nearest_shops_range",
		SelectedShopUUID: selectedShopUUID,
		Retailer:         e.retailer.Name,
		RetailerSlug:     e.retailer.Slug,
		Currency:         "RUB",
		Items:            make([]EstimatedItem, 0, len(req.Items)),
		UnpricedItems:    make([]UnpricedItem, 0),
	}
	if resp.SelectedShopUUID != "" {
		resp.PricingScope = "selected_shop"
	}
	if resp.Retailer == "" {
		resp.Retailer = e.cfg.RetailerName
	}
	if resp.RetailerSlug == "" {
		resp.RetailerSlug = e.cfg.RetailerSlug
	}

	for _, item := range req.Items {
		estimated := e.estimateItem(ctx, options.RequestID, item, options)
		resp.Items = append(resp.Items, estimated)
		if estimated.Status == "priced" && estimated.SelectedOption != nil {
			resp.PricedItemsCount++
			resp.TotalPrice += estimated.SelectedOption.TotalPrice
			resp.MinTotalPrice += estimated.SelectedOption.MinTotalPrice
			resp.MaxTotalPrice += estimated.SelectedOption.MaxTotalPrice
		} else {
			resp.UnpricedItemsCount++
			resp.UnpricedItems = append(resp.UnpricedItems, UnpricedItem{
				IngredientName:  item.IngredientName,
				RequestedAmount: estimated.RequestedAmount,
				RequestedUnit:   estimated.RequestedUnit,
				Reason:          estimated.Status,
			})
		}
	}
	resp.TotalPrice = roundMoney(resp.TotalPrice)
	resp.MinTotalPrice = roundMoney(resp.MinTotalPrice)
	resp.MaxTotalPrice = roundMoney(resp.MaxTotalPrice)
	resp.PriceRange = PriceRange{MinPrice: resp.MinTotalPrice, MaxPrice: resp.MaxTotalPrice}
	resp.IsFullyPriced = resp.UnpricedItemsCount == 0
	switch {
	case resp.PricedItemsCount == len(resp.Items):
		resp.Status = "ok"
	case resp.PricedItemsCount > 0:
		resp.Status = "partial"
	default:
		resp.Status = "failed"
	}
	resp.CalculatedAt = time.Now().UTC().Format(time.RFC3339)
	return resp, nil
}

func (r EstimateRequest) effectiveShopUUID() string {
	if shopUUID := strings.TrimSpace(r.SelectedShopUUID); shopUUID != "" {
		return shopUUID
	}
	return strings.TrimSpace(r.ShopUUID)
}

func (e *Estimator) estimateItem(ctx context.Context, requestID string, item ShoppingListItem, options EstimateOptions) EstimatedItem {
	query := normalizer.NormalizeQuery(item.IngredientName)
	selectedShopUUID := strings.TrimSpace(options.SelectedShopUUID)
	estimated := EstimatedItem{
		IngredientName:  item.IngredientName,
		RequestedAmount: item.Amount,
		RequestedUnit:   item.Unit,
		Query:           query,
	}

	amount, unit, err := normalizer.NormalizeInputUnit(item.Amount, item.Unit)
	if err != nil {
		estimated.Status = "invalid_unit"
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		return estimated
	}
	estimated.RequestedAmount = amount
	estimated.RequestedUnit = unit
	if strings.TrimSpace(item.IngredientName) == "" || query == "" || amount <= 0 {
		estimated.Status = "parse_error"
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		return estimated
	}

	lookup, err := e.lookupOffers(ctx, query)
	if err != nil {
		if edadeal.IsNoProducts(err) {
			estimated.Status = "no_products_found"
			estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
			if options.IncludeDebug {
				estimated.Debug = buildDebug(nil, lookup.Rejected, selectedShopUUID, pricingScope(selectedShopUUID))
			}
			return estimated
		}
		estimated.Status = "api_error"
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		if options.IncludeDebug {
			debug := buildDebug(nil, lookup.Rejected, selectedShopUUID, pricingScope(selectedShopUUID))
			debug["error"] = err.Error()
			estimated.Debug = debug
		}
		return estimated
	}
	if lookup.Reason == "no_products_found" {
		estimated.Status = "no_products_found"
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		if options.IncludeDebug {
			estimated.Debug = buildDebug(nil, lookup.Rejected, selectedShopUUID, pricingScope(selectedShopUUID))
		}
		return estimated
	}

	compatible := filterCompatible(lookup.Offers, unit)
	compatibleBeforeShopFilter := len(compatible)
	if selectedShopUUID != "" {
		compatible = filterBySelectedShop(compatible, selectedShopUUID)
	}
	if len(compatible) == 0 {
		estimated.Status = "no_compatible_products_found"
		if selectedShopUUID != "" && compatibleBeforeShopFilter > 0 {
			estimated.Status = "selected_shop_price_not_found"
		}
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		if options.IncludeDebug {
			estimated.Debug = buildDebug(lookup.Offers, lookup.Rejected, selectedShopUUID, pricingScope(selectedShopUUID))
		}
		return estimated
	}

	required := int(math.Ceil(amount))
	selected, err := OptimizePackages(required, unit, compatible)
	if err != nil {
		estimated.Status = "no_compatible_products_found"
		estimated.ErrorMessage = errorMessageForStatus(estimated.Status)
		return estimated
	}
	estimated.Status = "priced"
	estimated.ErrorMessage = nil
	estimated.SelectedOption = selected
	estimated.PriceRange = &selected.PriceRange
	if options.IncludeAlternatives || options.IncludeDebug {
		estimated.Alternatives = BuildAlternatives(required, compatible)
	}
	if options.IncludeDebug {
		estimated.Debug = buildDebug(compatible, lookup.Rejected, selectedShopUUID, pricingScope(selectedShopUUID))
	}
	e.log.Infof("request_id=%s endpoint=estimate ingredient=%q query=%q pricing_scope=%s shop_uuid=%q accepted=%d rejected=%d min_total=%.2f max_total=%.2f",
		requestID, item.IngredientName, query, pricingScope(selectedShopUUID), selectedShopUUID, len(compatible), len(lookup.Rejected), selected.MinTotalPrice, selected.MaxTotalPrice)
	return estimated
}

func errorMessageForStatus(status string) *string {
	messages := map[string]string{
		"unpriced":                      "Не удалось оценить позицию.",
		"invalid_unit":                  "Недопустимая единица измерения. Используйте g, ml или pcs.",
		"api_error":                     "Ошибка при обращении к Edadil.",
		"parse_error":                   "Не удалось разобрать позицию списка покупок.",
		"no_products_found":             "По запросу не найдены товары.",
		"no_compatible_products_found":  "Не найдены товары с совместимой фасовкой.",
		"selected_shop_price_not_found": "Для выбранного магазина не найдена цена по совместимым товарам.",
		"package_unknown":               "Не удалось определить фасовку товара.",
		"incompatible_unit":             "Единица товара несовместима с запрошенной единицей.",
	}
	message, ok := messages[status]
	if !ok {
		message = "Позиция не оценена."
	}
	return &message
}

func (e *Estimator) lookupOffers(ctx context.Context, query string) (offerLookupResult, error) {
	key := e.cfg.CacheKeyPrefix() + ":" + query
	if cached, ok := e.cache.Get(key); ok {
		if lookup, ok := cached.(offerLookupResult); ok {
			if lookup.Reason == "no_products_found" {
				return lookup, edadeal.ErrNoProductsFound
			}
			return lookup, nil
		}
	}

	search, err := e.client.SearchProducts(ctx, e.retailer.UUID, query)
	if err != nil {
		lookup := offerLookupResult{Rejected: search.Rejected}
		if edadeal.IsNoProducts(err) {
			lookup.Reason = "no_products_found"
			e.cache.Set(key, lookup, time.Minute)
		}
		return lookup, err
	}

	candidates := search.Candidates
	if len(candidates) > e.cfg.DetailCandidatesLimit {
		candidates = candidates[:e.cfg.DetailCandidatesLimit]
	}

	lookup := offerLookupResult{Rejected: append([]edadeal.RejectedCandidate(nil), search.Rejected...)}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.cfg.MaxConcurrentItemDetail)

	for _, candidate := range candidates {
		candidate := candidate
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			offer, rejected, err := e.client.GetItemDetail(ctx, e.retailer, query, candidate)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if rejected.Reason == "" {
					rejected = edadeal.RejectedCandidate{
						Reason:        err.Error(),
						Title:         candidate.Title,
						ItemUUID:      candidate.ItemUUID,
						BaseOfferUUID: candidate.BaseOfferUUID,
					}
				}
				lookup.Rejected = append(lookup.Rejected, rejected)
				return
			}
			lookup.Offers = append(lookup.Offers, offer)
		}()
	}
	wg.Wait()
	e.cache.Set(key, lookup, e.cfg.CacheTTL)
	return lookup, nil
}

func filterCompatible(offers []edadeal.ProductOffer, requestedUnit string) []edadeal.ProductOffer {
	out := make([]edadeal.ProductOffer, 0, len(offers))
	for _, offer := range offers {
		if offer.Price <= 0 || offer.PackageAmount <= 0 {
			continue
		}
		if !normalizer.CompatibleUnits(offer.PackageUnit, requestedUnit) {
			continue
		}
		if offer.PackageUnit != "g" && offer.PackageUnit != "ml" && offer.PackageUnit != "pcs" {
			continue
		}
		out = append(out, offer)
	}
	return out
}

func filterBySelectedShop(offers []edadeal.ProductOffer, shopUUID string) []edadeal.ProductOffer {
	out := make([]edadeal.ProductOffer, 0, len(offers))
	for _, offer := range offers {
		shopPrice, oldPrice, ok := priceForShop(offer, shopUUID)
		if !ok {
			continue
		}
		offer.Price = shopPrice
		offer.PriceRangeMax = &shopPrice
		offer.OldPrice = oldPrice
		offer.PriceSource = "selected_shop_price"
		offer.SelectedShopUUID = shopUUID
		out = append(out, offer)
	}
	return out
}

func priceForShop(offer edadeal.ProductOffer, shopUUID string) (float64, *float64, bool) {
	for _, shop := range offer.NearestShops {
		if shop.ShopUUID != shopUUID || shop.Price == nil || *shop.Price <= 0 {
			continue
		}
		return roundMoney(*shop.Price), shop.OldPrice, true
	}
	return 0, nil, false
}

func pricingScope(shopUUID string) string {
	if strings.TrimSpace(shopUUID) != "" {
		return "selected_shop"
	}
	return "nearest_shops_range"
}

func buildDebug(offers []edadeal.ProductOffer, rejected []edadeal.RejectedCandidate, selectedShopUUID string, scope string) map[string]any {
	if rejected == nil {
		rejected = []edadeal.RejectedCandidate{}
	}
	priceSources := make([]map[string]any, 0, len(offers))
	detailURLs := make([]string, 0, len(offers))
	nearest := make(map[string][]edadeal.NearestShop)
	raw := make([]map[string]any, 0, len(offers))
	for _, offer := range offers {
		priceSources = append(priceSources, map[string]any{
			"item_uuid":          offer.ItemUUID,
			"price_source":       offer.PriceSource,
			"price":              offer.Price,
			"price_min":          offer.Price,
			"price_max":          offer.EffectiveMaxPrice(),
			"selected_shop_uuid": offer.SelectedShopUUID,
		})
		if offer.DetailURL != "" {
			detailURLs = append(detailURLs, offer.DetailURL)
		}
		if len(offer.NearestShops) > 0 {
			nearest[offer.ItemUUID] = offer.NearestShops
		}
		if offer.RawFieldSummary != nil {
			raw = append(raw, offer.RawFieldSummary)
		}
	}
	return map[string]any{
		"pricing_scope":        scope,
		"selected_shop_uuid":   selectedShopUUID,
		"detail_endpoint_urls": detailURLs,
		"nearest_shops":        nearest,
		"rejected_candidates":  rejected,
		"raw_field_summary":    raw,
		"price_sources":        priceSources,
	}
}
