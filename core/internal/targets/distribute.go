package targets

import (
	"fmt"

	"nutrition-core/internal/domain"
)

// PinnedContribution — суммарный вклад уже закреплённых блюд в один
// конкретный приём пищи (docs/МАТМОДЕЛЬ.txt §2).
type PinnedContribution struct {
	Meal     domain.MealType
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbG    float64
}

// DistributeToMeals разносит дневные целевые КБЖУ по свободным приёмам
// пищи (МАТМОДЕЛЬ.txt §2).
//
//	share = (0.25, 0.35, 0.30, 0.10) для (breakfast, lunch, dinner, snack)
//	share' = share / Σ share (ренормализация на симплекс активных приёмов).
//
// Если есть закреплённые блюда, их фактический вклад вычитается из
// дневных целевых, а остаток распределяется по свободным приёмам.
//
// Возвращает map[MealType]MealTargets только по СВОБОДНЫМ приёмам.
// Ошибка возвращается, если:
//   - остаток по какому-либо показателю стал отрицательным
//     (план не удовлетворяет коридору — caller инициирует процедуру §4.4);
//   - свободных приёмов нет, а остаток ≠ 0;
//   - active содержит дубликаты или неизвестный MealType.
func DistributeToMeals(
	daily domain.DailyTargets,
	active []domain.MealType,
	pinned []PinnedContribution,
) (map[domain.MealType]domain.MealTargets, error) {
	if len(active) == 0 {
		return nil, fmt.Errorf("no active meals")
	}
	seen := make(map[domain.MealType]bool, len(active))
	for _, m := range active {
		if seen[m] {
			return nil, fmt.Errorf("duplicate meal %q in active list", m)
		}
		if _, ok := domain.MealShare[m]; !ok {
			return nil, fmt.Errorf("unknown meal %q", m)
		}
		seen[m] = true
	}

	pinnedMeals := make(map[domain.MealType]PinnedContribution)
	for _, p := range pinned {
		if !seen[p.Meal] {
			return nil, fmt.Errorf("pinned meal %q is not active", p.Meal)
		}
		// При двух pinned-блюдах в одном meal — суммируем (хотя в схеме
		// по UNIQUE (plan,day,meal) этого не должно случиться).
		acc := pinnedMeals[p.Meal]
		acc.Meal = p.Meal
		acc.Kcal += p.Kcal
		acc.ProteinG += p.ProteinG
		acc.FatG += p.FatG
		acc.CarbG += p.CarbG
		pinnedMeals[p.Meal] = acc
	}

	// Остаток после вычитания pinned.
	rem := daily
	for _, p := range pinnedMeals {
		rem.Kcal -= p.Kcal
		rem.ProteinG -= p.ProteinG
		rem.FatG -= p.FatG
		rem.CarbG -= p.CarbG
	}
	if rem.Kcal < 0 || rem.ProteinG < 0 || rem.FatG < 0 || rem.CarbG < 0 {
		return nil, fmt.Errorf(
			"pinned dishes exceed daily target (remainder: kcal=%.1f, p=%.1f, f=%.1f, c=%.1f)",
			rem.Kcal, rem.ProteinG, rem.FatG, rem.CarbG)
	}

	// Свободные приёмы и сумма их долей.
	freeShares := make(map[domain.MealType]float64)
	var sumShares float64
	for _, m := range active {
		if _, isPinned := pinnedMeals[m]; isPinned {
			continue
		}
		s := domain.MealShare[m]
		freeShares[m] = s
		sumShares += s
	}
	if len(freeShares) == 0 {
		// Все приёмы закреплены — проверим, что остаток нулевой.
		const eps = 0.5
		if rem.Kcal > eps || rem.ProteinG > eps || rem.FatG > eps || rem.CarbG > eps {
			return nil, fmt.Errorf("all meals pinned but remainder non-zero")
		}
		return map[domain.MealType]domain.MealTargets{}, nil
	}
	if sumShares == 0 {
		return nil, fmt.Errorf("share sum is zero")
	}

	out := make(map[domain.MealType]domain.MealTargets, len(freeShares))
	for m, s := range freeShares {
		w := s / sumShares
		out[m] = domain.MealTargets{
			Meal:     m,
			Kcal:     rem.Kcal * w,
			ProteinG: rem.ProteinG * w,
			FatG:     rem.FatG * w,
			CarbG:    rem.CarbG * w,
		}
	}
	return out, nil
}
