// Package planner — двухфазный алгоритм формирования плана питания.
// Источник: docs/МАТМОДЕЛЬ.txt §§3–5, docs/АЛГОРИТМ_блок-схема.svg.
package planner

// Границы коэффициента порции α из МАТМОДЕЛЬ.txt §5.
const (
	AlphaMin = 0.7
	AlphaMax = 2.0
)

// OptimalAlpha — аналитический оптимум одномерной задачи
// min_α |α·E_d − E*_m|, спроецированный на [α_min, α_max]
// (box-constraint-projection из МАТМОДЕЛЬ §5).
//
// Если E_d == 0 — возвращает 1.0 (нейтральная порция; пограничный
// случай, не должен возникать на корректных рецептах).
func OptimalAlpha(targetKcal, dishKcal float64) float64 {
	if dishKcal <= 0 {
		return 1.0
	}
	a := targetKcal / dishKcal
	if a < AlphaMin {
		return AlphaMin
	}
	if a > AlphaMax {
		return AlphaMax
	}
	return a
}
