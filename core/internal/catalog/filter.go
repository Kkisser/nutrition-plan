// Package catalog отсеивает блюда несовместимые с профилем пользователя.
// Источник: docs/ФУНКЦИОНАЛ.md §6, docs/АЛГОРИТМ_блок-схема.svg (блок Б4).
package catalog

import (
	"strings"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// FilterInput — то, по чему отсеиваются рецепты.
type FilterInput struct {
	Diet             domain.DietType
	Allergens        []domain.Allergen
	ExcludedProducts []string      // имя продукта, регистронезависимо
	ExcludedDishes   []uuid.UUID
	ActiveMeals      []domain.MealType
}

// Filter возвращает допустимые блюда, сгруппированные по типу приёма пищи.
//
// Правила (ФУНКЦИОНАЛ §6, цитируется):
//   "из каталога предварительно отсеиваются блюда, не соответствующие
//    типу питания, содержащие аллергены или индивидуально исключаемые
//    продукты, входящие в список исключённых блюд пользователя, либо
//    не соответствующие типу приёма пищи".
func Filter(all []domain.Recipe, in FilterInput) map[domain.MealType][]domain.Recipe {
	activeMeals := setOf(in.ActiveMeals)
	allergens := allergenSet(in.Allergens)
	excludedDishes := uuidSet(in.ExcludedDishes)
	excludedProducts := normalizedSet(in.ExcludedProducts)

	out := make(map[domain.MealType][]domain.Recipe)
	for _, r := range all {
		if !activeMeals[r.MealType] {
			continue
		}
		if excludedDishes[r.ID] {
			continue
		}
		if !dietOk(r, in.Diet) {
			continue
		}
		if hasAnyAllergen(r, allergens) {
			continue
		}
		if hasExcludedProduct(r, excludedProducts) {
			continue
		}
		out[r.MealType] = append(out[r.MealType], r)
	}
	return out
}

func dietOk(r domain.Recipe, diet domain.DietType) bool {
	for _, d := range r.Diets {
		if d == diet {
			return true
		}
	}
	return false
}

func hasAnyAllergen(r domain.Recipe, allergens map[domain.Allergen]struct{}) bool {
	if len(allergens) == 0 {
		return false
	}
	for _, a := range r.Allergens {
		if _, ok := allergens[a]; ok {
			return true
		}
	}
	return false
}

func hasExcludedProduct(r domain.Recipe, excluded map[string]struct{}) bool {
	if len(excluded) == 0 {
		return false
	}
	for _, ing := range r.Ingredients {
		if _, ok := excluded[strings.ToLower(strings.TrimSpace(ing.ProductName))]; ok {
			return true
		}
	}
	return false
}

func setOf(meals []domain.MealType) map[domain.MealType]bool {
	out := make(map[domain.MealType]bool, len(meals))
	for _, m := range meals {
		out[m] = true
	}
	return out
}

func allergenSet(a []domain.Allergen) map[domain.Allergen]struct{} {
	out := make(map[domain.Allergen]struct{}, len(a))
	for _, x := range a {
		out[x] = struct{}{}
	}
	return out
}

func uuidSet(ids []uuid.UUID) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func normalizedSet(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, n := range names {
		out[strings.ToLower(strings.TrimSpace(n))] = struct{}{}
	}
	return out
}
