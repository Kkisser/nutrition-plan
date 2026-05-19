package planner

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"nutrition-core/internal/config"
	"nutrition-core/internal/domain"
)

// Бенчмарк двухфазного планировщика (greedy + balance).
// Запуск:
//   go test ./internal/planner -bench=. -benchmem -run=^$
//
// Сценарии моделируют рост каталога: 20 / 80 / 200 рецептов равномерно
// распределённых по 4 приёмам. Размер 80 ≈ текущий smoke-каталог.

func makeBenchCatalog(perMeal int) map[domain.MealType][]domain.Recipe {
	meals := []domain.MealType{
		domain.MealBreakfast, domain.MealLunch, domain.MealDinner, domain.MealSnack,
	}
	cat := make(map[domain.MealType][]domain.Recipe, len(meals))
	for _, m := range meals {
		cat[m] = make([]domain.Recipe, perMeal)
		for i := 0; i < perMeal; i++ {
			// Раскидываем КБЖУ по разумной вилке, чтобы greedy не имел тривиального
			// выбора и balance делал реальную работу.
			kcal := 300.0 + float64(i%5)*120
			p := 12.0 + float64(i%4)*5
			f := 8.0 + float64(i%3)*4
			c := 30.0 + float64(i%6)*10
			cat[m][i] = domain.Recipe{
				ID:             uuid.New(),
				Name:           fmt.Sprintf("R-%s-%d", m, i),
				MealType:       m,
				Kcal:           kcal,
				ProteinG:       p,
				FatG:           f,
				CarbG:          c,
				Micronutrients: map[string]float64{},
			}
		}
	}
	return cat
}

func benchPlan(b *testing.B, perMeal int) {
	cat := makeBenchCatalog(perMeal)
	input := Input{
		Daily: domain.DailyTargets{
			Kcal: 2500, ProteinG: 100, FatG: 80, CarbG: 320,
		},
		ActiveMeals: []domain.MealType{
			domain.MealBreakfast, domain.MealLunch, domain.MealDinner, domain.MealSnack,
		},
		Catalog:   cat,
		TotalDays: 7,
	}
	p := New(config.DefaultPenalty())

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := p.PlanAndBalance(input)
		if err != nil {
			b.Fatalf("PlanAndBalance: %v", err)
		}
	}
}

func BenchmarkPlanAndBalance_Small(b *testing.B)  { benchPlan(b, 5) }
func BenchmarkPlanAndBalance_Medium(b *testing.B) { benchPlan(b, 20) }
func BenchmarkPlanAndBalance_Large(b *testing.B)  { benchPlan(b, 50) }
