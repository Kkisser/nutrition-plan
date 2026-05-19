// Package domain содержит value-объекты ядра.
// Источник: docs/СХЕМА_БД.md, docs/КОНТРАКТ_API.md, docs/МАТМОДЕЛЬ.txt.
package domain

import "fmt"

type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

type KfaGroup string

const (
	KfaI   KfaGroup = "I"
	KfaII  KfaGroup = "II"
	KfaIII KfaGroup = "III"
	KfaIV  KfaGroup = "IV"
)

// AgeGroup — возрастные группы взрослого населения по МР 2.3.1.0253-21
// (МАТМОДЕЛЬ.txt §1.1).
type AgeGroup string

const (
	Age18_29 AgeGroup = "18-29"
	Age30_44 AgeGroup = "30-44"
	Age45_64 AgeGroup = "45-64"
	Age65_74 AgeGroup = "65-74"
	Age75    AgeGroup = "75+"
)

// AgeGroupFor возвращает группу по возрасту в годах.
func AgeGroupFor(years int) (AgeGroup, error) {
	switch {
	case years >= 18 && years <= 29:
		return Age18_29, nil
	case years >= 30 && years <= 44:
		return Age30_44, nil
	case years >= 45 && years <= 64:
		return Age45_64, nil
	case years >= 65 && years <= 74:
		return Age65_74, nil
	case years >= 75:
		return Age75, nil
	default:
		return "", fmt.Errorf("age %d out of supported range (>=18)", years)
	}
}

type DietType string

const (
	DietClassic    DietType = "classic"
	DietKeto       DietType = "keto"
	DietVegetarian DietType = "vegetarian"
	DietVegan      DietType = "vegan"
	DietPaleo      DietType = "paleo"
	DietFasting    DietType = "fasting"
)

// Goal — направление коррекции калорийности. Поправка ±15 % к норме МР
// (МАТМОДЕЛЬ.txt §1.2).
type Goal string

const (
	GoalDeficit  Goal = "deficit"
	GoalMaintain Goal = "maintain"
	GoalSurplus  Goal = "surplus"
)

// Factor возвращает коэффициент k_goal для поправки энергонормы.
func (g Goal) Factor() float64 {
	switch g {
	case GoalDeficit:
		return 0.85
	case GoalSurplus:
		return 1.15
	default:
		return 1.00
	}
}

type MealType string

const (
	MealBreakfast MealType = "breakfast"
	MealLunch     MealType = "lunch"
	MealDinner    MealType = "dinner"
	MealSnack     MealType = "snack"
)

// MealShare — фиксированные доли распределения дневной калорийности
// по приёмам пищи (МАТМОДЕЛЬ.txt §2). Нумерация: breakfast, lunch, dinner, snack.
var MealShare = map[MealType]float64{
	MealBreakfast: 0.25,
	MealLunch:     0.35,
	MealDinner:    0.30,
	MealSnack:     0.10,
}

type Unit string

const (
	UnitG   Unit = "g"
	UnitMl  Unit = "ml"
	UnitPcs Unit = "pcs"
)

type Allergen string

const (
	AllergenMilk      Allergen = "milk"
	AllergenEggs      Allergen = "eggs"
	AllergenFish      Allergen = "fish"
	AllergenGluten    Allergen = "gluten"
	AllergenPeanut    Allergen = "peanut"
	AllergenSesame    Allergen = "sesame"
	AllergenShellfish Allergen = "shellfish"
	AllergenSoy       Allergen = "soy"
	AllergenNuts      Allergen = "nuts"
)
