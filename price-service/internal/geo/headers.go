package geo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"price-service/internal/config"
)

type HeadersResult struct {
	Headers map[string]string
	Area    *ChercherArea
	Encoded string
}

func BuildHeaders(cfg config.Config, generator AreaGenerator) (HeadersResult, error) {
	headers := map[string]string{
		"x-locality-countrygeoid": strconv.Itoa(cfg.CountryGeoID),
		"x-locality-geoid":        strconv.Itoa(cfg.GeoID),
		"x-os-version":            cfg.OSVersion,
		"x-platform":              cfg.Platform,
		"x-position-latitude":     fmt.Sprintf("%g", cfg.Latitude),
		"x-position-longitude":    fmt.Sprintf("%g", cfg.Longitude),
	}

	if cfg.UseGoAreaGenerator {
		if generator == nil {
			generator = H3AreaGenerator{}
		}
		generated, err := generator.Generate(cfg.Latitude, cfg.Longitude, cfg.GeoID, cfg.GeoPath)
		if err != nil {
			return HeadersResult{}, err
		}
		headers["x-edadeal-chercher-area"] = generated.Encoded
		return HeadersResult{Headers: headers, Area: &generated.Area, Encoded: generated.Encoded}, nil
	}

	if cfg.ChercherArea == "" {
		return HeadersResult{}, errors.New("EDADEAL_CHERCHER_AREA is required when EDADEAL_USE_GO_AREA_GENERATOR=false")
	}
	headers["x-edadeal-chercher-area"] = cfg.ChercherArea
	decoded := decodeArea(cfg.ChercherArea)
	return HeadersResult{Headers: headers, Area: decoded, Encoded: cfg.ChercherArea}, nil
}

func decodeArea(encoded string) *ChercherArea {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	var area ChercherArea
	if err := json.Unmarshal(raw, &area); err != nil {
		return nil
	}
	return &area
}
