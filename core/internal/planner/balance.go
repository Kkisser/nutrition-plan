package planner

import (
	"math"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// BalanceDays — фаза 2 алгоритма (МАТМОДЕЛЬ.txt §4.3).
// Для каждого дня смотрит суммарные КБЖУ и при выходе из коридора
// пытается заменить блюдо с наибольшим вкладом в отклонение на
// блюдо из того же приёма пищи (из catalog[meal]), уменьшающее
// суммарное дневное отклонение.
//
// Ограничения:
//   - закреплённые блюда (Pinned=true) не заменяются;
//   - F(new) ≤ c · F(old), c = cfg.ReplaceMaxRise (МАТМОДЕЛЬ §4.5);
//   - максимум maxIter итераций на день — защита от бесконечного цикла;
//   - изменение состояния lastSeen на этой фазе НЕ ведём (повторяемость
//     остаётся консервативно как после фазы 1).
//
// Возвращает обновлённый список слотов.
func (p *Planner) BalanceDays(
	plan []domain.PlanSlot,
	daily domain.DailyTargets,
	catalog map[domain.MealType][]domain.Recipe,
	mealTargetsByDay map[int]map[domain.MealType]domain.MealTargets,
) []domain.PlanSlot {
	const maxIter = 20

	byDay := groupByDay(plan)
	maxDay := 0
	for d := range byDay {
		if d > maxDay {
			maxDay = d
		}
	}

	// Дни обходятся по возрастанию, чтобы lastSeen строился консистентно
	// (МАТМОДЕЛЬ §4.3 — повторяемость учитывается и в фазе 2).
	for day := 1; day <= maxDay; day++ {
		slots := byDay[day]
		if len(slots) == 0 {
			continue
		}
		// Что уже было в плане до текущего дня (для штрафа за повтор).
		lastSeen := buildLastSeenBefore(byDay, day)
		mt := mealTargetsByDay[day]
		for iter := 0; iter < maxIter; iter++ {
			devBefore := dayDeviation(slots, daily)
			if devBefore <= p.cfg.CorridorRel {
				break
			}
			improved := p.tryReplaceWorstSlot(slots, mt, catalog, daily, lastSeen, day)
			if !improved {
				break
			}
		}
		byDay[day] = slots
	}
	return flattenByDay(byDay, len(plan))
}

// buildLastSeenBefore возвращает map recipeID → последний день включения
// в план среди дней < currentDay.
func buildLastSeenBefore(
	byDay map[int][]domain.PlanSlot, currentDay int,
) map[uuid.UUID]int {
	out := make(map[uuid.UUID]int)
	for d := 1; d < currentDay; d++ {
		for _, s := range byDay[d] {
			if cur, ok := out[s.Recipe.ID]; !ok || cur < d {
				out[s.Recipe.ID] = d
			}
		}
	}
	return out
}

// dayDeviation — суммарное относительное отклонение от дневной нормы.
// Вид: (|ΣE − E*|/E* + |ΣP − P*|/P* + |ΣF − F*|/F* + |ΣC − C*|/C*) / 4.
// 0 — идеальное попадание; > cfg.CorridorRel — выход из коридора.
func dayDeviation(slots []domain.PlanSlot, daily domain.DailyTargets) float64 {
	var e, p, f, c float64
	for _, s := range slots {
		e += s.Kcal
		p += s.ProteinG
		f += s.FatG
		c += s.CarbG
	}
	return (relAbs(e, daily.Kcal) +
		relAbs(p, daily.ProteinG) +
		relAbs(f, daily.FatG) +
		relAbs(c, daily.CarbG)) / 4.0
}

func relAbs(x, target float64) float64 {
	if target <= 0 {
		return 0
	}
	return math.Abs(x-target) / target
}

func groupByDay(plan []domain.PlanSlot) map[int][]domain.PlanSlot {
	out := make(map[int][]domain.PlanSlot)
	for _, s := range plan {
		out[s.Day] = append(out[s.Day], s)
	}
	return out
}

func flattenByDay(byDay map[int][]domain.PlanSlot, total int) []domain.PlanSlot {
	out := make([]domain.PlanSlot, 0, total)
	maxDay := 0
	for d := range byDay {
		if d > maxDay {
			maxDay = d
		}
	}
	for d := 1; d <= maxDay; d++ {
		out = append(out, byDay[d]...)
	}
	return out
}

// tryReplaceWorstSlot выбирает свободный слот с наибольшим вкладом в
// дневное отклонение и пытается его заменить.
//
// Возвращает true, если замена принята (slots мутируется на месте).
func (p *Planner) tryReplaceWorstSlot(
	slots []domain.PlanSlot,
	mt map[domain.MealType]domain.MealTargets,
	catalog map[domain.MealType][]domain.Recipe,
	daily domain.DailyTargets,
	lastSeen map[uuid.UUID]int,
	day int,
) bool {
	worstIdx := -1
	var worstContrib float64
	for i, s := range slots {
		if s.Pinned {
			continue
		}
		contrib := slotDayContribution(s, daily)
		if contrib > worstContrib {
			worstContrib = contrib
			worstIdx = i
		}
	}
	if worstIdx < 0 {
		return false
	}
	current := slots[worstIdx]
	cands := catalog[current.Meal]
	target := mt[current.Meal]

	devBefore := dayDeviation(slots, daily)
	oldF := Penalty(current.Recipe, current.Portions, target, lastSeen[current.Recipe.ID], day, p.cfg)

	var bestSlot *domain.PlanSlot
	var bestDev = devBefore
	for _, r := range cands {
		if r.ID == current.Recipe.ID {
			continue
		}
		// Жёсткий запрет на замену в последние K дней — иначе balance
		// «съедает» разнообразие, выбираемое фазой 1 (МАТМОДЕЛЬ §3.2:
		// штраф за повтор интегрирован в общий критерий).
		if seen := lastSeen[r.ID]; seen > 0 && (day-seen) <= p.cfg.K {
			continue
		}
		alpha := OptimalAlpha(target.Kcal, r.Kcal)
		newF := Penalty(r, alpha, target, lastSeen[r.ID], day, p.cfg)
		// Trust-region: запрет на слишком "плохое" по приёму пищи блюдо
		// (МАТМОДЕЛЬ §4.5).
		if oldF > 0 && newF > p.cfg.ReplaceMaxRise*oldF {
			continue
		}
		scaled := Scale(r, alpha)
		candidate := domain.PlanSlot{
			Day: current.Day, Meal: current.Meal, Recipe: r,
			Portions: alpha, Pinned: false,
			Kcal: scaled.Kcal, ProteinG: scaled.ProteinG,
			FatG: scaled.FatG, CarbG: scaled.CarbG,
		}
		// Считаем девиацию с подставленным кандидатом.
		slots[worstIdx] = candidate
		newDev := dayDeviation(slots, daily)
		slots[worstIdx] = current // откат, выберем лучшего

		if newDev < bestDev {
			bestDev = newDev
			c := candidate
			bestSlot = &c
		}
	}
	if bestSlot == nil {
		return false
	}
	slots[worstIdx] = *bestSlot
	return true
}

// slotDayContribution — относительное «насколько слот тянет день в сторону»:
// средневзвешенный вклад в суммарное отклонение четырёх метрик.
// Эвристика для выбора кандидата на замену (МАТМОДЕЛЬ §4.3: "от блюда с
// наибольшим вкладом в отклонение").
func slotDayContribution(s domain.PlanSlot, daily domain.DailyTargets) float64 {
	return (relShare(s.Kcal, daily.Kcal) +
		relShare(s.ProteinG, daily.ProteinG) +
		relShare(s.FatG, daily.FatG) +
		relShare(s.CarbG, daily.CarbG)) / 4.0
}

func relShare(x, target float64) float64 {
	if target <= 0 {
		return 0
	}
	return x / target
}

// dishKeyOf — псевдоним для будущей телеметрии.
var _ = func(id uuid.UUID) uuid.UUID { return id }
