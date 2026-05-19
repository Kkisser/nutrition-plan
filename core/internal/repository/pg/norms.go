package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"nutrition-core/internal/domain"
)

// GetEnergyNorm читает строку из energy_norms по (sex, age_group, kfa_group).
func (r *Repo) GetEnergyNorm(
	ctx context.Context, sex domain.Sex, age domain.AgeGroup, kfa domain.KfaGroup,
) (domain.EnergyNorm, error) {
	const q = `
		SELECT sex, age_group, kfa_group,
		       kcal_norm, protein_g_norm, fat_g_norm, carb_g_norm
		  FROM energy_norms
		 WHERE sex = $1 AND age_group = $2 AND kfa_group = $3
	`
	var n domain.EnergyNorm
	row := r.pool.QueryRow(ctx, q, sex, age, kfa)
	if err := row.Scan(
		&n.Sex, &n.AgeGroup, &n.KfaGroup,
		&n.KcalNorm, &n.ProteinGNorm, &n.FatGNorm, &n.CarbGNorm,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.EnergyNorm{}, fmt.Errorf(
				"energy norm not found for %s/%s/%s", sex, age, kfa)
		}
		return domain.EnergyNorm{}, fmt.Errorf("query energy_norm: %w", err)
	}
	return n, nil
}

// GetDietShares возвращает доли БЖУ для типа диеты.
// Для classic возвращает (nil, nil, nil, nil) — значения берутся
// из energy_norms на стороне расчёта целевых.
func (r *Repo) GetDietShares(
	ctx context.Context, diet domain.DietType,
) (*float64, *float64, *float64, error) {
	const q = `SELECT protein_share, fat_share, carb_share FROM diets WHERE diet_id = $1`
	var p, f, c *float64
	if err := r.pool.QueryRow(ctx, q, diet).Scan(&p, &f, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, fmt.Errorf("diet not found: %s", diet)
		}
		return nil, nil, nil, fmt.Errorf("query diet shares: %w", err)
	}
	return p, f, c, nil
}

// GetMicronutrients возвращает справочник микронутриентов.
func (r *Repo) GetMicronutrients(ctx context.Context) ([]domain.Micronutrient, error) {
	const q = `SELECT nutrient_id, name, norm_unit, COALESCE(ul_value, 0) FROM micronutrients ORDER BY nutrient_id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query micronutrients: %w", err)
	}
	defer rows.Close()

	var out []domain.Micronutrient
	for rows.Next() {
		var m domain.Micronutrient
		if err := rows.Scan(&m.ID, &m.Name, &m.NormUnit, &m.ULValue); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMicroNorms возвращает нормы микронутриентов для (sex, age_group).
func (r *Repo) GetMicroNorms(
	ctx context.Context, sex domain.Sex, age domain.AgeGroup,
) ([]domain.MicroNorm, error) {
	const q = `
		SELECT nutrient_id, sex, age_group, norm_value
		  FROM micronutrient_norms
		 WHERE sex = $1 AND age_group = $2
		 ORDER BY nutrient_id
	`
	rows, err := r.pool.Query(ctx, q, sex, age)
	if err != nil {
		return nil, fmt.Errorf("query micronutrient_norms: %w", err)
	}
	defer rows.Close()

	var out []domain.MicroNorm
	for rows.Next() {
		var n domain.MicroNorm
		if err := rows.Scan(&n.NutrientID, &n.Sex, &n.AgeGroup, &n.NormValue); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
