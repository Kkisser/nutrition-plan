package compliance

import (
	"testing"

	"nutrition-core/internal/domain"
)

func slot(day int, kcal, p, f, c float64) domain.PlanSlot {
	return domain.PlanSlot{Day: day, Kcal: kcal, ProteinG: p, FatG: f, CarbG: c, Portions: 1.0}
}

func TestCheck_InCorridor(t *testing.T) {
	plan := []domain.PlanSlot{
		slot(1, 700, 25, 23, 100),
		slot(1, 1100, 35, 35, 160), // итого: 1800 ккал ~ target 2000 ±10%
		slot(1, 200, 5, 5, 30),     // 2000 ккал, 65, 63, 290
	}
	daily := domain.DailyTargets{Kcal: 2000, ProteinG: 70, FatG: 70, CarbG: 290}
	c := Check(plan, daily, 1, 0.10, nil)
	if !c.InCorridor {
		t.Errorf("expected in corridor, got violations: %+v", c.Violations)
	}
}

func TestCheck_OutOfCorridor(t *testing.T) {
	plan := []domain.PlanSlot{
		slot(1, 3000, 50, 50, 50), // сильное превышение по ккал
	}
	daily := domain.DailyTargets{Kcal: 2000, ProteinG: 70, FatG: 70, CarbG: 290}
	c := Check(plan, daily, 1, 0.10, nil)
	if c.InCorridor {
		t.Error("expected out of corridor")
	}
	if len(c.Violations) == 0 {
		t.Error("expected violations list non-empty")
	}
}

func TestCheck_MicronutrientDeficitDoesNotBreakCorridor(t *testing.T) {
	// План с попаданием в коридор по КБЖУ, но с недобором микронутриента —
	// in_corridor должен оставаться true (микронутриенты не входят в основной
	// критерий, МАТМОДЕЛЬ.txt §7).
	plan := []domain.PlanSlot{{Day: 1, Kcal: 2000, ProteinG: 70, FatG: 70, CarbG: 290, Portions: 1}}
	daily := domain.DailyTargets{Kcal: 2000, ProteinG: 70, FatG: 70, CarbG: 290}
	norms := []domain.MicroNorm{{NutrientID: "ca", NormValue: 1000}}

	c := Check(plan, daily, 1, 0.10, norms)
	if !c.InCorridor {
		t.Error("micronutrient deficit must NOT break daily corridor")
	}
	if len(c.Violations) == 0 {
		t.Error("expected micronutrient deficit to be reported in violations")
	}
}
