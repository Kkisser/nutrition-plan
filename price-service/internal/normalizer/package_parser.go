package normalizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ErrPackageUnknown = errors.New("package_unknown")

var packageRE = regexp.MustCompile(`(?i)(\d+(?:[,.]\d+)?)\s*(кг|kg|г|g|мл|ml|л|l|штук|шт\.?|pcs)(?:\s|$|[),.;])`)

func ParsePackage(amountRaw any, unitRaw any, title string) (int, string, error) {
	if amount, ok := anyToFloat(amountRaw); ok {
		if unit, ok := anyToString(unitRaw); ok {
			if normalizedAmount, normalizedUnit, err := NormalizePackageUnit(amount, unit); err == nil && normalizedAmount > 0 {
				return normalizedAmount, normalizedUnit, nil
			}
		}
	}
	return ParsePackageFromTitle(title)
}

func ParsePackageFromTitle(title string) (int, string, error) {
	normalized := strings.ToLower(strings.ReplaceAll(title, "ё", "е"))
	matches := packageRE.FindAllStringSubmatch(normalized, -1)
	if len(matches) == 0 {
		return 0, "", ErrPackageUnknown
	}
	for _, match := range matches {
		amountText := strings.ReplaceAll(match[1], ",", ".")
		amount, err := strconv.ParseFloat(amountText, 64)
		if err != nil {
			continue
		}
		unit := strings.TrimSuffix(match[2], ".")
		normalizedAmount, normalizedUnit, err := NormalizePackageUnit(amount, unit)
		if err == nil && normalizedAmount > 0 {
			return normalizedAmount, normalizedUnit, nil
		}
	}
	return 0, "", ErrPackageUnknown
}

func anyToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(strings.ReplaceAll(x, ",", "."))
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func anyToString(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", false
	case string:
		return strings.TrimSpace(x), strings.TrimSpace(x) != ""
	default:
		return fmt.Sprint(x), true
	}
}
