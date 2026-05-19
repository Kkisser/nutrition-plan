package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"nutrition-core/internal/catalog"
	"nutrition-core/internal/domain"
)

// CatalogItem — облегчённое представление блюда для UI замены.
// Без рецептуры — она не нужна для выбора.
type CatalogItem struct {
	DishID    string  `json:"dish_id"`
	Name      string  `json:"name"`
	Meal      string  `json:"meal"`
	Kcal      float64 `json:"kcal"`
	ProteinG  float64 `json:"protein_g"`
	FatG      float64 `json:"fat_g"`
	CarbG     float64 `json:"carb_g"`
	CookTime  int     `json:"cook_time_min"`
}

// GetCatalog — GET /catalog?meal=...&diet=...&allergens=a,b&excluded_products=...&excluded_dishes=...
// Возвращает блюда для конкретного приёма пищи с учётом фильтров профиля.
// Используется страницей замены блюда (docs/ФУНКЦИОНАЛ.md §9).
func (h *Handler) GetCatalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	mealStr := q.Get("meal")
	if mealStr == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("meal is required"))
		return
	}
	meal := domain.MealType(mealStr)

	diet := domain.DietType(q.Get("diet"))
	if diet == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("diet is required"))
		return
	}

	allergens := splitNonEmpty(q.Get("allergens"))
	excludedProducts := splitNonEmpty(q.Get("excluded_products"))
	excludedDishes := splitUUIDs(q.Get("excluded_dishes"))

	all, err := h.repo.LoadCatalog(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	allergensDomain := make([]domain.Allergen, 0, len(allergens))
	for _, a := range allergens {
		allergensDomain = append(allergensDomain, domain.Allergen(a))
	}

	filtered := catalog.Filter(all, catalog.FilterInput{
		Diet:             diet,
		Allergens:        allergensDomain,
		ExcludedProducts: excludedProducts,
		ExcludedDishes:   excludedDishes,
		ActiveMeals:      []domain.MealType{meal},
	})

	items := make([]CatalogItem, 0, len(filtered[meal]))
	for _, r := range filtered[meal] {
		items = append(items, CatalogItem{
			DishID:   r.ID.String(),
			Name:     r.Name,
			Meal:     string(r.MealType),
			Kcal:     round1(r.Kcal),
			ProteinG: round1(r.ProteinG),
			FatG:     round1(r.FatG),
			CarbG:    round1(r.CarbG),
			CookTime: r.CookTimeMin,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitUUIDs(s string) []uuid.UUID {
	parts := splitNonEmpty(s)
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		if u, err := uuid.Parse(p); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// небольшое замечание: json marshaling этого map[string]any создаёт корневой
// объект {"items": [...]}; UI читает через .items. Это удобнее на расширение,
// чем голый массив (например, потом можно добавить total/cursor).
var _ = json.Marshal // silence linter; пакет используется в writeJSON
