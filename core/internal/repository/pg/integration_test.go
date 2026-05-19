package pg

import (
	"context"
	"os"
	"testing"
	"time"

	"nutrition-core/internal/db"
	"nutrition-core/internal/domain"
)

// Запускается только при DATABASE_DSN, иначе skip. Ожидает smoke-данные
// в БД (loader load-all --data-dir loader/data/smoke).
func setup(t *testing.T) *Repo {
	t.Helper()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set — integration test skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return New(pool)
}

func TestGetEnergyNorm_Smoke(t *testing.T) {
	r := setup(t)
	ctx := context.Background()

	n, err := r.GetEnergyNorm(ctx, domain.SexMale, domain.Age18_29, domain.KfaII)
	if err != nil {
		t.Fatalf("GetEnergyNorm: %v", err)
	}
	if n.KcalNorm != 2800 {
		t.Errorf("male/18-29/II kcal_norm = %v, want 2800", n.KcalNorm)
	}
	if n.ProteinGNorm != 80 || n.FatGNorm != 93 || n.CarbGNorm != 411 {
		t.Errorf("unexpected BJU: P=%v F=%v C=%v", n.ProteinGNorm, n.FatGNorm, n.CarbGNorm)
	}
}

// TestGetEnergyNorm_NotFound удалён: после расширения energy_norms
// до полного покрытия МР 2.3.1.0253-21 (5 возрастов × 4 КФА × 2 пола)
// нет пар, отсутствующих в БД. Negative-case теперь покрывается
// unit-тестом targets.Calculate через мок репозитория.

func TestGetDietShares_Keto(t *testing.T) {
	r := setup(t)
	p, f, c, err := r.GetDietShares(context.Background(), domain.DietKeto)
	if err != nil {
		t.Fatalf("GetDietShares: %v", err)
	}
	if p == nil || f == nil || c == nil {
		t.Fatal("keto shares must be non-nil")
	}
	// Текущие доли keto после миграции 0006: 20/75/5 (Volek/Phinney +
	// Paoli + StatPearls). Если значения изменятся, проверь миграцию.
	if *p != 0.20 || *f != 0.75 || *c != 0.05 {
		t.Errorf("keto shares = %v/%v/%v, want 0.20/0.75/0.05", *p, *f, *c)
	}
}

func TestGetDietShares_Classic(t *testing.T) {
	r := setup(t)
	p, f, c, err := r.GetDietShares(context.Background(), domain.DietClassic)
	if err != nil {
		t.Fatalf("GetDietShares: %v", err)
	}
	if p != nil || f != nil || c != nil {
		t.Errorf("classic shares must be NULL, got %v/%v/%v", p, f, c)
	}
}

func TestLoadCatalog_Smoke(t *testing.T) {
	r := setup(t)
	recipes, err := r.LoadCatalog(context.Background())
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	// Каталог растёт с расширением диет — фиксируем нижнюю границу,
	// а не точное число. Текущая планка: ≥ 60 рецептов (smoke + fasting +
	// keto-расширение). Если станет меньше — это потеря данных.
	if len(recipes) < 60 {
		t.Fatalf("got %d recipes, want >=60 (smoke + extensions)", len(recipes))
	}

	// Каждый рецепт должен иметь >0 КБЖУ и хотя бы один ингредиент.
	for _, rec := range recipes {
		if len(rec.Ingredients) == 0 {
			t.Errorf("recipe %q has no ingredients", rec.Name)
		}
		if rec.Kcal <= 0 || rec.ProteinG < 0 || rec.FatG < 0 || rec.CarbG < 0 {
			t.Errorf("recipe %q has invalid nutrition: %+v", rec.Name, rec)
		}
	}

	// Проверим конкретное блюдо: "Овсянка с яблоком".
	// Овсянка 60 г: kcal_100=342 → 205.2 ккал
	// Молоко 200 мл (трактуем как 200 г): kcal_100=58 → 116
	// Яблоко 100 г: kcal_100=47 → 47
	// Итого ≈ 368.2 ккал
	var oats *domain.Recipe
	for i := range recipes {
		if recipes[i].ExternalID == "oats_apple" {
			oats = &recipes[i]
			break
		}
	}
	if oats == nil {
		t.Fatal("recipe oats_apple not found")
	}
	if len(oats.Ingredients) != 3 {
		t.Errorf("oats_apple has %d ingredients, want 3", len(oats.Ingredients))
	}
	const expectedKcal = 205.2 + 116.0 + 47.0
	const eps = 0.5
	if oats.Kcal < expectedKcal-eps || oats.Kcal > expectedKcal+eps {
		t.Errorf("oats_apple kcal = %v, want ~%v", oats.Kcal, expectedKcal)
	}

	// Совместимость с диетами.
	var hasClassic, hasVegetarian bool
	for _, d := range oats.Diets {
		if d == domain.DietClassic {
			hasClassic = true
		}
		if d == domain.DietVegetarian {
			hasVegetarian = true
		}
	}
	if !hasClassic || !hasVegetarian {
		t.Errorf("oats_apple diets = %v, expected classic+vegetarian", oats.Diets)
	}

	// Микронутриенты — должны быть посчитаны.
	if len(oats.Micronutrients) == 0 {
		t.Error("oats_apple has no computed micronutrients")
	}
}

func TestGetMicronutrients_Smoke(t *testing.T) {
	r := setup(t)
	ms, err := r.GetMicronutrients(context.Background())
	if err != nil {
		t.Fatalf("GetMicronutrients: %v", err)
	}
	if len(ms) != 6 {
		t.Errorf("got %d micronutrients, want 6 (smoke)", len(ms))
	}
}

func TestGetMicroNorms_Smoke(t *testing.T) {
	r := setup(t)
	norms, err := r.GetMicroNorms(context.Background(), domain.SexMale, domain.Age18_29)
	if err != nil {
		t.Fatalf("GetMicroNorms: %v", err)
	}
	if len(norms) != 6 {
		t.Errorf("got %d norms, want 6", len(norms))
	}
}

// TestCatalogCoverage_PerDietMeal — регрессия покрытия каталога.
// Для каждой пары (диета × приём) в БД должно быть не меньше 3 рецептов:
// планировщику нужно >1 кандидата, чтобы balance-фаза могла свопать слоты,
// а 3 — комфортный минимум для разнообразия 7-дневного плана.
// Если CSV-загрузка обрезала каталог или кто-то удалил блюдо — тест упадёт
// с указанием конкретной пары.
func TestCatalogCoverage_PerDietMeal(t *testing.T) {
	r := setup(t)
	recipes, err := r.LoadCatalog(context.Background())
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	type key struct {
		diet domain.DietType
		meal domain.MealType
	}
	counts := make(map[key]int)
	for _, rec := range recipes {
		for _, d := range rec.Diets {
			counts[key{d, rec.MealType}]++
		}
	}

	diets := []domain.DietType{
		domain.DietClassic, domain.DietKeto, domain.DietVegetarian,
		domain.DietVegan, domain.DietPaleo, domain.DietFasting,
	}
	meals := []domain.MealType{
		domain.MealBreakfast, domain.MealLunch, domain.MealDinner, domain.MealSnack,
	}
	const minPerPair = 3
	for _, d := range diets {
		for _, m := range meals {
			got := counts[key{d, m}]
			if got < minPerPair {
				t.Errorf("diet=%s meal=%s: %d recipes, want >=%d",
					d, m, got, minPerPair)
			}
		}
	}
}
