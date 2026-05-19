// Package activity вычисляет группу коэффициента физической активности
// (КФА) по ответам мини-анкеты.
//
// Источник правил: docs/АНКЕТА_активности.md, дерево решений без Q2/Q5
// (зафиксированное решение по открытому вопросу №1).
//
// Анкета содержит три вопроса:
//   Q1 — характер обычной дневной активности;
//   Q3 — частота отдельной физической нагрузки длительностью не менее 30 мин;
//   Q4 — интенсивность этой нагрузки (nullable, не задаётся при Q3=none).
package activity

import (
	"fmt"

	"nutrition-core/internal/domain"
)

type Q1 string

const (
	Q1Sedentary         Q1 = "sedentary"
	Q1StandingLow       Q1 = "standing_low"
	Q1FrequentMovement  Q1 = "frequent_movement"
	Q1HeavyPhysical     Q1 = "heavy_physical"
)

type Q3 string

const (
	Q3None    Q3 = "none"
	Q3OneTwo  Q3 = "1_to_2"
	Q3ThreeFive Q3 = "3_to_5"
	Q3SixPlus Q3 = "6_plus"
)

type Q4 string

const (
	Q4None     Q4 = ""
	Q4Light    Q4 = "light"
	Q4Moderate Q4 = "moderate"
	Q4Intense  Q4 = "intense"
)

// Survey — нормализованные ответы пользователя.
type Survey struct {
	Q1 Q1
	Q3 Q3
	Q4 Q4 // пустая, если Q3 == none
}

// DeriveKFA реализует дерево решений по docs/АНКЕТА_активности.md.
// Решение по открытому вопросу №1 — вариант B (без Q2/Q5):
//
//   IV  = Q1=heavy_physical OR (Q3=6_plus AND Q4=intense)
//   III = Q1=frequent_movement OR (Q3=3_to_5 AND Q4 in {moderate, intense})
//   II  = Q1=standing_low
//          OR Q3=1_to_2
//          OR (Q3 in {3_to_5, 6_plus} AND Q4=light)
//   I   = иначе
//
// Порядок проверки сверху вниз — попадание в более высокую группу
// фиксируется первым.
func DeriveKFA(s Survey) (domain.KfaGroup, error) {
	if err := s.validate(); err != nil {
		return "", err
	}

	// IV
	if s.Q1 == Q1HeavyPhysical {
		return domain.KfaIV, nil
	}
	if s.Q3 == Q3SixPlus && s.Q4 == Q4Intense {
		return domain.KfaIV, nil
	}

	// III
	if s.Q1 == Q1FrequentMovement {
		return domain.KfaIII, nil
	}
	if s.Q3 == Q3ThreeFive && (s.Q4 == Q4Moderate || s.Q4 == Q4Intense) {
		return domain.KfaIII, nil
	}

	// II
	if s.Q1 == Q1StandingLow {
		return domain.KfaII, nil
	}
	if s.Q3 == Q3OneTwo {
		return domain.KfaII, nil
	}
	if (s.Q3 == Q3ThreeFive || s.Q3 == Q3SixPlus) && s.Q4 == Q4Light {
		return domain.KfaII, nil
	}

	return domain.KfaI, nil
}

func (s Survey) validate() error {
	switch s.Q1 {
	case Q1Sedentary, Q1StandingLow, Q1FrequentMovement, Q1HeavyPhysical:
	default:
		return fmt.Errorf("activity: invalid Q1 %q", s.Q1)
	}
	switch s.Q3 {
	case Q3None, Q3OneTwo, Q3ThreeFive, Q3SixPlus:
	default:
		return fmt.Errorf("activity: invalid Q3 %q", s.Q3)
	}
	if s.Q3 == Q3None {
		if s.Q4 != Q4None {
			return fmt.Errorf("activity: Q4 must be empty when Q3=none, got %q", s.Q4)
		}
		return nil
	}
	switch s.Q4 {
	case Q4Light, Q4Moderate, Q4Intense:
	default:
		return fmt.Errorf("activity: Q4 required when Q3!=none, got %q", s.Q4)
	}
	return nil
}
