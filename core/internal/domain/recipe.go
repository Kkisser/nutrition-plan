package domain

import "github.com/google/uuid"

// Ingredient — позиция рецептуры блюда (масса × единица).
type Ingredient struct {
	ProductID       uuid.UUID
	ProductName     string
	ProductCategory string
	Amount          float64
	Unit            Unit
}

// Recipe — курируемое блюдо с рассчитанным КБЖУ.
// КБЖУ — взвешенная сумма содержаний по продуктам пропорционально массе
// (docs/МАТМОДЕЛЬ.txt §1.3).
type Recipe struct {
	ID            uuid.UUID
	ExternalID    string
	Name          string
	Instruction   string
	CookTimeMin   int
	BasePortions  int
	MealType      MealType
	Ingredients   []Ingredient
	Diets         []DietType
	Allergens     []Allergen
	// Pre-computed nutrition per one base portion:
	Kcal          float64
	ProteinG      float64
	FatG          float64
	CarbG         float64
	Micronutrients map[string]float64 // nutrient_id → amount per portion
}
