package pg

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// GetCarryover читает недобор микронутриентов за указанную неделю.
// Возвращает пустой срез при отсутствии записей.
func (r *Repo) GetCarryover(
	ctx context.Context, userID uuid.UUID, weekRef string,
) ([]domain.MicroDeficit, error) {
	const q = `
		SELECT nutrient_id, deficit_per_day
		  FROM micronutrient_carryover
		 WHERE user_id = $1 AND week_ref = $2
		 ORDER BY nutrient_id
	`
	rows, err := r.pool.Query(ctx, q, userID, weekRef)
	if err != nil {
		return nil, fmt.Errorf("query carryover: %w", err)
	}
	defer rows.Close()

	var out []domain.MicroDeficit
	for rows.Next() {
		var d domain.MicroDeficit
		if err := rows.Scan(&d.NutrientID, &d.DeficitPerDay); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
