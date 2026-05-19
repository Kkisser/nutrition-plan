package domain

import "github.com/google/uuid"

// Product — продукт из справочника Скурихина-Тутельяна.
// docs/СХЕМА_БД.md §3 PRODUCTS.
type Product struct {
	ID          uuid.UUID
	Name        string
	Category    string
	Kcal100     float64
	Protein100  float64
	Fat100      float64
	Carb100     float64
	DefaultUnit Unit
}

// ProductMicro — содержание микронутриента в продукте (на 100 г сырого).
type ProductMicro struct {
	ProductID  uuid.UUID
	NutrientID string
	Amount100  float64
}
