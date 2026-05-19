package config

import "testing"

func TestDefaultPenalty(t *testing.T) {
	p := DefaultPenalty()
	if p.W1 != 1.0 || p.W2 != 1.0 || p.W3 != 0.5 || p.K != 3 ||
		p.CorridorRel != 0.10 || p.ReplaceMaxRise != 1.8 {
		t.Errorf("default penalty = %+v", p)
	}
}

func TestLoadPenalty_FromEnv(t *testing.T) {
	t.Setenv("CORE_W1", "2.5")
	t.Setenv("CORE_W2", "0.7")
	t.Setenv("CORE_W3", "0.1")
	t.Setenv("CORE_K", "5")

	p, err := LoadPenalty()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.W1 != 2.5 || p.W2 != 0.7 || p.W3 != 0.1 || p.K != 5 {
		t.Errorf("got %+v", p)
	}
}

func TestLoadPenalty_Invalid(t *testing.T) {
	t.Setenv("CORE_W1", "not-a-number")
	if _, err := LoadPenalty(); err == nil {
		t.Error("expected error on invalid CORE_W1")
	}
}

func TestLoadPenalty_NegativeK(t *testing.T) {
	t.Setenv("CORE_K", "-1")
	if _, err := LoadPenalty(); err == nil {
		t.Error("expected error on negative CORE_K")
	}
}
