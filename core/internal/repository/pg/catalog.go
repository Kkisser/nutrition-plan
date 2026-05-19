package pg

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// LoadCatalog загружает все рецепты с предрасчитанным КБЖУ и микронутриентами.
//
// Источник расчёта: МАТМОДЕЛЬ.txt §1.3 — КБЖУ блюда есть взвешенная сумма
// содержаний по продуктам пропорционально их массе (на 100 г сырого).
//
// Упрощение по единицам: amount в recipe_ingredients интерпретируется как
// граммы независимо от unit. Для smoke-набора это корректно (молоко считаем
// 1 г/мл, штучных ингредиентов в smoke нет). Полная поддержка pcs требует
// поля «вес штуки» в products — отдельная задача.
func (r *Repo) LoadCatalog(ctx context.Context) ([]domain.Recipe, error) {
	products, err := r.loadProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("load products: %w", err)
	}
	productMicros, err := r.loadProductMicros(ctx)
	if err != nil {
		return nil, fmt.Errorf("load product micronutrients: %w", err)
	}

	recipes, recipeIdx, err := r.loadRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("load recipes: %w", err)
	}
	if err := r.loadIngredients(ctx, recipes, recipeIdx, products); err != nil {
		return nil, fmt.Errorf("load ingredients: %w", err)
	}
	if err := r.loadDietCompat(ctx, recipes, recipeIdx); err != nil {
		return nil, fmt.Errorf("load diet compat: %w", err)
	}
	if err := r.loadAllergens(ctx, recipes, recipeIdx); err != nil {
		return nil, fmt.Errorf("load allergens: %w", err)
	}

	// Предрасчёт КБЖУ и микронутриентов из ингредиентов.
	for i := range recipes {
		computeNutrition(&recipes[i], products, productMicros)
	}
	return recipes, nil
}

func (r *Repo) loadProducts(ctx context.Context) (map[uuid.UUID]domain.Product, error) {
	const q = `
		SELECT product_id, name, COALESCE(category, ''),
		       kcal_100, protein_100, fat_100, carb_100, default_unit
		  FROM products
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID]domain.Product)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Category,
			&p.Kcal100, &p.Protein100, &p.Fat100, &p.Carb100, &p.DefaultUnit,
		); err != nil {
			return nil, err
		}
		out[p.ID] = p
	}
	return out, rows.Err()
}

// productMicros: product_id -> nutrient_id -> amount_100
func (r *Repo) loadProductMicros(ctx context.Context) (map[uuid.UUID]map[string]float64, error) {
	const q = `SELECT product_id, nutrient_id, amount_100 FROM product_micronutrients`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID]map[string]float64)
	for rows.Next() {
		var pid uuid.UUID
		var nid string
		var amt float64
		if err := rows.Scan(&pid, &nid, &amt); err != nil {
			return nil, err
		}
		if out[pid] == nil {
			out[pid] = make(map[string]float64)
		}
		out[pid][nid] = amt
	}
	return out, rows.Err()
}

func (r *Repo) loadRecipes(ctx context.Context) ([]domain.Recipe, map[uuid.UUID]int, error) {
	const q = `
		SELECT recipe_id, COALESCE(external_id, ''), name, instruction,
		       COALESCE(cook_time_min, 0), base_portions, meal_type
		  FROM recipes
		 ORDER BY name
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var recipes []domain.Recipe
	idx := make(map[uuid.UUID]int)
	for rows.Next() {
		var rec domain.Recipe
		if err := rows.Scan(
			&rec.ID, &rec.ExternalID, &rec.Name, &rec.Instruction,
			&rec.CookTimeMin, &rec.BasePortions, &rec.MealType,
		); err != nil {
			return nil, nil, err
		}
		rec.Micronutrients = make(map[string]float64)
		idx[rec.ID] = len(recipes)
		recipes = append(recipes, rec)
	}
	return recipes, idx, rows.Err()
}

func (r *Repo) loadIngredients(
	ctx context.Context,
	recipes []domain.Recipe,
	recipeIdx map[uuid.UUID]int,
	products map[uuid.UUID]domain.Product,
) error {
	const q = `SELECT recipe_id, product_id, amount, unit FROM recipe_ingredients`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rid, pid uuid.UUID
		var amount float64
		var unit domain.Unit
		if err := rows.Scan(&rid, &pid, &amount, &unit); err != nil {
			return err
		}
		ri, ok := recipeIdx[rid]
		if !ok {
			continue
		}
		prod, ok := products[pid]
		if !ok {
			return fmt.Errorf("ingredient references missing product %s", pid)
		}
		recipes[ri].Ingredients = append(recipes[ri].Ingredients, domain.Ingredient{
			ProductID:       pid,
			ProductName:     prod.Name,
			ProductCategory: prod.Category,
			Amount:          amount,
			Unit:            unit,
		})
	}
	return rows.Err()
}

func (r *Repo) loadDietCompat(
	ctx context.Context, recipes []domain.Recipe, recipeIdx map[uuid.UUID]int,
) error {
	const q = `SELECT recipe_id, diet_id FROM recipe_diet_compat`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rid uuid.UUID
		var diet domain.DietType
		if err := rows.Scan(&rid, &diet); err != nil {
			return err
		}
		if ri, ok := recipeIdx[rid]; ok {
			recipes[ri].Diets = append(recipes[ri].Diets, diet)
		}
	}
	return rows.Err()
}

func (r *Repo) loadAllergens(
	ctx context.Context, recipes []domain.Recipe, recipeIdx map[uuid.UUID]int,
) error {
	const q = `SELECT recipe_id, allergen FROM recipe_allergens`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rid uuid.UUID
		var allergen domain.Allergen
		if err := rows.Scan(&rid, &allergen); err != nil {
			return err
		}
		if ri, ok := recipeIdx[rid]; ok {
			recipes[ri].Allergens = append(recipes[ri].Allergens, allergen)
		}
	}
	return rows.Err()
}

// computeNutrition пересчитывает КБЖУ и микронутриенты блюда по ингредиентам.
// Допущение: amount в граммах (см. doc-комментарий LoadCatalog).
func computeNutrition(
	rec *domain.Recipe,
	products map[uuid.UUID]domain.Product,
	productMicros map[uuid.UUID]map[string]float64,
) {
	for _, ing := range rec.Ingredients {
		p, ok := products[ing.ProductID]
		if !ok {
			continue
		}
		factor := ing.Amount / 100.0
		rec.Kcal += p.Kcal100 * factor
		rec.ProteinG += p.Protein100 * factor
		rec.FatG += p.Fat100 * factor
		rec.CarbG += p.Carb100 * factor

		if micros := productMicros[ing.ProductID]; micros != nil {
			for nid, amt100 := range micros {
				rec.Micronutrients[nid] += amt100 * factor
			}
		}
	}
}
