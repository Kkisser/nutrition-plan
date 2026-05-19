package planner

import (
	"fmt"

	"github.com/google/uuid"

	"nutrition-core/internal/config"
	"nutrition-core/internal/domain"
	"nutrition-core/internal/targets"
)

// Input — данные для формирования недельного плана.
type Input struct {
	Daily        domain.DailyTargets
	ActiveMeals  []domain.MealType
	Catalog      map[domain.MealType][]domain.Recipe // уже отфильтрованный
	Pinned       []domain.PinnedDish                 // (day, meal, recipe_id)
	PinnedLookup map[uuid.UUID]domain.Recipe         // быстро достать pinned-блюдо
	TotalDays    int                                 // обычно 7
}

// Planner — реализует двухфазный алгоритм.
type Planner struct {
	cfg config.Penalty
}

func New(cfg config.Penalty) *Planner {
	return &Planner{cfg: cfg}
}

// PlanAndBalance — последовательно запускает фазу 1 (жадный подбор) и
// фазу 2 (локальный поиск для балансировки дневных КБЖУ).
// Используется как штатная точка входа в API.
func (p *Planner) PlanAndBalance(in Input) ([]domain.PlanSlot, error) {
	slots, mealTargetsByDay, err := p.planWithTargets(in)
	if err != nil {
		return nil, err
	}
	return p.BalanceDays(slots, in.Daily, in.Catalog, mealTargetsByDay), nil
}

// Plan — только фаза 1 (жадный подбор). Удобно для тестов алгоритма.
func (p *Planner) Plan(in Input) ([]domain.PlanSlot, error) {
	slots, _, err := p.planWithTargets(in)
	return slots, err
}

// planWithTargets делает фазу 1 и попутно сохраняет mealTargets для каждого
// дня — это нужно фазе 2.
func (p *Planner) planWithTargets(in Input) (
	[]domain.PlanSlot,
	map[int]map[domain.MealType]domain.MealTargets,
	error,
) {
	if in.TotalDays <= 0 {
		in.TotalDays = 7
	}
	if len(in.ActiveMeals) == 0 {
		return nil, nil, fmt.Errorf("planner: no active meals")
	}

	mealTargetsByDay := make(map[int]map[domain.MealType]domain.MealTargets)

	// Группируем pinned по (day, meal) для быстрой выборки.
	pinnedByDayMeal := make(map[int]map[domain.MealType]uuid.UUID)
	for _, pd := range in.Pinned {
		if pinnedByDayMeal[pd.Day] == nil {
			pinnedByDayMeal[pd.Day] = make(map[domain.MealType]uuid.UUID)
		}
		pinnedByDayMeal[pd.Day][pd.Meal] = pd.RecipeID
	}

	lastSeen := make(map[uuid.UUID]int)
	var out []domain.PlanSlot

	for day := 1; day <= in.TotalDays; day++ {
		dayPinned := pinnedByDayMeal[day]

		// Соберём вклад pinned-блюд в этот день для DistributeToMeals.
		var pinnedContrib []targets.PinnedContribution
		for meal, recipeID := range dayPinned {
			r, ok := in.PinnedLookup[recipeID]
			if !ok {
				return nil, nil, fmt.Errorf(
					"planner: pinned recipe %s not found in lookup", recipeID)
			}
			// α = 1.0 для pinned (МАТМОДЕЛЬ §3.1: pinned не подбирается).
			pinnedContrib = append(pinnedContrib, targets.PinnedContribution{
				Meal:     meal,
				Kcal:     r.Kcal,
				ProteinG: r.ProteinG,
				FatG:     r.FatG,
				CarbG:    r.CarbG,
			})
			out = append(out, domain.PlanSlot{
				Day: day, Meal: meal, Recipe: r,
				Portions: 1.0, Pinned: true,
				Kcal: r.Kcal, ProteinG: r.ProteinG, FatG: r.FatG, CarbG: r.CarbG,
			})
			lastSeen[r.ID] = day
		}

		// Целевые на свободные приёмы пищи этого дня.
		mealTargets, err := targets.DistributeToMeals(in.Daily, in.ActiveMeals, pinnedContrib)
		if err != nil {
			return nil, nil, fmt.Errorf("day %d: distribute: %w", day, err)
		}
		mealTargetsByDay[day] = mealTargets

		// Жадный подбор для каждого свободного приёма пищи.
		for _, meal := range in.ActiveMeals {
			if _, isPinned := dayPinned[meal]; isPinned {
				continue
			}
			target, ok := mealTargets[meal]
			if !ok {
				continue
			}
			candidates := in.Catalog[meal]
			if len(candidates) == 0 {
				return nil, nil, fmt.Errorf(
					"в каталоге нет рецептов на приём «%s» под выбранную диету и фильтры (день %d). "+
						"Уменьшите список аллергенов / исключённых продуктов или измените диету в анкете",
					mealRu(meal), day)
			}
			pick, alpha := p.greedyPick(candidates, target, lastSeen, day)
			scaled := Scale(pick, alpha)
			out = append(out, domain.PlanSlot{
				Day: day, Meal: meal, Recipe: pick,
				Portions: alpha, Pinned: false,
				Kcal:     scaled.Kcal,
				ProteinG: scaled.ProteinG,
				FatG:     scaled.FatG,
				CarbG:    scaled.CarbG,
			})
			lastSeen[pick.ID] = day
		}
	}
	return out, mealTargetsByDay, nil
}

// mealRu — короткое русское имя приёма пищи для сообщений об ошибках UI.
func mealRu(m domain.MealType) string {
	switch m {
	case domain.MealBreakfast:
		return "завтрак"
	case domain.MealLunch:
		return "обед"
	case domain.MealDinner:
		return "ужин"
	case domain.MealSnack:
		return "перекус"
	}
	return string(m)
}

// greedyPick — выбирает блюдо с минимальным F.
// Параллельно подбирает оптимальный α для каждого кандидата (§5).
func (p *Planner) greedyPick(
	cands []domain.Recipe,
	target domain.MealTargets,
	lastSeen map[uuid.UUID]int,
	day int,
) (domain.Recipe, float64) {
	best := cands[0]
	bestAlpha := OptimalAlpha(target.Kcal, best.Kcal)
	bestF := Penalty(best, bestAlpha, target, lastSeen[best.ID], day, p.cfg)

	for _, r := range cands[1:] {
		alpha := OptimalAlpha(target.Kcal, r.Kcal)
		f := Penalty(r, alpha, target, lastSeen[r.ID], day, p.cfg)
		if f < bestF {
			best = r
			bestAlpha = alpha
			bestF = f
		}
	}
	return best, bestAlpha
}
