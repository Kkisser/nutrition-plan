package normalizer

import (
	"errors"
	"math"
	"strings"
)

var ErrInvalidUnit = errors.New("invalid_unit")

func NormalizeInputUnit(amount float64, unit string) (float64, string, error) {
	unit = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(unit, "ё", "е")))
	switch unit {
	case "g":
		return amount, "g", nil
	case "ml":
		return amount, "ml", nil
	case "pcs":
		return amount, "pcs", nil
	default:
		return 0, "", ErrInvalidUnit
	}
}

func NormalizePackageUnit(amount float64, unit string) (int, string, error) {
	unit = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(unit, "ё", "е")))
	unit = strings.TrimSuffix(unit, ".")
	switch unit {
	case "kg", "кг":
		return positiveRounded(amount * 1000), "g", nil
	case "g", "г":
		return positiveRounded(amount), "g", nil
	case "l", "л":
		return positiveRounded(amount * 1000), "ml", nil
	case "ml", "мл":
		return positiveRounded(amount), "ml", nil
	case "pcs", "шт", "штук":
		return positiveRounded(amount), "pcs", nil
	default:
		return 0, "", ErrInvalidUnit
	}
}

func CompatibleUnits(a, b string) bool {
	return a == b
}

func positiveRounded(v float64) int {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int(math.Round(v))
}
