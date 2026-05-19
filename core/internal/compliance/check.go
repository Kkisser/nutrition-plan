// Package compliance проверяет, укладывается ли план в коридор по дневным
// КБЖУ и в нормы микронутриентов за неделю.
// Источник: docs/МАТМОДЕЛЬ.txt §4.4, docs/КОНТРАКТ_API.md §2.2.
package compliance

import (
	"fmt"
	"math"

	"nutrition-core/internal/domain"
	"nutrition-core/internal/micronutrients"
)

// Check возвращает результат проверки коридора.
//
//	in_corridor = true ⇔ ни одного нарушения.
//
// Нарушения формируются по двум осям:
//  1. Дневной: |Σ_слотов(день) − daily_target| / daily_target ≤ corridorRel
//     по каждой из четырёх метрик (kcal, protein, fat, carb).
//  2. Недельный по микронутриентам: среднесуточное запланированное
//     потребление ≥ нормы пользователя.
func Check(
	plan []domain.PlanSlot,
	daily domain.DailyTargets,
	totalDays int,
	corridorRel float64,
	microNorms []domain.MicroNorm,
) domain.Compliance {
	var violations []domain.Violation

	byDay := groupByDay(plan)
	for day, slots := range byDay {
		sum := sumNutrition(slots)
		violations = append(violations, dailyViolations(day, sum, daily, corridorRel)...)
	}

	deficits := micronutrients.Deficit(plan, totalDays, microNorms)
	for _, d := range deficits {
		violations = append(violations, domain.Violation{
			Day:    0, // 0 = недельный показатель
			Metric: d.NutrientID,
			Value:  0,
			Target: d.DeficitPerDay,
			Comment: fmt.Sprintf(
				"micronutrient %s daily deficit %.2f — will carry over",
				d.NutrientID, d.DeficitPerDay),
		})
	}

	c := domain.Compliance{
		InCorridor: len(filterDailyOnly(violations)) == 0,
		Violations: violations,
	}
	if !c.InCorridor {
		c.Message = "План не укладывается в дневной коридор по одному или нескольким показателям."
	}
	return c
}

func groupByDay(plan []domain.PlanSlot) map[int][]domain.PlanSlot {
	out := make(map[int][]domain.PlanSlot)
	for _, s := range plan {
		out[s.Day] = append(out[s.Day], s)
	}
	return out
}

type sums struct{ Kcal, P, F, C float64 }

func sumNutrition(slots []domain.PlanSlot) sums {
	var s sums
	for _, sl := range slots {
		s.Kcal += sl.Kcal
		s.P += sl.ProteinG
		s.F += sl.FatG
		s.C += sl.CarbG
	}
	return s
}

func dailyViolations(day int, s sums, daily domain.DailyTargets, rel float64) []domain.Violation {
	var v []domain.Violation
	check := func(metric string, value, target float64) {
		if target <= 0 {
			return
		}
		dev := math.Abs(value-target) / target
		if dev > rel {
			v = append(v, domain.Violation{
				Day:    day,
				Metric: metric,
				Value:  value,
				Target: target,
				Comment: fmt.Sprintf("day %d %s deviation %.1f%% > corridor %.1f%%",
					day, metric, dev*100, rel*100),
			})
		}
	}
	check("kcal", s.Kcal, daily.Kcal)
	check("protein_g", s.P, daily.ProteinG)
	check("fat_g", s.F, daily.FatG)
	check("carb_g", s.C, daily.CarbG)
	return v
}

// Дневные нарушения определяют флаг in_corridor; недельные микронутриенты
// фиксируются как «недобор для переноса», но НЕ выводят план из коридора
// (docs/МАТМОДЕЛЬ.txt §7: микронутриенты не входят в основной критерий).
func filterDailyOnly(v []domain.Violation) []domain.Violation {
	out := make([]domain.Violation, 0, len(v))
	for _, x := range v {
		if x.Day > 0 {
			out = append(out, x)
		}
	}
	return out
}
