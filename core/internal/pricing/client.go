package pricing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"nutrition-core/internal/domain"
)

// Client — HTTP-клиент к price-service.
type Client struct {
	baseURL string
	http    *http.Client
}

// Disabled означает, что URL price-service не сконфигурирован.
// В этом случае оценка стоимости недоступна, но это не ошибка
// (docs/КОНТРАКТ_API.md §1: «опционально, если пользователь выбрал
// магазин»).
var ErrDisabled = errors.New("pricing: price-service URL is not configured")

// ErrUpstream — price-service вернул 5xx или сетевую ошибку.
var ErrUpstream = errors.New("pricing: upstream failure")

// ErrBadRequest — 400 от price-service (структурная ошибка запроса).
var ErrBadRequest = errors.New("pricing: bad request")

// New создаёт клиента по явному URL. Пустой URL → ErrDisabled при попытке
// вызова. Таймаут 15 секунд согласован с REQUEST_TIMEOUT_SECONDS
// price-service.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FromEnv создаёт клиента, читая PRICE_SERVICE_URL.
// Если переменная пуста — клиент возвращает ErrDisabled при вызовах.
func FromEnv() *Client {
	return New(os.Getenv("PRICE_SERVICE_URL"))
}

// Estimate — POST {baseURL}/estimate. shopUUID может быть пустым:
// price-service вернёт вилку по ближайшим магазинам.
//
// requestID попадает в заголовок X-Request-ID, что позволяет
// связать логи бэкенда и price-service. Если пуст — заголовок не
// проставляется.
func (c *Client) Estimate(
	ctx context.Context,
	items []domain.ShoppingItem,
	shopUUID string,
	requestID string,
) (*EstimateResponse, error) {
	if c.baseURL == "" {
		return nil, ErrDisabled
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("pricing: items list is empty")
	}

	req := EstimateRequest{
		SelectedShopUUID: shopUUID,
		Items:            itemsFromShopping(items),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := c.baseURL + "/estimate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		httpReq.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusBadRequest:
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, string(raw))
	case resp.StatusCode >= 500:
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: HTTP %d %s",
			ErrUpstream, resp.StatusCode, string(raw))
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("pricing: unexpected HTTP %d", resp.StatusCode)
	}

	var out EstimateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &out, nil
}
