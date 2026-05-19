package api

import (
	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// buildResponse собирает PlanResponse из доменных результатов pipeline.
func buildResponse(
	userID uuid.UUID,
	weekRef string,
	slots []domain.PlanSlot,
	shopping []domain.ShoppingItem,
	comp domain.Compliance,
	deficits []domain.MicroDeficit,
) PlanResponse {
	resp := PlanResponse{
		UserID:  userID,
		WeekRef: weekRef,
	}

	byDay := make(map[int][]domain.PlanSlot)
	for _, s := range slots {
		byDay[s.Day] = append(byDay[s.Day], s)
	}

	maxDay := 0
	for d := range byDay {
		if d > maxDay {
			maxDay = d
		}
	}
	resp.Plan = make([]DayDTO, 0, maxDay)
	for d := 1; d <= maxDay; d++ {
		day := DayDTO{Day: d}
		var t DayTotalsDTO
		for _, s := range byDay[d] {
			day.Meals = append(day.Meals, MealSlotDTO{
				Meal:      string(s.Meal),
				DishID:    s.Recipe.ID,
				DishTitle: s.Recipe.Name,
				Portions:  s.Portions,
				Pinned:    s.Pinned,
				Kcal:      round1(s.Kcal),
				ProteinG:  round1(s.ProteinG),
				FatG:      round1(s.FatG),
				CarbG:     round1(s.CarbG),
			})
			t.Kcal += s.Kcal
			t.ProteinG += s.ProteinG
			t.FatG += s.FatG
			t.CarbG += s.CarbG
		}
		day.DayTotals = DayTotalsDTO{
			Kcal: round1(t.Kcal), ProteinG: round1(t.ProteinG),
			FatG: round1(t.FatG), CarbG: round1(t.CarbG),
		}
		resp.Plan = append(resp.Plan, day)
	}

	resp.ShoppingList = make([]ShoppingItemDTO, 0, len(shopping))
	for _, it := range shopping {
		resp.ShoppingList = append(resp.ShoppingList, ShoppingItemDTO{
			IngredientName: it.IngredientName,
			Category:       it.Category,
			Amount:         it.Amount,
			Unit:           string(it.Unit),
		})
	}

	resp.Compliance = ComplianceDTO{
		InCorridor: comp.InCorridor,
		Message:    comp.Message,
		Violations: make([]ViolationDTO, 0, len(comp.Violations)),
	}
	for _, v := range comp.Violations {
		resp.Compliance.Violations = append(resp.Compliance.Violations, ViolationDTO{
			Day:     v.Day,
			Metric:  v.Metric,
			Value:   round1(v.Value),
			Target:  round1(v.Target),
			Comment: v.Comment,
		})
	}

	resp.MicronutrientCarryoverNext = MicroCarryoverDTO{
		WeekRef:  weekRef,
		Deficits: make([]MicroDeficitDTO, 0, len(deficits)),
	}
	for _, d := range deficits {
		resp.MicronutrientCarryoverNext.Deficits = append(
			resp.MicronutrientCarryoverNext.Deficits,
			MicroDeficitDTO{NutrientID: d.NutrientID, DeficitPerDay: round2(d.DeficitPerDay)},
		)
	}
	return resp
}

func round1(x float64) float64 { return float64(int64(x*10+0.5)) / 10 }
func round2(x float64) float64 { return float64(int64(x*100+0.5)) / 100 }
