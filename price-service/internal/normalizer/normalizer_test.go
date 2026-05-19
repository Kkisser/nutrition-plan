package normalizer

import "testing"

func TestNormalizePackageUnit(t *testing.T) {
	tests := []struct {
		name       string
		amount     float64
		unit       string
		wantAmount int
		wantUnit   string
	}{
		{"kg to g", 1, "kg", 1000, "g"},
		{"cyrillic kg to g", 1, "кг", 1000, "g"},
		{"l to ml", 1, "l", 1000, "ml"},
		{"cyrillic l to ml", 1, "л", 1000, "ml"},
		{"pcs", 10, "шт", 10, "pcs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAmount, gotUnit, err := NormalizePackageUnit(tt.amount, tt.unit)
			if err != nil {
				t.Fatalf("NormalizePackageUnit() error = %v", err)
			}
			if gotAmount != tt.wantAmount || gotUnit != tt.wantUnit {
				t.Fatalf("NormalizePackageUnit() = %d %s, want %d %s", gotAmount, gotUnit, tt.wantAmount, tt.wantUnit)
			}
		})
	}
}

func TestParsePackageFromTitle(t *testing.T) {
	tests := []struct {
		title      string
		wantAmount int
		wantUnit   string
	}{
		{"Молоко 950 мл", 950, "ml"},
		{"Молоко 1 л", 1000, "ml"},
		{"Рис 900 г", 900, "g"},
		{"Крупа гречневая 1 кг", 1000, "g"},
		{"Яйцо куриное 10 шт", 10, "pcs"},
		{"Яйца С1 20 шт", 20, "pcs"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			gotAmount, gotUnit, err := ParsePackageFromTitle(tt.title)
			if err != nil {
				t.Fatalf("ParsePackageFromTitle() error = %v", err)
			}
			if gotAmount != tt.wantAmount || gotUnit != tt.wantUnit {
				t.Fatalf("ParsePackageFromTitle() = %d %s, want %d %s", gotAmount, gotUnit, tt.wantAmount, tt.wantUnit)
			}
		})
	}
}
