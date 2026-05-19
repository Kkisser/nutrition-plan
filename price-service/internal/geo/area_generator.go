package geo

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"sort"

	h3 "github.com/lightboxre/h3-go"
)

const overallRadiusDelta = 0.3

type AreaGenerator interface {
	Generate(lat, lon float64, geoID int, geoPath []int) (GeneratedArea, error)
}

type H3AreaGenerator struct{}

type GeneratedArea struct {
	Encoded string
	Area    ChercherArea
}

type ChercherArea struct {
	SearchGeoID   int          `json:"search_geo_id"`
	SearchGeoPath []int        `json:"search_geo_path"`
	Country       *string      `json:"country"`
	IsReal        bool         `json:"is_real"`
	Regions       []AreaRegion `json:"regions"`
	H3Cells       []uint64     `json:"h3_cells"`
	Radius        float64      `json:"radius"`
	Center        []float64    `json:"center"`
}

type AreaRegion struct {
	Filter AreaFilter `json:"filter"`
	H3     []uint64   `json:"h3"`
	Radius float64    `json:"radius"`
}

type AreaFilter struct {
	Formats []string `json:"formats"`
}

type storeConfig struct {
	Formats []string
	Radius  float64
	H3Res   int
}

var storeConfigs = []storeConfig{
	{Formats: []string{"corner-store"}, Radius: 2.6, H3Res: 7},
	{Formats: []string{"others"}, Radius: 7.1, H3Res: 6},
	{Formats: []string{"grocery-store"}, Radius: 13.8, H3Res: 6},
}

func (g H3AreaGenerator) Generate(lat, lon float64, geoID int, geoPath []int) (GeneratedArea, error) {
	regions := make([]AreaRegion, 0, len(storeConfigs))
	all := make(map[uint64]struct{})
	maxRadius := 0.0

	for _, cfg := range storeConfigs {
		cells := latLonToH3Disk(lat, lon, cfg.Radius, cfg.H3Res)
		regions = append(regions, AreaRegion{
			Filter: AreaFilter{Formats: append([]string(nil), cfg.Formats...)},
			H3:     cells,
			Radius: cfg.Radius,
		})
		for _, cell := range cells {
			all[cell] = struct{}{}
		}
		if cfg.Radius > maxRadius {
			maxRadius = cfg.Radius
		}
	}

	h3Cells := make([]uint64, 0, len(all))
	for cell := range all {
		h3Cells = append(h3Cells, cell)
	}
	sort.Slice(h3Cells, func(i, j int) bool { return h3Cells[i] < h3Cells[j] })

	area := ChercherArea{
		SearchGeoID:   geoID,
		SearchGeoPath: append([]int(nil), geoPath...),
		Country:       nil,
		IsReal:        false,
		Regions:       regions,
		H3Cells:       h3Cells,
		Radius:        roundTo(maxRadius+overallRadiusDelta, 1),
		Center:        []float64{roundTo(lat, 4), roundTo(lon, 1)},
	}

	raw, err := json.Marshal(area)
	if err != nil {
		return GeneratedArea{}, err
	}
	return GeneratedArea{
		Encoded: base64.StdEncoding.EncodeToString(raw),
		Area:    area,
	}, nil
}

func latLonToH3Disk(lat, lon, radiusKm float64, res int) []uint64 {
	center := h3.LatLngToCell(lat, lon, res)
	edgeKm := h3.EdgeLengthKm(res)
	k := int(math.Round(radiusKm / (edgeKm * 1.5)))
	if k < 1 {
		k = 1
	}
	cells, err := h3.GridDisk(center, k)
	if err != nil {
		cells = []h3.Cell{center}
	}
	out := make([]uint64, 0, len(cells))
	for _, cell := range cells {
		if cell == 0 {
			continue
		}
		out = append(out, uint64(cell))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func roundTo(v float64, digits int) float64 {
	pow := math.Pow10(digits)
	return math.Round(v*pow) / pow
}
