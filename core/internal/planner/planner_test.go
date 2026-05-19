package planner

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"nutrition-core/internal/config"
	"nutrition-core/internal/domain"
)

func makeRecipe(name string, meal domain.MealType, kcal, p, f, c float64) domain.Recipe {
	return domain.Recipe{
		ID: uuid.New(), Name: name, MealType: meal,
		Kcal: kcal, ProteinG: p, FatG: f, CarbG: c,
		Micronutrients: map[string]float64{},
	}
}

func TestPlanner_GreedyBasic(t *testing.T) {
	cat := map[domain.MealType][]domain.Recipe{
		domain.MealBreakfast: {
			makeRecipe("Oats", domain.MealBreakfast, 350, 12, 6, 60),
			makeRecipe("Cottage", domain.MealBreakfast, 200, 18, 9, 4),
		},
		domain.MealLunch: {
			makeRecipe("Chicken", domain.MealLunch, 500, 35, 10, 50),
			makeRecipe("Salad", domain.MealLunch, 300, 5, 15, 30),
		},
		domain.MealDinner: {
			makeRecipe("Salmon", domain.MealDinner, 450, 30, 20, 30),
			makeRecipe("Lentils", domain.MealDinner, 400, 22, 5, 60),
		},
	}
	p := New(config.DefaultPenalty())
	plan, err := p.Plan(Input{
		Daily:       domain.DailyTargets{Kcal: 2000, ProteinG: 100, FatG: 70, CarbG: 250},
		ActiveMeals: []domain.MealType{domain.MealBreakfast, domain.MealLunch, domain.MealDinner},
		Catalog:     cat,
		TotalDays:   3,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan) != 3*3 {
		t.Errorf("got %d slots, want 9", len(plan))
	}
	for _, s := range plan {
		if s.Portions < AlphaMin-1e-9 || s.Portions > AlphaMax+1e-9 {
			t.Errorf("alpha %v out of [%v, %v]", s.Portions, AlphaMin, AlphaMax)
		}
		if s.Kcal <= 0 {
			t.Errorf("slot kcal must be positive: %+v", s)
		}
		if s.Pinned {
			t.Error("no pins were given, but slot is marked pinned")
		}
	}
}

func TestPlanner_NoCandidates(t *testing.T) {
	p := New(config.DefaultPenalty())
	_, err := p.Plan(Input{
		Daily:       domain.DailyTargets{Kcal: 2000, ProteinG: 80, FatG: 70, CarbG: 250},
		ActiveMeals: []domain.MealType{domain.MealBreakfast},
		Catalog:     map[domain.MealType][]domain.Recipe{}, // пусто
		TotalDays:   1,
	})
	if err == nil {
		t.Error("expected error when catalog has no candidates")
	}
}

func TestPlanner_WithPinned(t *testing.T) {
	pinned := makeRecipe("FixedBreakfast", domain.MealBreakfast, 400, 20, 10, 50)
	cat := map[domain.MealType][]domain.Recipe{
		domain.MealLunch: {
			makeRecipe("Chicken", domain.MealLunch, 500, 35, 10, 50),
		},
	}
	p := New(config.DefaultPenalty())
	plan, err := p.Plan(Input{
		Daily:       domain.DailyTargets{Kcal: 2000, ProteinG: 100, FatG: 70, CarbG: 250},
		ActiveMeals: []domain.MealType{domain.MealBreakfast, domain.MealLunch},
		Catalog:     cat,
		Pinned: []domain.PinnedDish{
			{Day: 1, Meal: domain.MealBreakfast, RecipeID: pinned.ID},
		},
		PinnedLookup: map[uuid.UUID]domain.Recipe{pinned.ID: pinned},
		TotalDays:    1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("got %d slots, want 2", len(plan))
	}
	var bf *domain.PlanSlot
	for i, s := range plan {
		if s.Meal == domain.MealBreakfast {
			bf = &plan[i]
		}
	}
	if bf == nil || !bf.Pinned {
		t.Error("breakfast must be pinned slot")
	}
	if bf.Recipe.Name != "FixedBreakfast" {
		t.Errorf("pinned name = %q", bf.Recipe.Name)
	}
}

func TestPlanner_RepetitionPenaltyDiversifies(t *testing.T) {
	// Два почти равных по КБЖУ блюда. Без штрафа за повтор алгоритм
	// мог бы выбрать одно и то же 7 дней подряд. С w3>0 и k=3 ожидаем
	// чередование.
	cat := map[domain.MealType][]domain.Recipe{
		domain.MealLunch: {
			makeRecipe("A", domain.MealLunch, 500, 30, 20, 40),
			makeRecipe("B", domain.MealLunch, 510, 31, 20, 40),
		},
	}
	p := New(config.DefaultPenalty())
	plan, err := p.Plan(Input{
		Daily:       domain.DailyTargets{Kcal: 500, ProteinG: 30, FatG: 20, CarbG: 40},
		ActiveMeals: []domain.MealType{domain.MealLunch},
		Catalog:     cat,
		TotalDays:   7,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	uniq := map[string]int{}
	for _, s := range plan {
		uniq[s.Recipe.Name]++
	}
	if len(uniq) < 2 {
		t.Errorf("expected diversification, got only %v", uniq)
	}
}

func TestBalanceDays_NoOpInsideCorridor(t *testing.T) {
	cfg := config.DefaultPenalty()
	cfg.CorridorRel = 0.5 // широкий коридор → ничего не меняем
	cat := map[domain.MealType][]domain.Recipe{
		domain.MealLunch: {makeRecipe("A", domain.MealLunch, 1000, 50, 30, 100)},
	}
	p := New(cfg)
	in := Input{
		Daily:       domain.DailyTargets{Kcal: 1000, ProteinG: 50, FatG: 30, CarbG: 100},
		ActiveMeals: []domain.MealType{domain.MealLunch},
		Catalog:     cat,
		TotalDays:   1,
	}
	plan, _ := p.Plan(in)
	mt := map[int]map[domain.MealType]domain.MealTargets{
		1: {domain.MealLunch: {Meal: domain.MealLunch, Kcal: 1000, ProteinG: 50, FatG: 30, CarbG: 100}},
	}
	balanced := p.BalanceDays(plan, in.Daily, cat, mt)
	if len(balanced) != len(plan) {
		t.Errorf("balance changed count: %d → %d", len(plan), len(balanced))
	}
	if math.Abs(balanced[0].Kcal-plan[0].Kcal) > 1e-9 {
		t.Errorf("balance changed slot inside corridor")
	}
}
