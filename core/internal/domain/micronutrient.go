package domain

// Micronutrient — справочник из таблицы micronutrients.
type Micronutrient struct {
	ID       string  // nutrient_id, напр. "ca", "fe", "vit_d"
	Name     string
	NormUnit string  // mg, mcg, IU
	ULValue  float64 // верхний допустимый уровень; 0 = не задан
}

// MicroNorm — норма потребления по (пол, возрастная группа).
type MicroNorm struct {
	NutrientID string
	Sex        Sex
	AgeGroup   AgeGroup
	NormValue  float64
}

// MicroDeficit — итоговый недельный недобор, пересчитанный на сутки.
// docs/МАТМОДЕЛЬ.txt §7.2: Δ_i = max(0, N_norm_i − N_plan_i).
type MicroDeficit struct {
	NutrientID    string
	DeficitPerDay float64
}
