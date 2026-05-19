package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"price-service/internal/api"
	"price-service/internal/cache"
	"price-service/internal/config"
	debugrec "price-service/internal/debug"
	"price-service/internal/edadeal"
	"price-service/internal/estimator"
	"price-service/internal/geo"
	"price-service/internal/logger"
	"price-service/internal/requestid"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		if err := serve(); err != nil {
			log.Fatal(err)
		}
	case "estimate":
		if err := estimateCLI(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "header":
		if err := headerCLI(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func serve() error {
	deps, err := buildDependencies(".")
	if err != nil {
		return err
	}
	runtime := api.NewRuntime(deps, func(ctx context.Context) (api.Dependencies, error) {
		return buildDependencies(".")
	})
	log.Printf("price-service listening on http://localhost:%d", deps.Config.HTTPPort)
	return http.ListenAndServe(deps.Config.Address(), api.Router(runtime))
}

func estimateCLI(args []string) error {
	fs := flag.NewFlagSet("estimate", flag.ExitOnError)
	input := fs.String("input", "", "path to request JSON")
	debug := fs.Bool("debug", false, "include debug fields")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var raw []byte
	var err error
	if *input == "" || *input == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(*input)
	}
	if err != nil {
		return err
	}

	var req estimator.EstimateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return err
	}

	deps, err := buildDependencies(".")
	if err != nil {
		return err
	}
	resp, err := deps.Estimator.Estimate(context.Background(), req, estimator.EstimateOptions{
		RequestID:           requestid.New(),
		IncludeDebug:        *debug,
		IncludeAlternatives: *debug,
	})
	if err != nil {
		return err
	}
	return writePrettyJSON(os.Stdout, resp)
}

func headerCLI(args []string) error {
	cfg, err := config.Load(".")
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("header", flag.ExitOnError)
	preset := fs.String("preset", cfg.Preset, "city preset")
	lat := fs.Float64("lat", cfg.Latitude, "latitude")
	lon := fs.Float64("lon", cfg.Longitude, "longitude")
	jsonFlag := fs.Bool("json", false, "include decoded chercher area JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	cfg.Preset = *preset
	if p, ok := config.Preset(cfg.Preset); ok {
		cfg.GeoID = p.GeoID
		cfg.GeoPath = append([]int(nil), p.GeoPath...)
		cfg.CountryGeoID = p.CountryGeoID
		if !visited["lat"] {
			*lat = p.DefaultLat
		}
		if !visited["lon"] {
			*lon = p.DefaultLon
		}
	}
	cfg.Latitude = *lat
	cfg.Longitude = *lon
	cfg.UseGoAreaGenerator = true

	headers, err := geo.BuildHeaders(cfg, geo.H3AreaGenerator{})
	if err != nil {
		return err
	}
	out := map[string]any{"headers": headers.Headers}
	if *jsonFlag && headers.Area != nil {
		out["decoded_chercher_area"] = headers.Area
	}
	return writePrettyJSON(os.Stdout, out)
}

func buildDependencies(root string) (api.Dependencies, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return api.Dependencies{}, err
	}
	headers, err := geo.BuildHeaders(cfg, geo.H3AreaGenerator{})
	if err != nil {
		return api.Dependencies{}, err
	}
	recorder := debugrec.NewRecorder(100)
	client := edadeal.NewClient(cfg, headers, recorder)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout+5*time.Second)
	defer cancel()
	retailer, err := client.GetRetailerInfo(ctx)
	if err != nil {
		return api.Dependencies{}, err
	}
	if retailer.UUID == "" {
		return api.Dependencies{}, edadeal.ErrUUIDMissing
	}
	cacheStore := cache.NewMemoryCache()
	logg := logger.New(cfg.Debug)
	est := estimator.New(cfg, client, retailer, cacheStore, logg)
	return api.Dependencies{
		Config:    cfg,
		Estimator: est,
		Retailer:  retailer,
		Cache:     cacheStore,
		Recorder:  recorder,
	}, nil
}

func writePrettyJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  price-service serve")
	fmt.Fprintln(os.Stderr, "  price-service estimate --input ./data/sample_request.json")
	fmt.Fprintln(os.Stderr, "  price-service header --preset moscow --lat 55.6965 --lon 37.5 --json")
}
