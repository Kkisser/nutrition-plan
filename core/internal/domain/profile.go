package domain

import "github.com/google/uuid"

// UserProfile — входные данные пользователя для формирования плана.
// Источник: docs/КОНТРАКТ_API.md §2.1, docs/СХЕМА_БД.md USERS.
type UserProfile struct {
	UserID            uuid.UUID
	Sex               Sex
	Age               int
	HeightCm          int
	WeightKg          float64
	KfaGroup          KfaGroup
	Goal              Goal
	DietType          DietType
	Allergens         []Allergen
	ExcludedProducts  []string
	Meals             []MealType
	Persons           int
	ManualOverride    *ManualTargets
}

// ManualTargets — ручная корректировка КБЖУ (валиден диапазон ±15 % от нормы).
type ManualTargets struct {
	Kcal      *float64
	ProteinG  *float64
	FatG      *float64
	CarbG     *float64
}

// PinnedDish — закреплённое пользователем блюдо (docs/ФУНКЦИОНАЛ.md §10).
type PinnedDish struct {
	Day      int      // 1..7
	Meal     MealType
	RecipeID uuid.UUID
}

// MicroCarryover — недобор микронутриентов с предыдущей недели
// (docs/МАТМОДЕЛЬ.txt §7.2).
type MicroCarryover struct {
	WeekRef  string
	Deficits []MicroDeficitPerDay
}

type MicroDeficitPerDay struct {
	NutrientID     string
	DeficitPerDay  float64
}
