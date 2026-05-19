package micronutrients

import (
	"math"
	"testing"

	"nutrition-core/internal/domain"
)

const eps = 0.001

func makeSlot(day int, meal domain.MealType, alpha float64, micros map[string]float64) domain.PlanSlot {
	return domain.PlanSlot{
		Day: day, Meal: meal, Portions: alpha,
		Recipe: domain.Recipe{Micronutrients: micros},
	}
}

func TestPlannedDaily(t *testing.T) {
	plan := []domain.PlanSlot{
		makeSlot(1, domain.MealBreakfast, 1.0, map[string]float64{"ca": 200, "fe": 5}),
		makeSlot(1, domain.MealLunch, 2.0, map[string]float64{"ca": 100}),    // 200 после α
		makeSlot(2, domain.MealBreakfast, 1.0, map[string]float64{"fe": 10}), // 10
		makeSlot(2, domain.MealLunch, 0.7, map[string]float64{"ca": 100}),    // 70
	}
	got := PlannedDaily(plan, 7)
	// ca weekly: 200 + 200 + 70 = 470; /7 = 67.14
	// fe weekly: 5 + 10 = 15; /7 = 2.14
	if math.Abs(got["ca"]-470.0/7) > eps {
		t.Errorf("ca daily = %v, want %v", got["ca"], 470.0/7)
	}
	if math.Abs(got["fe"]-15.0/7) > eps {
		t.Errorf("fe daily = %v, want %v", got["fe"], 15.0/7)
	}
}

func TestDeficit(t *testing.T) {
	plan := []domain.PlanSlot{
		makeSlot(1, domain.MealBreakfast, 1.0, map[string]float64{"ca": 100, "fe": 3}),
	}
	// 1 день, 1 слот. Daily: ca=100, fe=3 (за 7 дней — тоже).
	// Норма ca=1000 → Δ = 1000 - 100/7 ≈ 985.71
	// Норма fe=10  → Δ = 10 - 3/7 ≈ 9.57
	// Норма zn=10  → Δ = 10 - 0   = 10
	norms := []domain.MicroNorm{
		{NutrientID: "ca", NormValue: 1000},
		{NutrientID: "fe", NormValue: 10},
		{NutrientID: "zn", NormValue: 10},
	}
	got := Deficit(plan, 7, norms)
	if len(got) != 3 {
		t.Fatalf("got %d deficits, want 3", len(got))
	}
	// Должны быть упорядочены по nutrient_id.
	if got[0].NutrientID != "ca" || got[1].NutrientID != "fe" || got[2].NutrientID != "zn" {
		t.Errorf("order broken: %+v", got)
	}
}

func TestDeficit_FullyCovered(t *testing.T) {
	plan := []domain.PlanSlot{
		makeSlot(1, domain.MealLunch, 1.0, map[string]float64{"ca": 7000}), // weekly 7000 → daily 1000
	}
	norms := []domain.MicroNorm{{NutrientID: "ca", NormValue: 1000}}
	got := Deficit(plan, 7, norms)
	if len(got) != 0 {
		t.Errorf("expected no deficit, got %+v", got)
	}
}

func TestTargetWithCarryover_CapByUL(t *testing.T) {
	norms := []domain.MicroNorm{
		{NutrientID: "ca", NormValue: 1000},
	}
	carry := []domain.MicroDeficit{
		{NutrientID: "ca", DeficitPerDay: 50000}, // огромный недобор → должно зажаться UL
	}
	uls := map[string]float64{"ca": 2500}

	got := TargetWithCarryover(norms, carry, uls)
	if math.Abs(got["ca"]-2500) > eps {
		t.Errorf("ca target = %v, want 2500 (UL)", got["ca"])
	}
}

func TestTargetWithCarryover_NoUL(t *testing.T) {
	norms := []domain.MicroNorm{{NutrientID: "k", NormValue: 2500}}
	carry := []domain.MicroDeficit{{NutrientID: "k", DeficitPerDay: 700}} // +100/day
	got := TargetWithCarryover(norms, carry, map[string]float64{})
	if math.Abs(got["k"]-2600) > eps {
		t.Errorf("k target = %v, want 2600", got["k"])
	}
}
