// Package config держит настраиваемые параметры алгоритма.
//
// Решение по открытому вопросу №3 (вариант C):
//   w1 — вес отклонения по калориям;
//   w2 — вес отклонения по белкам/жирам/углеводам;
//   w3 — штраф за повторение блюда в последние k дней;
//   k  — горизонт повторяемости (в днях);
//   CorridorRel    — допустимое относительное дневное отклонение для проверки
//                    коридора (МАТМОДЕЛЬ.txt §4.4); дефолт 0.10 (±10%);
//   ReplaceMaxRise — c в ограничении F(new) ≤ c·F(old) при замене в фазе 2
//                    (МАТМОДЕЛЬ.txt §4.5); дефолт 1.8.
//
// Дефолты выбраны как нейтральные: w1 = w2 = 1.0 (равный вклад энергии
// и БЖУ в штраф, нормированы относительной L1-нормой из МАТМОДЕЛЬ.txt §3.2),
// w3 = 0.5 (мягкий штраф — повтор допустим, но снижает рейтинг блюда),
// k = 3 (типичный пользовательский горизонт «не повторять в ближайшие
// три дня»). Все шесть переопределяемы из окружения.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Penalty — настраиваемые параметры штрафной функции F (МАТМОДЕЛЬ.txt §3.2)
// и проверки коридора (МАТМОДЕЛЬ §4.4–4.5).
type Penalty struct {
	W1             float64
	W2             float64
	W3             float64
	K              int
	CorridorRel    float64
	ReplaceMaxRise float64
}

// DefaultPenalty возвращает дефолты (см. doc-комментарий пакета).
func DefaultPenalty() Penalty {
	return Penalty{
		W1:             1.0,
		W2:             1.0,
		W3:             0.5,
		K:              3,
		CorridorRel:    0.10,
		ReplaceMaxRise: 1.8,
	}
}

// LoadPenalty читает Penalty из переменных окружения, падая на дефолт
// при отсутствии значения. Имена: CORE_W1, CORE_W2, CORE_W3, CORE_K.
// Возвращает ошибку при синтаксически некорректном значении в ENV.
func LoadPenalty() (Penalty, error) {
	p := DefaultPenalty()

	if v, ok := os.LookupEnv("CORE_W1"); ok {
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return p, fmt.Errorf("CORE_W1: %w", err)
		}
		p.W1 = x
	}
	if v, ok := os.LookupEnv("CORE_W2"); ok {
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return p, fmt.Errorf("CORE_W2: %w", err)
		}
		p.W2 = x
	}
	if v, ok := os.LookupEnv("CORE_W3"); ok {
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return p, fmt.Errorf("CORE_W3: %w", err)
		}
		p.W3 = x
	}
	if v, ok := os.LookupEnv("CORE_K"); ok {
		x, err := strconv.Atoi(v)
		if err != nil {
			return p, fmt.Errorf("CORE_K: %w", err)
		}
		if x < 0 {
			return p, fmt.Errorf("CORE_K must be non-negative, got %d", x)
		}
		p.K = x
	}
	if v, ok := os.LookupEnv("CORE_CORRIDOR"); ok {
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return p, fmt.Errorf("CORE_CORRIDOR: %w", err)
		}
		if x < 0 || x > 1 {
			return p, fmt.Errorf("CORE_CORRIDOR must be in [0,1], got %v", x)
		}
		p.CorridorRel = x
	}
	if v, ok := os.LookupEnv("CORE_REPLACE_MAX_RISE"); ok {
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return p, fmt.Errorf("CORE_REPLACE_MAX_RISE: %w", err)
		}
		if x < 1 {
			return p, fmt.Errorf("CORE_REPLACE_MAX_RISE must be >= 1, got %v", x)
		}
		p.ReplaceMaxRise = x
	}
	return p, nil
}
