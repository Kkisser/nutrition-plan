package planner

import (
	"math"

	"github.com/google/uuid"

	"nutrition-core/internal/config"
	"nutrition-core/internal/domain"
)

// Penalty считает штраф F(d, t, m) из МАТМОДЕЛЬ.txt §3.2:
//
//	F = w1 · |E_d − E*_m|/E*_m
//	  + w2 · (1/3) · ( |P_d−P*|/P* + |F_d−F*|/F* + |C_d−C*|/C* )
//	  + w3 · 1[ τ(d, t) ≤ k ]
//
// E_d, P_d, F_d, C_d — КБЖУ блюда С УЧЁТОМ множителя порции α.
// E*_m, … — целевые показатели приёма пищи.
// lastSeen — день последнего включения блюда (≤0 = никогда).
// today — номер текущего дня (1..7).
func Penalty(
	dish domain.Recipe,
	alpha float64,
	target domain.MealTargets,
	lastSeen int,
	today int,
	cfg config.Penalty,
) float64 {
	e := dish.Kcal * alpha
	p := dish.ProteinG * alpha
	f := dish.FatG * alpha
	c := dish.CarbG * alpha

	relE := relDev(e, target.Kcal)
	relP := relDev(p, target.ProteinG)
	relF := relDev(f, target.FatG)
	relC := relDev(c, target.CarbG)

	var repeat float64
	if lastSeen > 0 && (today-lastSeen) <= cfg.K {
		repeat = 1.0
	}

	return cfg.W1*relE +
		cfg.W2*(relP+relF+relC)/3.0 +
		cfg.W3*repeat
}

// relDev — |x − target| / target. При target == 0 возвращает 0
// (защита от деления на ноль; в норме target > 0).
func relDev(x, target float64) float64 {
	if target <= 0 {
		return 0
	}
	return math.Abs(x-target) / target
}

// ScaledSlot — слот плана после применения порции α к блюду.
type ScaledSlot struct {
	Recipe   domain.Recipe
	Alpha    float64
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbG    float64
	Micros   map[string]float64
}

// Scale умножает КБЖУ и микронутриенты блюда на α.
func Scale(r domain.Recipe, alpha float64) ScaledSlot {
	s := ScaledSlot{
		Recipe:   r,
		Alpha:    alpha,
		Kcal:     r.Kcal * alpha,
		ProteinG: r.ProteinG * alpha,
		FatG:     r.FatG * alpha,
		CarbG:    r.CarbG * alpha,
		Micros:   make(map[string]float64, len(r.Micronutrients)),
	}
	for k, v := range r.Micronutrients {
		s.Micros[k] = v * alpha
	}
	return s
}

// dishKey — псевдоним для читаемости.
type dishKey = uuid.UUID
