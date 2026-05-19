package targets

import (
	"math"
	"testing"

	"nutrition-core/internal/domain"
)

const eps = 0.01

func ptr(v float64) *float64 { return &v }

var smokeNorm = domain.EnergyNorm{
	Sex: domain.SexMale, AgeGroup: domain.Age18_29, KfaGroup: domain.KfaII,
	KcalNorm: 2800, ProteinGNorm: 80, FatGNorm: 93, CarbGNorm: 411,
}

func TestCalculate_ClassicMaintain(t *testing.T) {
	profile := domain.UserProfile{
		Sex: domain.SexMale, Age: 25,
		Goal: domain.GoalMaintain, DietType: domain.DietClassic,
	}
	got, err := Calculate(profile, smokeNorm, DietShares{})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if got.Kcal != 2800 || got.ProteinG != 80 || got.FatG != 93 || got.CarbG != 411 {
		t.Errorf("classic+maintain: got %+v, want norm verbatim", got)
	}
}

func TestCalculate_ClassicDeficit(t *testing.T) {
	profile := domain.UserProfile{
		Goal: domain.GoalDeficit, DietType: domain.DietClassic,
	}
	got, _ := Calculate(profile, smokeNorm, DietShares{})
	// 2800 * 0.85 = 2380; 80*0.85=68; 93*0.85=79.05; 411*0.85=349.35
	want := domain.DailyTargets{Kcal: 2380, ProteinG: 68, FatG: 79.05, CarbG: 349.35}
	if !approxEqualTargets(got, want) {
		t.Errorf("classic+deficit: got %+v, want %+v", got, want)
	}
}

func TestCalculate_KetoSurplus(t *testing.T) {
	profile := domain.UserProfile{
		Goal: domain.GoalSurplus, DietType: domain.DietKeto,
	}
	shares := DietShares{Protein: ptr(0.25), Fat: ptr(0.70), Carb: ptr(0.05)}
	got, err := Calculate(profile, smokeNorm, shares)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	// E* = 2800 * 1.15 = 3220
	// P  = 3220 * 0.25 / 4 = 201.25
	// F  = 3220 * 0.70 / 9 = 250.444
	// C  = 3220 * 0.05 / 4 = 40.25
	want := domain.DailyTargets{Kcal: 3220, ProteinG: 201.25, FatG: 250.444, CarbG: 40.25}
	if !approxEqualTargets(got, want) {
		t.Errorf("keto+surplus: got %+v, want %+v", got, want)
	}
}

func TestCalculate_ManualOverride_Clamped(t *testing.T) {
	profile := domain.UserProfile{
		Goal: domain.GoalMaintain, DietType: domain.DietClassic,
		ManualOverride: &domain.ManualTargets{
			Kcal:     ptr(5000),  // > +15% от 2800 = 3220 → должно зажаться на 3220
			ProteinG: ptr(0),     // < -15% от 80 = 68     → должно зажаться на 68
		},
	}
	got, _ := Calculate(profile, smokeNorm, DietShares{})
	if math.Abs(got.Kcal-3220) > eps {
		t.Errorf("kcal clamp top: got %v, want 3220", got.Kcal)
	}
	if math.Abs(got.ProteinG-68) > eps {
		t.Errorf("protein clamp bottom: got %v, want 68", got.ProteinG)
	}
	// fat и carb остались как от classic+maintain
	if got.FatG != 93 || got.CarbG != 411 {
		t.Errorf("unchanged BJU: F=%v C=%v", got.FatG, got.CarbG)
	}
}

func TestDistribute_FourMeals_NoPinned(t *testing.T) {
	daily := domain.DailyTargets{Kcal: 2800, ProteinG: 80, FatG: 93, CarbG: 411}
	active := []domain.MealType{
		domain.MealBreakfast, domain.MealLunch, domain.MealDinner, domain.MealSnack,
	}
	out, err := DistributeToMeals(daily, active, nil)
	if err != nil {
		t.Fatalf("Distribute: %v", err)
	}
	// При всех 4 активных суммы долей = 1.0, w = share.
	if math.Abs(out[domain.MealBreakfast].Kcal-2800*0.25) > eps {
		t.Errorf("breakfast kcal = %v", out[domain.MealBreakfast].Kcal)
	}
	if math.Abs(out[domain.MealLunch].Kcal-2800*0.35) > eps {
		t.Errorf("lunch kcal = %v", out[domain.MealLunch].Kcal)
	}

	// Проверим что сумма по приёмам = дневной норме.
	var sumKcal float64
	for _, mt := range out {
		sumKcal += mt.Kcal
	}
	if math.Abs(sumKcal-2800) > eps {
		t.Errorf("sum kcal = %v, want 2800", sumKcal)
	}
}

func TestDistribute_ThreeMeals_Renormalized(t *testing.T) {
	daily := domain.DailyTargets{Kcal: 2000}
	active := []domain.MealType{
		domain.MealBreakfast, domain.MealLunch, domain.MealDinner, // без snack
	}
	out, _ := DistributeToMeals(daily, active, nil)
	// share = 0.25 + 0.35 + 0.30 = 0.90
	// breakfast = 2000 * 0.25/0.90 = 555.56
	want := 2000 * 0.25 / 0.90
	if math.Abs(out[domain.MealBreakfast].Kcal-want) > eps {
		t.Errorf("breakfast renorm = %v, want %v", out[domain.MealBreakfast].Kcal, want)
	}
}

func TestDistribute_WithPinned(t *testing.T) {
	daily := domain.DailyTargets{Kcal: 2800, ProteinG: 80, FatG: 93, CarbG: 411}
	active := []domain.MealType{
		domain.MealBreakfast, domain.MealLunch, domain.MealDinner,
	}
	// Закреплённый завтрак на 500 ккал; остаётся 2300 на lunch+dinner.
	pinned := []PinnedContribution{
		{Meal: domain.MealBreakfast, Kcal: 500, ProteinG: 20, FatG: 15, CarbG: 60},
	}
	out, err := DistributeToMeals(daily, active, pinned)
	if err != nil {
		t.Fatalf("Distribute: %v", err)
	}
	if _, ok := out[domain.MealBreakfast]; ok {
		t.Error("pinned meal must not appear in output")
	}
	// share лунч+ужин = 0.35 + 0.30 = 0.65
	// lunch = 2300 * 0.35/0.65 = 1238.46
	want := 2300.0 * 0.35 / 0.65
	if math.Abs(out[domain.MealLunch].Kcal-want) > eps {
		t.Errorf("lunch with pinned = %v, want %v", out[domain.MealLunch].Kcal, want)
	}
}

func TestDistribute_PinnedExceedsDaily(t *testing.T) {
	daily := domain.DailyTargets{Kcal: 2000, ProteinG: 60, FatG: 70, CarbG: 250}
	active := []domain.MealType{domain.MealBreakfast, domain.MealLunch}
	pinned := []PinnedContribution{
		{Meal: domain.MealBreakfast, Kcal: 2500},
	}
	_, err := DistributeToMeals(daily, active, pinned)
	if err == nil {
		t.Error("expected error when pinned exceeds daily")
	}
}

func TestDistribute_UnknownMeal(t *testing.T) {
	daily := domain.DailyTargets{Kcal: 2000}
	_, err := DistributeToMeals(daily, []domain.MealType{"brunch"}, nil)
	if err == nil {
		t.Error("expected error on unknown meal")
	}
}

func approxEqualTargets(a, b domain.DailyTargets) bool {
	return math.Abs(a.Kcal-b.Kcal) < eps &&
		math.Abs(a.ProteinG-b.ProteinG) < eps &&
		math.Abs(a.FatG-b.FatG) < eps &&
		math.Abs(a.CarbG-b.CarbG) < eps
}
