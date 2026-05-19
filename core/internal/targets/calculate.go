// Package targets рассчитывает целевые КБЖУ пользователя.
// Источник: docs/МАТМОДЕЛЬ.txt §§1.1-1.3, 2 + docs/ФУНКЦИОНАЛ.md §4.
package targets

import (
	"fmt"

	"nutrition-core/internal/domain"
)

// DietShares — доли макронутриентов для диеты. nil-указатели означают
// «брать из energy_norms» (применимо только к classic).
type DietShares struct {
	Protein *float64
	Fat     *float64
	Carb    *float64
}

// Calculate рассчитывает дневные целевые КБЖУ.
//
// Формула (МАТМОДЕЛЬ.txt §1.2 + §1.3):
//   E* = E_norm × k_goal
//   classic:  P*, F*, C* = (P_norm, F_norm, C_norm) × k_goal
//   non-classic: P* = E*·p_p/4,  F* = E*·p_f/9,  C* = E*·p_c/4
//                (коэффициенты Этуотера, FAO Paper 77)
//
// Manual override (docs/ФУНКЦИОНАЛ.md §4): пользователь может вручную
// скорректировать любое из четырёх значений, но строго в пределах ±15%
// от соответствующей нормы МР; при выходе — зажим на границу
// (box-constraint-projection, МАТМОДЕЛЬ.txt §1.2).
func Calculate(
	profile domain.UserProfile,
	norm domain.EnergyNorm,
	shares DietShares,
) (domain.DailyTargets, error) {
	k := profile.Goal.Factor()

	t := domain.DailyTargets{
		Kcal: norm.KcalNorm * k,
	}
	if isClassic(shares) {
		t.ProteinG = norm.ProteinGNorm * k
		t.FatG = norm.FatGNorm * k
		t.CarbG = norm.CarbGNorm * k
	} else {
		if shares.Protein == nil || shares.Fat == nil || shares.Carb == nil {
			return domain.DailyTargets{}, fmt.Errorf(
				"non-classic diet requires all three shares")
		}
		t.ProteinG = t.Kcal * (*shares.Protein) / 4.0
		t.FatG = t.Kcal * (*shares.Fat) / 9.0
		t.CarbG = t.Kcal * (*shares.Carb) / 4.0
	}

	if o := profile.ManualOverride; o != nil {
		applyOverride(&t, norm, o)
	}
	return t, nil
}

func isClassic(s DietShares) bool {
	return s.Protein == nil && s.Fat == nil && s.Carb == nil
}

func applyOverride(t *domain.DailyTargets, norm domain.EnergyNorm, o *domain.ManualTargets) {
	if o.Kcal != nil {
		t.Kcal = clamp(*o.Kcal, norm.KcalNorm*0.85, norm.KcalNorm*1.15)
	}
	if o.ProteinG != nil {
		t.ProteinG = clamp(*o.ProteinG, norm.ProteinGNorm*0.85, norm.ProteinGNorm*1.15)
	}
	if o.FatG != nil {
		t.FatG = clamp(*o.FatG, norm.FatGNorm*0.85, norm.FatGNorm*1.15)
	}
	if o.CarbG != nil {
		t.CarbG = clamp(*o.CarbG, norm.CarbGNorm*0.85, norm.CarbGNorm*1.15)
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
