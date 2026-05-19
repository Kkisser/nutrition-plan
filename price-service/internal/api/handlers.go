package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"price-service/internal/cache"
	"price-service/internal/config"
	debugrec "price-service/internal/debug"
	"price-service/internal/edadeal"
	"price-service/internal/estimator"
	"price-service/internal/requestid"
)

type Dependencies struct {
	Config    config.Config
	Estimator *estimator.Estimator
	Retailer  edadeal.RetailerInfo
	Cache     *cache.MemoryCache
	Recorder  *debugrec.Recorder
}

type Reloader func(ctx context.Context) (Dependencies, error)

type Runtime struct {
	mu       sync.RWMutex
	deps     Dependencies
	reloader Reloader
}

func NewRuntime(deps Dependencies, reloader Reloader) *Runtime {
	return &Runtime{deps: deps, reloader: reloader}
}

func (r *Runtime) Current() Dependencies {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.deps
}

func (r *Runtime) Health(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deps := r.Current()
	retailer := deps.Retailer.Name
	if retailer == "" {
		retailer = deps.Config.RetailerName
	}
	slug := deps.Retailer.Slug
	if slug == "" {
		slug = deps.Config.RetailerSlug
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Retailer: retailer, RetailerSlug: slug})
}

func (r *Runtime) Estimate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body estimator.EstimateRequest
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if reason := wholeRequestValidationError(body); reason != "" {
		writeError(w, http.StatusBadRequest, reason)
		return
	}
	deps := r.Current()
	includeDebug := req.URL.Query().Get("debug") == "true"
	includeAlternatives := req.URL.Query().Get("include_alternatives") == "true"
	selectedShopUUID := strings.TrimSpace(body.SelectedShopUUID)
	if selectedShopUUID == "" {
		selectedShopUUID = strings.TrimSpace(body.ShopUUID)
	}
	if queryShopUUID := strings.TrimSpace(req.URL.Query().Get("selected_shop_uuid")); queryShopUUID != "" {
		selectedShopUUID = queryShopUUID
	}
	if queryShopUUID := strings.TrimSpace(req.URL.Query().Get("shop_uuid")); queryShopUUID != "" {
		selectedShopUUID = queryShopUUID
	}
	reqID := strings.TrimSpace(req.Header.Get("X-Request-ID"))
	if reqID == "" {
		reqID = requestid.New()
	}
	resp, err := deps.Estimator.Estimate(req.Context(), body, estimator.EstimateOptions{
		RequestID:           reqID,
		IncludeDebug:        includeDebug,
		IncludeAlternatives: includeAlternatives,
		SelectedShopUUID:    selectedShopUUID,
	})
	if err != nil {
		if errors.Is(err, estimator.ErrEmptyItems) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func wholeRequestValidationError(body estimator.EstimateRequest) string {
	if len(body.Items) != 1 {
		return ""
	}
	item := body.Items[0]
	if strings.TrimSpace(item.IngredientName) == "" {
		return "ingredient_name is required"
	}
	if item.Amount <= 0 {
		return "amount must be greater than 0"
	}
	switch strings.TrimSpace(item.Unit) {
	case "g", "ml", "pcs":
		return ""
	default:
		return "unit must be one of: g, ml, pcs"
	}
}

func (r *Runtime) Reload(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.reloader != nil {
		deps, err := r.reloader(req.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		r.mu.Lock()
		r.deps = deps
		r.mu.Unlock()
		writeJSON(w, http.StatusOK, reloadResponse{Status: "ok", CacheCleared: true})
		return
	}
	deps := r.Current()
	if deps.Cache != nil {
		deps.Cache.Clear()
	}
	writeJSON(w, http.StatusOK, reloadResponse{Status: "ok", CacheCleared: true})
}

func (r *Runtime) LastRequests(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deps := r.Current()
	if deps.Recorder == nil {
		writeJSON(w, http.StatusOK, lastRequestsResponse{Requests: []any{}})
		return
	}
	writeJSON(w, http.StatusOK, lastRequestsResponse{Requests: deps.Recorder.List()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
