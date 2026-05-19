// Package micronutrients — расчёт среднесуточного потребления и переноса
// недобора. Источник: docs/МАТМОДЕЛЬ.txt §7.
package micronutrients

import (
	"math"

	"nutrition-core/internal/domain"
)

// PlannedDaily считает среднесуточное запланированное потребление
// каждого микронутриента по плану на totalDays дней.
//
// Микронутриент берётся из slot.Recipe.Micronutrients (содержание на
// 1 базовую порцию) и масштабируется на slot.Portions (α).
// «Фактическим» принимается запланированное (МАТМОДЕЛЬ §7.2).
func PlannedDaily(plan []domain.PlanSlot, totalDays int) map[string]float64 {
	if totalDays <= 0 {
		totalDays = 7
	}
	weekly := make(map[string]float64)
	for _, s := range plan {
		for nid, amt := range s.Recipe.Micronutrients {
			weekly[nid] += amt * s.Portions
		}
	}
	daily := make(map[string]float64, len(weekly))
	for nid, w := range weekly {
		daily[nid] = w / float64(totalDays)
	}
	return daily
}

// Deficit считает недельный недобор каждого микронутриента и возвращает
// его в пересчёте на сутки:
//
//	Δ_i = max(0, N_norm_i − N_plan_i)    (МАТМОДЕЛЬ §7.2)
//
// Возвращаются только те нутриенты, где Δ > 0. Перечисляются в порядке
// возрастания nutrient_id для детерминизма.
func Deficit(plan []domain.PlanSlot, totalDays int, norms []domain.MicroNorm) []domain.MicroDeficit {
	planned := PlannedDaily(plan, totalDays)

	out := make([]domain.MicroDeficit, 0, len(norms))
	for _, n := range norms {
		delta := n.NormValue - planned[n.NutrientID]
		if delta > 0 {
			out = append(out, domain.MicroDeficit{
				NutrientID:    n.NutrientID,
				DeficitPerDay: delta,
			})
		}
	}
	sortDeficitsByID(out)
	return out
}

// TargetWithCarryover применяет недобор предыдущей недели к норме
// текущей: N_target_i = min(N_norm_i + Δ_prev_i / 7, UL_i)  (§7.2).
//
// nutrients нужен для UL: возвращает map nutrient_id → UL (0 = не задан).
//
// Возвращает map nutrient_id → effective target.
func TargetWithCarryover(
	norms []domain.MicroNorm,
	carryover []domain.MicroDeficit,
	uls map[string]float64,
) map[string]float64 {
	carryByID := make(map[string]float64, len(carryover))
	for _, c := range carryover {
		carryByID[c.NutrientID] = c.DeficitPerDay
	}
	out := make(map[string]float64, len(norms))
	for _, n := range norms {
		t := n.NormValue + carryByID[n.NutrientID]/7.0
		if ul := uls[n.NutrientID]; ul > 0 && t > ul {
			t = ul
		}
		out[n.NutrientID] = t
	}
	return out
}

func sortDeficitsByID(d []domain.MicroDeficit) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j-1].NutrientID > d[j].NutrientID; j-- {
			d[j-1], d[j] = d[j], d[j-1]
		}
	}
}

// round2 — округление до 2 знаков для удобства логирования.
func round2(x float64) float64 { return math.Round(x*100) / 100 }

var _ = round2
