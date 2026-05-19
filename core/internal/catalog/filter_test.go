package catalog

import (
	"testing"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// fixtures: 4 «синтетических» блюда, покрывающих все ветки фильтра.
func fixtures() (recipes []domain.Recipe, oatsID, saladID uuid.UUID) {
	oatsID = uuid.New()
	saladID = uuid.New()

	recipes = []domain.Recipe{
		{
			ID: oatsID, Name: "Овсянка", MealType: domain.MealBreakfast,
			Diets:     []domain.DietType{domain.DietClassic, domain.DietVegetarian},
			Allergens: []domain.Allergen{domain.AllergenMilk, domain.AllergenGluten},
			Ingredients: []domain.Ingredient{
				{ProductName: "Овсянка"},
				{ProductName: "Молоко 3.2%"},
			},
		},
		{
			ID: saladID, Name: "Салат", MealType: domain.MealLunch,
			Diets: []domain.DietType{domain.DietClassic, domain.DietVegan},
			Ingredients: []domain.Ingredient{
				{ProductName: "Помидор"},
				{ProductName: "Огурец"},
			},
		},
		{
			ID: uuid.New(), Name: "Курица с гречкой", MealType: domain.MealLunch,
			Diets: []domain.DietType{domain.DietClassic, domain.DietPaleo},
			Ingredients: []domain.Ingredient{
				{ProductName: "Куриное филе"},
				{ProductName: "Гречка"},
			},
		},
		{
			ID: uuid.New(), Name: "Лосось", MealType: domain.MealDinner,
			Diets:     []domain.DietType{domain.DietClassic, domain.DietPaleo},
			Allergens: []domain.Allergen{domain.AllergenFish},
			Ingredients: []domain.Ingredient{
				{ProductName: "Лосось"},
			},
		},
	}
	return
}

func TestFilter_DietRestriction(t *testing.T) {
	recipes, _, _ := fixtures()
	// Веган — должна остаться только «Салат»
	got := Filter(recipes, FilterInput{
		Diet: domain.DietVegan,
		ActiveMeals: []domain.MealType{
			domain.MealBreakfast, domain.MealLunch, domain.MealDinner,
		},
	})
	total := 0
	for _, rs := range got {
		total += len(rs)
	}
	if total != 1 {
		t.Errorf("vegan filter: got %d, want 1", total)
	}
	if len(got[domain.MealLunch]) != 1 || got[domain.MealLunch][0].Name != "Салат" {
		t.Errorf("expected 'Салат' as lunch, got %+v", got[domain.MealLunch])
	}
}

func TestFilter_AllergenCutsAllergic(t *testing.T) {
	recipes, _, _ := fixtures()
	got := Filter(recipes, FilterInput{
		Diet:        domain.DietClassic,
		Allergens:   []domain.Allergen{domain.AllergenFish, domain.AllergenMilk},
		ActiveMeals: []domain.MealType{domain.MealBreakfast, domain.MealLunch, domain.MealDinner},
	})
	// Должны выпасть: "Овсянка" (milk) и "Лосось" (fish).
	for _, rs := range got {
		for _, r := range rs {
			if r.Name == "Овсянка" || r.Name == "Лосось" {
				t.Errorf("allergenic recipe %q must be filtered out", r.Name)
			}
		}
	}
}

func TestFilter_ExcludedProductByName(t *testing.T) {
	recipes, _, _ := fixtures()
	got := Filter(recipes, FilterInput{
		Diet:             domain.DietClassic,
		ExcludedProducts: []string{"гречка"},
		ActiveMeals:      []domain.MealType{domain.MealLunch},
	})
	// "Курица с гречкой" должна выпасть.
	for _, r := range got[domain.MealLunch] {
		if r.Name == "Курица с гречкой" {
			t.Error("recipe with excluded product must be filtered out")
		}
	}
}

func TestFilter_ExcludedDish(t *testing.T) {
	recipes, oatsID, _ := fixtures()
	got := Filter(recipes, FilterInput{
		Diet:           domain.DietClassic,
		ExcludedDishes: []uuid.UUID{oatsID},
		ActiveMeals:    []domain.MealType{domain.MealBreakfast, domain.MealLunch},
	})
	if len(got[domain.MealBreakfast]) != 0 {
		t.Errorf("excluded dish must not appear: %+v", got[domain.MealBreakfast])
	}
}

func TestFilter_InactiveMealsHidden(t *testing.T) {
	recipes, _, _ := fixtures()
	got := Filter(recipes, FilterInput{
		Diet:        domain.DietClassic,
		ActiveMeals: []domain.MealType{domain.MealLunch}, // только обед
	})
	if len(got) != 1 {
		t.Errorf("only lunch should be present, got %v", keys(got))
	}
	if _, ok := got[domain.MealBreakfast]; ok {
		t.Error("breakfast must be absent")
	}
}

func TestFilter_EmptyAllergens(t *testing.T) {
	recipes, _, _ := fixtures()
	got := Filter(recipes, FilterInput{
		Diet: domain.DietClassic,
		ActiveMeals: []domain.MealType{
			domain.MealBreakfast, domain.MealLunch, domain.MealDinner,
		},
	})
	total := 0
	for _, rs := range got {
		total += len(rs)
	}
	if total != 4 {
		t.Errorf("classic+no restrictions: got %d, want 4", total)
	}
}

func keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
