package domain

// EnergyNorm — нормативное значение из energy_norms (МР 2.3.1.0253-21).
type EnergyNorm struct {
	Sex          Sex
	AgeGroup     AgeGroup
	KfaGroup     KfaGroup
	KcalNorm     float64
	ProteinGNorm float64
	FatGNorm     float64
	CarbGNorm    float64
}

// DailyTargets — итоговое дневное целевое число (норма × k_goal).
type DailyTargets struct {
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbG    float64
}

// MealTargets — целевые показатели одного приёма пищи после
// ренормализации долей и вычитания закреплённых блюд (МАТМОДЕЛЬ.txt §2).
type MealTargets struct {
	Meal     MealType
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbG    float64
}
