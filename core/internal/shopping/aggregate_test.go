package shopping

import (
	"testing"

	"nutrition-core/internal/domain"
)

func mkSlot(alpha float64, ings ...domain.Ingredient) domain.PlanSlot {
	return domain.PlanSlot{
		Portions: alpha,
		Recipe:   domain.Recipe{Ingredients: ings},
	}
}

func TestAggregate_SumsByName(t *testing.T) {
	plan := []domain.PlanSlot{
		mkSlot(1.0,
			domain.Ingredient{ProductName: "Молоко 3.2%", Amount: 200, Unit: domain.UnitMl},
			domain.Ingredient{ProductName: "Овсянка", Amount: 60, Unit: domain.UnitG},
		),
		mkSlot(2.0,
			domain.Ingredient{ProductName: "Молоко 3.2%", Amount: 100, Unit: domain.UnitMl}, // 200 после α
		),
	}
	out := Aggregate(plan, 1)
	if len(out) != 2 {
		t.Fatalf("got %d items, want 2", len(out))
	}
	var milk *domain.ShoppingItem
	for i := range out {
		if out[i].IngredientName == "Молоко 3.2%" {
			milk = &out[i]
		}
	}
	if milk == nil || milk.Amount != 400 {
		t.Errorf("milk amount = %v, want 400 (200 + 100*2)", milk)
	}
}

func TestAggregate_MultipliesByPersons(t *testing.T) {
	plan := []domain.PlanSlot{
		mkSlot(1.0,
			domain.Ingredient{ProductName: "Гречка", Amount: 80, Unit: domain.UnitG},
		),
	}
	out := Aggregate(plan, 3)
	if len(out) != 1 || out[0].Amount != 240 {
		t.Errorf("got %+v, want 240 g (80 × 3 persons)", out)
	}
}

func TestAggregate_DifferentUnitsKeptSeparate(t *testing.T) {
	plan := []domain.PlanSlot{
		mkSlot(1.0,
			domain.Ingredient{ProductName: "Масло", Amount: 10, Unit: domain.UnitG},
			domain.Ingredient{ProductName: "Масло", Amount: 5, Unit: domain.UnitMl},
		),
	}
	out := Aggregate(plan, 1)
	if len(out) != 2 {
		t.Errorf("expected 2 separate items, got %+v", out)
	}
}

func TestAggregate_CaseInsensitiveNameKey(t *testing.T) {
	plan := []domain.PlanSlot{
		mkSlot(1.0,
			domain.Ingredient{ProductName: "Сахар", Amount: 10, Unit: domain.UnitG},
			domain.Ingredient{ProductName: " сахар ", Amount: 5, Unit: domain.UnitG},
		),
	}
	out := Aggregate(plan, 1)
	if len(out) != 1 || out[0].Amount != 15 {
		t.Errorf("expected merged sugar = 15, got %+v", out)
	}
}

func TestAggregate_SortedAlphabetically(t *testing.T) {
	plan := []domain.PlanSlot{
		mkSlot(1.0,
			domain.Ingredient{ProductName: "Яблоко", Amount: 100, Unit: domain.UnitG},
			domain.Ingredient{ProductName: "Авокадо", Amount: 50, Unit: domain.UnitG},
		),
	}
	out := Aggregate(plan, 1)
	if out[0].IngredientName != "Авокадо" || out[1].IngredientName != "Яблоко" {
		t.Errorf("sorting broken: %+v", out)
	}
}
