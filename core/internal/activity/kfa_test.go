package activity

import (
	"testing"

	"nutrition-core/internal/domain"
)

func TestDeriveKFA(t *testing.T) {
	cases := []struct {
		name string
		s    Survey
		want domain.KfaGroup
	}{
		// IV — физически тяжёлая работа.
		{"IV by Q1=heavy", Survey{Q1: Q1HeavyPhysical, Q3: Q3None}, domain.KfaIV},
		{"IV by intense 6+", Survey{Q1: Q1Sedentary, Q3: Q3SixPlus, Q4: Q4Intense}, domain.KfaIV},

		// III — частое перемещение или умеренная/интенсивная 3-5/нед.
		{"III by Q1=frequent", Survey{Q1: Q1FrequentMovement, Q3: Q3None}, domain.KfaIII},
		{"III by moderate 3-5", Survey{Q1: Q1Sedentary, Q3: Q3ThreeFive, Q4: Q4Moderate}, domain.KfaIII},
		{"III by intense 3-5", Survey{Q1: Q1Sedentary, Q3: Q3ThreeFive, Q4: Q4Intense}, domain.KfaIII},

		// II — стоячая, либо 1-2/нед, либо лёгкая 3+/нед.
		{"II by Q1=standing", Survey{Q1: Q1StandingLow, Q3: Q3None}, domain.KfaII},
		{"II by 1-2 light", Survey{Q1: Q1Sedentary, Q3: Q3OneTwo, Q4: Q4Light}, domain.KfaII},
		{"II by light 3-5", Survey{Q1: Q1Sedentary, Q3: Q3ThreeFive, Q4: Q4Light}, domain.KfaII},
		{"II by light 6+", Survey{Q1: Q1Sedentary, Q3: Q3SixPlus, Q4: Q4Light}, domain.KfaII},

		// I — сидячая, нагрузок нет.
		{"I default", Survey{Q1: Q1Sedentary, Q3: Q3None}, domain.KfaI},
		// Q1=sedentary, Q3=6+ moderate — пограничный, попадает в I по строгой
		// реализации (не подходит ни под III, ни под IV).
		{"I edge case moderate 6+", Survey{Q1: Q1Sedentary, Q3: Q3SixPlus, Q4: Q4Moderate}, domain.KfaI},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := DeriveKFA(c.s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("DeriveKFA(%+v) = %q, want %q", c.s, got, c.want)
			}
		})
	}
}

func TestDeriveKFA_Validation(t *testing.T) {
	bad := []Survey{
		{Q1: "bogus", Q3: Q3None},
		{Q1: Q1Sedentary, Q3: "bogus"},
		{Q1: Q1Sedentary, Q3: Q3OneTwo, Q4: ""},        // Q4 обязателен
		{Q1: Q1Sedentary, Q3: Q3None, Q4: Q4Light},     // Q4 запрещён
	}
	for _, s := range bad {
		if _, err := DeriveKFA(s); err == nil {
			t.Errorf("expected error for %+v", s)
		}
	}
}
