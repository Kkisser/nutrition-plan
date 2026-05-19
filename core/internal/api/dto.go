// Package api — HTTP-слой ядра по docs/КОНТРАКТ_API.md §2.
package api

import "github.com/google/uuid"

// PlanRequest — вход POST /plan (КОНТРАКТ_API.md §2.1).
type PlanRequest struct {
	UserID                 uuid.UUID              `json:"user_id"`
	Profile                ProfileDTO             `json:"profile"`
	ManualTargetsOverride  *ManualOverrideDTO     `json:"manual_targets_override,omitempty"`
	PinnedDishes           []PinnedDishDTO        `json:"pinned_dishes,omitempty"`
	ExcludedDishes         []uuid.UUID            `json:"excluded_dishes,omitempty"`
	MicronutrientCarryover *MicroCarryoverDTO     `json:"micronutrient_carryover,omitempty"`
}

type ProfileDTO struct {
	Sex              string   `json:"sex"`
	Age              int      `json:"age"`
	HeightCm         int      `json:"height_cm"`
	WeightKg         float64  `json:"weight_kg"`
	KfaGroup         string   `json:"kfa_group"`
	Goal             string   `json:"goal"`
	DietType         string   `json:"diet_type"`
	Allergens        []string `json:"allergens"`
	ExcludedProducts []string `json:"excluded_products"`
	Meals            []string `json:"meals"`
	Persons          int      `json:"persons"`
}

type ManualOverrideDTO struct {
	Enabled  bool     `json:"enabled"`
	Kcal     *float64 `json:"kcal,omitempty"`
	ProteinG *float64 `json:"protein_g,omitempty"`
	FatG     *float64 `json:"fat_g,omitempty"`
	CarbG    *float64 `json:"carb_g,omitempty"`
}

type PinnedDishDTO struct {
	Day      int       `json:"day"`
	Meal     string    `json:"meal"`
	DishID   uuid.UUID `json:"dish_id"`
}

type MicroCarryoverDTO struct {
	WeekRef  string             `json:"week_ref"`
	Deficits []MicroDeficitDTO  `json:"deficits"`
}

type MicroDeficitDTO struct {
	NutrientID     string  `json:"nutrient_id"`
	DeficitPerDay  float64 `json:"deficit_per_day"`
}

// PlanResponse — выход POST /plan (КОНТРАКТ_API.md §2.2).
type PlanResponse struct {
	UserID                     uuid.UUID          `json:"user_id"`
	WeekRef                    string             `json:"week_ref"`
	Plan                       []DayDTO           `json:"plan"`
	ShoppingList               []ShoppingItemDTO  `json:"shopping_list"`
	Compliance                 ComplianceDTO      `json:"compliance"`
	MicronutrientCarryoverNext MicroCarryoverDTO  `json:"micronutrient_carryover_next"`
}

type DayDTO struct {
	Day       int           `json:"day"`
	Meals     []MealSlotDTO `json:"meals"`
	DayTotals DayTotalsDTO  `json:"day_totals"`
}

type MealSlotDTO struct {
	Meal      string    `json:"meal"`
	DishID    uuid.UUID `json:"dish_id"`
	DishTitle string    `json:"dish_title"`
	Portions  float64   `json:"portions"`
	Pinned    bool      `json:"pinned"`
	Kcal      float64   `json:"kcal"`
	ProteinG  float64   `json:"protein_g"`
	FatG      float64   `json:"fat_g"`
	CarbG     float64   `json:"carb_g"`
}

type DayTotalsDTO struct {
	Kcal     float64 `json:"kcal"`
	ProteinG float64 `json:"protein_g"`
	FatG     float64 `json:"fat_g"`
	CarbG    float64 `json:"carb_g"`
}

type ShoppingItemDTO struct {
	IngredientName string  `json:"ingredient_name"`
	Category       string  `json:"category,omitempty"`
	Amount         float64 `json:"amount"`
	Unit           string  `json:"unit"`
}

type ComplianceDTO struct {
	InCorridor bool           `json:"in_corridor"`
	Violations []ViolationDTO `json:"violations"`
	Message    string         `json:"message,omitempty"`
}

type ViolationDTO struct {
	Day     int     `json:"day"`
	Metric  string  `json:"metric"`
	Value   float64 `json:"value"`
	Target  float64 `json:"target"`
	Comment string  `json:"comment,omitempty"`
}
