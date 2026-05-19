package edadeal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"price-service/internal/config"
	debugrec "price-service/internal/debug"
	"price-service/internal/geo"
)

type ProductClient interface {
	GetRetailerInfo(ctx context.Context) (RetailerInfo, error)
	SearchProducts(ctx context.Context, retailerUUID string, query string) (SearchProductsResult, error)
	GetItemDetail(ctx context.Context, retailer RetailerInfo, query string, candidate SearchCandidate) (ProductOffer, RejectedCandidate, error)
}

type Client struct {
	cfg        config.Config
	httpClient *http.Client
	geoHeaders map[string]string
	recorder   *debugrec.Recorder
}

func NewClient(cfg config.Config, headers geo.HeadersResult, recorder *debugrec.Recorder) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		geoHeaders: headers.Headers,
		recorder:   recorder,
	}
}

func (c *Client) GetRetailerInfo(ctx context.Context) (RetailerInfo, error) {
	q := url.Values{}
	q.Set("disablePlatformSourceExclusion", "true")
	q.Set("retailerSlug", c.cfg.RetailerSlug)
	raw, requestURL, status, dur, err := c.get(ctx, "/api/v4/retailer_info", q, false)
	record := debugrec.RequestRecord{Operation: "get_retailer_info", URL: requestURL, StatusCode: status, DurationMS: dur.Milliseconds()}
	if err != nil {
		record.Error = err.Error()
		c.recorder.Record(record)
		return RetailerInfo{}, err
	}
	info, err := ParseRetailerInfo(raw)
	if err != nil {
		record.Error = err.Error()
		c.recorder.Record(record)
		return RetailerInfo{}, err
	}
	if info.Slug == "" {
		info.Slug = c.cfg.RetailerSlug
	}
	if info.Name == "" {
		info.Name = c.cfg.RetailerName
	}
	c.recorder.Record(record)
	return info, nil
}

func (c *Client) SearchProducts(ctx context.Context, retailerUUID string, query string) (SearchProductsResult, error) {
	q := url.Values{}
	q.Set("checkAdult", "true")
	q.Set("disablePlatformSourceExclusion", "true")
	q.Set("excludeAlcohol", "true")
	q.Set("groupBy", "sku_or_meta")
	q.Set("limit", strconv.Itoa(c.cfg.SearchLimit))
	q.Set("offset", "0")
	q.Set("retailerUuid", retailerUUID)
	q.Set("text", query)
	raw, requestURL, status, dur, err := c.get(ctx, "/api/v4/search", q, true)
	record := debugrec.RequestRecord{Operation: "search_products", URL: requestURL, StatusCode: status, DurationMS: dur.Milliseconds()}
	if err != nil {
		record.Error = err.Error()
		c.recorder.Record(record)
		return SearchProductsResult{}, err
	}
	result, err := ParseSearchProducts(raw)
	record.ItemsCount = result.ItemsCount
	if err != nil {
		record.Error = err.Error()
		c.recorder.Record(record)
		return result, err
	}
	c.recorder.Record(record)
	return result, nil
}

func (c *Client) GetItemDetail(ctx context.Context, retailer RetailerInfo, query string, candidate SearchCandidate) (ProductOffer, RejectedCandidate, error) {
	endpoint := path.Join("/api/v4/item", candidate.ItemUUID)
	q := url.Values{}
	if candidate.BaseOfferUUID != "" {
		q.Set("baseOfferUuid", candidate.BaseOfferUUID)
	}
	q.Set("disablePlatformSourceExclusion", "true")
	q.Set("maxShops", strconv.Itoa(c.cfg.MaxShops))
	q.Set("type", "meta_offer")
	raw, requestURL, status, dur, err := c.get(ctx, endpoint, q, true)
	record := debugrec.RequestRecord{Operation: "get_item_detail", URL: requestURL, StatusCode: status, DurationMS: dur.Milliseconds()}
	defer func() { c.recorder.Record(record) }()
	if err != nil {
		record.Error = err.Error()
		return ProductOffer{}, RejectedCandidate{
			Reason:        "detail_http_error",
			Title:         candidate.Title,
			ItemUUID:      candidate.ItemUUID,
			BaseOfferUUID: candidate.BaseOfferUUID,
			DetailURL:     requestURL,
		}, err
	}
	offer, rejected, err := ParseItemDetail(raw, candidate, retailer, query, requestURL)
	if err != nil {
		record.Error = err.Error()
		return ProductOffer{}, rejected, err
	}
	return offer, RejectedCandidate{}, nil
}

func (c *Client) get(ctx context.Context, endpoint string, q url.Values, withGeo bool) ([]byte, string, int, time.Duration, error) {
	base, err := url.Parse(c.cfg.EdadealBaseURL)
	if err != nil {
		return nil, "", 0, 0, err
	}
	base.Path = strings.TrimRight(base.Path, "/") + endpoint
	base.RawQuery = q.Encode()
	requestURL := base.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, requestURL, 0, 0, err
	}
	for key, value := range c.commonHeaders() {
		req.Header.Set(key, value)
	}
	if withGeo {
		for key, value := range c.geoHeaders {
			req.Header.Set(key, value)
		}
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return nil, requestURL, 0, duration, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, requestURL, resp.StatusCode, duration, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, requestURL, resp.StatusCode, duration, fmt.Errorf("%w: status %d", ErrHTTP, resp.StatusCode)
	}
	return body, requestURL, resp.StatusCode, duration, nil
}

func (c *Client) commonHeaders() map[string]string {
	headers := map[string]string{
		"Accept":        "application/json",
		"Origin":        c.cfg.Origin,
		"Referer":       c.cfg.Referer,
		"x-app-id":      c.cfg.AppID,
		"x-app-version": c.cfg.AppVersion,
		"x-platform":    c.cfg.Platform,
	}
	if c.cfg.OSVersion != "" {
		headers["x-os-version"] = c.cfg.OSVersion
	}
	if c.cfg.DUID != "" {
		headers["edadeal-duid"] = c.cfg.DUID
	}
	return headers
}

func IsHTTPError(err error) bool {
	return errors.Is(err, ErrHTTP)
}
