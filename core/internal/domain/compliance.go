package domain

// Compliance — результат проверки коридора плана.
// docs/КОНТРАКТ_API.md §2.2, docs/МАТМОДЕЛЬ.txt §4.4.
type Compliance struct {
	InCorridor bool
	Violations []Violation
	Message    string
}

type Violation struct {
	Day     int     // 0 = недельный показатель
	Metric  string  // "kcal", "protein_g", "fat_g", "carb_g", "vit_d", ...
	Value   float64
	Target  float64
	Comment string
}
