package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanSlot — позиция плана (блюдо в день/приём пищи).
// docs/СХЕМА_БД.md §5 MEAL_PLAN_SLOTS.
type PlanSlot struct {
	Day       int      // 1..7
	Meal      MealType
	Recipe    Recipe
	Portions  float64 // α из МАТМОДЕЛЬ §5
	Pinned    bool
	// Дневная пищевая ценность позиции с учётом множителя порций:
	Kcal      float64
	ProteinG  float64
	FatG      float64
	CarbG     float64
}

// DayTotals — суммарные дневные показатели после фазы 2.
type DayTotals struct {
	Day      int
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbG    float64
}

// MealPlan — недельный план питания.
type MealPlan struct {
	UserID    uuid.UUID
	WeekRef   string // ISO-неделя "2026-W20"
	DateStart time.Time
	DateEnd   time.Time
	Slots     []PlanSlot
	DayTotals []DayTotals
}
