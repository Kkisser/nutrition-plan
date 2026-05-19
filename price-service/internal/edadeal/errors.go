package edadeal

import "errors"

var (
	ErrRetailerNotFound = errors.New("retailer_not_found")
	ErrHTTP             = errors.New("http_error")
	ErrInvalidJSON      = errors.New("invalid_json")
	ErrUUIDMissing      = errors.New("uuid_missing")
	ErrNoProductsFound  = errors.New("no_products_found")
	ErrMissingPrice     = errors.New("reject_missing_price")
	ErrMissingAmount    = errors.New("reject_missing_amount")
)
