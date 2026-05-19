package geo

type Preset struct {
	Name         string
	GeoID        int
	GeoPath      []int
	CountryGeoID int
	DefaultLat   float64
	DefaultLon   float64
}

var Presets = map[string]Preset{
	"moscow":       {Name: "moscow", GeoID: 213, GeoPath: []int{225, 3, 1, 213}, CountryGeoID: 225, DefaultLat: 55.6965, DefaultLon: 37.5},
	"spb":          {Name: "spb", GeoID: 2, GeoPath: []int{225, 3, 10174, 2}, CountryGeoID: 225, DefaultLat: 59.9311, DefaultLon: 30.3609},
	"novosibirsk":  {Name: "novosibirsk", GeoID: 65, GeoPath: []int{225, 3, 11316, 65}, CountryGeoID: 225, DefaultLat: 55.0084, DefaultLon: 82.9357},
	"ekaterinburg": {Name: "ekaterinburg", GeoID: 54, GeoPath: []int{225, 3, 11162, 54}, CountryGeoID: 225, DefaultLat: 56.8389, DefaultLon: 60.6057},
	"kazan":        {Name: "kazan", GeoID: 43, GeoPath: []int{225, 3, 11119, 43}, CountryGeoID: 225, DefaultLat: 55.7961, DefaultLon: 49.1064},
}
