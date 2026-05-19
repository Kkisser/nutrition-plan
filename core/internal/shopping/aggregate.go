// Package shopping агрегирует список покупок из плана питания.
// Источник: docs/ФУНКЦИОНАЛ.md §11, docs/КОНТРАКТ_API.md §2.2.
package shopping

import (
	"sort"
	"strings"

	"nutrition-core/internal/domain"
)

// Aggregate возвращает агрегированный список покупок:
//   - одноимённые продукты разных блюд суммируются;
//   - количество умножается на множитель порции (α) и на persons;
//   - формат соответствует items[] price-service.
//
// Ключ агрегации: (нормализованное имя продукта, единица). Если у одного
// и того же продукта в разных рецептах разные единицы (что нештатно
// после нормализации в loader) — суммируем отдельно, не смешиваем.
func Aggregate(plan []domain.PlanSlot, persons int) []domain.ShoppingItem {
	if persons <= 0 {
		persons = 1
	}
	type key struct {
		name string
		unit domain.Unit
	}
	acc := make(map[key]float64)
	displayName := make(map[key]string)
	displayCategory := make(map[key]string)

	for _, s := range plan {
		for _, ing := range s.Recipe.Ingredients {
			k := key{
				name: strings.ToLower(strings.TrimSpace(ing.ProductName)),
				unit: ing.Unit,
			}
			acc[k] += ing.Amount * s.Portions * float64(persons)
			if _, ok := displayName[k]; !ok {
				displayName[k] = ing.ProductName
				displayCategory[k] = ing.ProductCategory
			}
		}
	}

	out := make([]domain.ShoppingItem, 0, len(acc))
	for k, amount := range acc {
		out = append(out, domain.ShoppingItem{
			IngredientName: displayName[k],
			Category:       displayCategory[k],
			Amount:         round2(amount),
			Unit:           k.unit,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IngredientName == out[j].IngredientName {
			return out[i].Unit < out[j].Unit
		}
		return out[i].IngredientName < out[j].IngredientName
	})
	return out
}

func round2(x float64) float64 {
	return float64(int64(x*100+0.5)) / 100
}
