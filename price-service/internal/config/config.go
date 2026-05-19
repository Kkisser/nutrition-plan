package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort                int           `json:"http_port"`
	Debug                   bool          `json:"debug"`
	DebugMaskSecrets        bool          `json:"debug_mask_secrets"`
	EdadealBaseURL          string        `json:"edadeal_base_url"`
	RetailerSlug            string        `json:"edadeal_retailer_slug"`
	RetailerName            string        `json:"edadeal_retailer_name"`
	AppID                   string        `json:"edadeal_app_id"`
	AppVersion              string        `json:"edadeal_app_version"`
	Platform                string        `json:"edadeal_platform"`
	OSVersion               string        `json:"edadeal_os_version"`
	Origin                  string        `json:"edadeal_origin"`
	Referer                 string        `json:"edadeal_referer"`
	DUID                    string        `json:"edadeal_duid"`
	Preset                  string        `json:"edadeal_preset"`
	CountryGeoID            int           `json:"edadeal_country_geo_id"`
	GeoID                   int           `json:"edadeal_geo_id"`
	GeoPath                 []int         `json:"edadeal_geo_path"`
	Latitude                float64       `json:"edadeal_latitude"`
	Longitude               float64       `json:"edadeal_longitude"`
	ChercherArea            string        `json:"edadeal_chercher_area"`
	UseGoAreaGenerator      bool          `json:"edadeal_use_go_area_generator"`
	SearchLimit             int           `json:"search_limit"`
	DetailCandidatesLimit   int           `json:"detail_candidates_limit"`
	MaxShops                int           `json:"max_shops"`
	RequestTimeout          time.Duration `json:"-"`
	RequestTimeoutSeconds   int           `json:"request_timeout_seconds"`
	CacheTTL                time.Duration `json:"-"`
	CacheTTLSeconds         int           `json:"cache_ttl_seconds"`
	MaxConcurrentItemDetail int           `json:"max_concurrent_item_details"`
}

func Default() Config {
	return Config{
		HTTPPort:                8085,
		Debug:                   true,
		DebugMaskSecrets:        true,
		EdadealBaseURL:          "https://search.edadeal.io",
		RetailerSlug:            "5ka",
		RetailerName:            "Пятёрочка",
		AppID:                   "edadeal",
		AppVersion:              "1.92.0",
		Platform:                "desktop",
		OSVersion:               "1.0.0",
		Origin:                  "https://edadeal.ru",
		Referer:                 "https://edadeal.ru/",
		Preset:                  "moscow",
		CountryGeoID:            225,
		GeoID:                   213,
		GeoPath:                 []int{225, 3, 1, 213},
		Latitude:                55.6965,
		Longitude:               37.5,
		UseGoAreaGenerator:      true,
		SearchLimit:             20,
		DetailCandidatesLimit:   8,
		MaxShops:                10,
		RequestTimeoutSeconds:   15,
		CacheTTLSeconds:         600,
		MaxConcurrentItemDetail: 4,
	}
}

func Load(root string) (Config, error) {
	cfg := Default()
	if root == "" {
		root = "."
	}

	if err := loadJSON(filepath.Join(root, "config.json"), &cfg); err != nil {
		return cfg, err
	}

	env, err := readDotEnv(filepath.Join(root, ".env"))
	if err != nil {
		return cfg, err
	}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	latSet := hasEnv(env, "EDADEAL_LATITUDE")
	lonSet := hasEnv(env, "EDADEAL_LONGITUDE")
	geoIDSet := hasEnv(env, "EDADEAL_GEO_ID")
	geoPathSet := hasEnv(env, "EDADEAL_GEO_PATH")

	applyString(env, "EDADEAL_PRESET", &cfg.Preset)
	if cfg.Preset != "" {
		if p, ok := Preset(cfg.Preset); ok {
			if !geoIDSet {
				cfg.GeoID = p.GeoID
			}
			if !geoPathSet {
				cfg.GeoPath = append([]int(nil), p.GeoPath...)
			}
			cfg.CountryGeoID = p.CountryGeoID
			if !latSet && p.DefaultLat != 0 {
				cfg.Latitude = p.DefaultLat
			}
			if !lonSet && p.DefaultLon != 0 {
				cfg.Longitude = p.DefaultLon
			}
		}
	}

	applyInt(env, "HTTP_PORT", &cfg.HTTPPort)
	applyBool(env, "DEBUG", &cfg.Debug)
	applyBool(env, "DEBUG_MASK_SECRETS", &cfg.DebugMaskSecrets)
	applyString(env, "EDADEAL_BASE_URL", &cfg.EdadealBaseURL)
	applyString(env, "EDADEAL_RETAILER_SLUG", &cfg.RetailerSlug)
	applyString(env, "EDADEAL_RETAILER_NAME", &cfg.RetailerName)
	applyString(env, "EDADEAL_APP_ID", &cfg.AppID)
	applyString(env, "EDADEAL_APP_VERSION", &cfg.AppVersion)
	applyString(env, "EDADEAL_PLATFORM", &cfg.Platform)
	applyString(env, "EDADEAL_OS_VERSION", &cfg.OSVersion)
	applyString(env, "EDADEAL_ORIGIN", &cfg.Origin)
	applyString(env, "EDADEAL_REFERER", &cfg.Referer)
	applyString(env, "EDADEAL_DUID", &cfg.DUID)
	applyInt(env, "EDADEAL_COUNTRY_GEO_ID", &cfg.CountryGeoID)
	applyInt(env, "EDADEAL_GEO_ID", &cfg.GeoID)
	applyIntSlice(env, "EDADEAL_GEO_PATH", &cfg.GeoPath)
	applyFloat(env, "EDADEAL_LATITUDE", &cfg.Latitude)
	applyFloat(env, "EDADEAL_LONGITUDE", &cfg.Longitude)
	applyString(env, "EDADEAL_CHERCHER_AREA", &cfg.ChercherArea)
	applyBool(env, "EDADEAL_USE_GO_AREA_GENERATOR", &cfg.UseGoAreaGenerator)
	applyInt(env, "SEARCH_LIMIT", &cfg.SearchLimit)
	applyInt(env, "DETAIL_CANDIDATES_LIMIT", &cfg.DetailCandidatesLimit)
	applyInt(env, "MAX_SHOPS", &cfg.MaxShops)
	applyInt(env, "REQUEST_TIMEOUT_SECONDS", &cfg.RequestTimeoutSeconds)
	applyInt(env, "CACHE_TTL_SECONDS", &cfg.CacheTTLSeconds)
	applyInt(env, "MAX_CONCURRENT_ITEM_DETAILS", &cfg.MaxConcurrentItemDetail)

	if cfg.SearchLimit <= 0 {
		cfg.SearchLimit = 20
	}
	if cfg.DetailCandidatesLimit <= 0 {
		cfg.DetailCandidatesLimit = 8
	}
	if cfg.MaxShops <= 0 {
		cfg.MaxShops = 10
	}
	if cfg.RequestTimeoutSeconds <= 0 {
		cfg.RequestTimeoutSeconds = 15
	}
	if cfg.CacheTTLSeconds <= 0 {
		cfg.CacheTTLSeconds = 600
	}
	if cfg.MaxConcurrentItemDetail <= 0 {
		cfg.MaxConcurrentItemDetail = 4
	}
	cfg.RequestTimeout = time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	cfg.CacheTTL = time.Duration(cfg.CacheTTLSeconds) * time.Second
	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func (c Config) CacheKeyPrefix() string {
	return fmt.Sprintf("%s:%d:%.4f:%.4f", c.RetailerSlug, c.GeoID, c.Latitude, c.Longitude)
}

type CityPreset struct {
	GeoID        int
	GeoPath      []int
	CountryGeoID int
	DefaultLat   float64
	DefaultLon   float64
}

func Preset(name string) (CityPreset, bool) {
	presets := map[string]CityPreset{
		"moscow":       {GeoID: 213, GeoPath: []int{225, 3, 1, 213}, CountryGeoID: 225, DefaultLat: 55.6965, DefaultLon: 37.5},
		"spb":          {GeoID: 2, GeoPath: []int{225, 3, 10174, 2}, CountryGeoID: 225, DefaultLat: 59.9311, DefaultLon: 30.3609},
		"novosibirsk":  {GeoID: 65, GeoPath: []int{225, 3, 11316, 65}, CountryGeoID: 225, DefaultLat: 55.0084, DefaultLon: 82.9357},
		"ekaterinburg": {GeoID: 54, GeoPath: []int{225, 3, 11162, 54}, CountryGeoID: 225, DefaultLat: 56.8389, DefaultLon: 60.6057},
		"kazan":        {GeoID: 43, GeoPath: []int{225, 3, 11119, 43}, CountryGeoID: 225, DefaultLat: 55.7961, DefaultLon: 49.1064},
	}
	p, ok := presets[strings.ToLower(strings.TrimSpace(name))]
	return p, ok
}

func loadJSON(path string, cfg *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	return json.Unmarshal(b, cfg)
}

func readDotEnv(path string) (map[string]string, error) {
	env := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return env, nil
		}
		return env, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		env[key] = value
	}
	return env, scanner.Err()
}

func hasEnv(env map[string]string, key string) bool {
	_, ok := env[key]
	return ok
}

func applyString(env map[string]string, key string, target *string) {
	if v, ok := env[key]; ok {
		*target = strings.TrimSpace(v)
	}
}

func applyBool(env map[string]string, key string, target *bool) {
	if v, ok := env[key]; ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "y", "on":
			*target = true
		case "0", "false", "no", "n", "off":
			*target = false
		}
	}
}

func applyInt(env map[string]string, key string, target *int) {
	if v, ok := env[key]; ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			*target = parsed
		}
	}
}

func applyFloat(env map[string]string, key string, target *float64) {
	if v, ok := env[key]; ok {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			*target = parsed
		}
	}
}

func applyIntSlice(env map[string]string, key string, target *[]int) {
	v, ok := env[key]
	if !ok {
		return
	}
	parts := strings.Split(v, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return
		}
		out = append(out, parsed)
	}
	*target = out
}
