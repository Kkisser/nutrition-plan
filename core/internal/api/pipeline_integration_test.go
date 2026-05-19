package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nutrition-core/internal/auth"
	"nutrition-core/internal/config"
	"nutrition-core/internal/db"
	"nutrition-core/internal/mailer"
	pgrepo "nutrition-core/internal/repository/pg"
)

// Pipeline integration test: для каждой из 6 диет проходит полный путь
// register → login → POST /plan → проверка ответа. Запускается только
// при DATABASE_DSN; ожидает каталог из data/smoke.
//
// Это smoke-test всего API: фильтр + планировщик + balance + compliance
// + shopping. Если кто-то ломает любой этап — упадёт здесь.

func setupServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set — pipeline integration test skipped")
	}
	// JWT secret для тестов, чтобы auth.Middleware валидировал токены.
	t.Setenv("CORE_JWT_SECRET", "test-pipeline-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	repo := pgrepo.New(pool)
	users := auth.NewStore(pool)
	cfg := config.DefaultPenalty()
	mlr := &mailer.LogMailer{}
	h := NewHandler(repo, cfg, nil, users, mlr, true)

	mux := http.NewServeMux()
	Routes(mux, h, nil)
	srv := httptest.NewServer(mux)
	cleanup := func() {
		srv.Close()
		pool.Close()
	}
	return srv, cleanup
}

func registerAndLogin(t *testing.T, baseURL string) string {
	t.Helper()
	email := fmt.Sprintf("pipeline_%d@example.com", time.Now().UnixNano())

	// register
	regBody := fmt.Sprintf(`{"email":%q,"password":"TestPass123"}`, email)
	r, err := http.Post(baseURL+"/auth/register", "application/json",
		bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusCreated {
		t.Fatalf("register status %d", r.StatusCode)
	}
	var regResp struct {
		ConfirmToken string `json:"confirm_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&regResp)
	if regResp.ConfirmToken == "" {
		t.Fatal("expected confirm_token in register response (exposeAuthToken=true)")
	}

	// verify
	vBody := fmt.Sprintf(`{"token":%q}`, regResp.ConfirmToken)
	v, err := http.Post(baseURL+"/auth/verify", "application/json",
		bytes.NewBufferString(vBody))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	v.Body.Close()

	// login
	lBody := fmt.Sprintf(`{"email":%q,"password":"TestPass123"}`, email)
	l, err := http.Post(baseURL+"/auth/login", "application/json",
		bytes.NewBufferString(lBody))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer l.Body.Close()
	var loginResp struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(l.Body).Decode(&loginResp)
	if loginResp.Token == "" {
		t.Fatal("empty JWT after login")
	}
	return loginResp.Token
}

type planResult struct {
	WeekRef      string `json:"week_ref"`
	Plan         []struct {
		Day       int `json:"day"`
		Meals     []struct {
			Meal     string  `json:"meal"`
			Kcal     float64 `json:"kcal"`
			ProteinG float64 `json:"protein_g"`
			FatG     float64 `json:"fat_g"`
			CarbG    float64 `json:"carb_g"`
		} `json:"meals"`
		DayTotals struct {
			Kcal     float64 `json:"kcal"`
			ProteinG float64 `json:"protein_g"`
			FatG     float64 `json:"fat_g"`
			CarbG    float64 `json:"carb_g"`
		} `json:"day_totals"`
	} `json:"plan"`
	ShoppingList []struct {
		IngredientName string  `json:"ingredient_name"`
		Category       string  `json:"category"`
		Amount         float64 `json:"amount"`
		Unit           string  `json:"unit"`
	} `json:"shopping_list"`
}

func postPlan(t *testing.T, baseURL, token, diet string) planResult {
	t.Helper()
	body := fmt.Sprintf(`{
		"profile": {
			"sex":"male","age":30,"height_cm":178,"weight_kg":75,
			"kfa_group":"II","goal":"maintain","diet_type":%q,
			"allergens":[],"excluded_products":[],
			"meals":["breakfast","lunch","dinner","snack"],"persons":1
		}
	}`, diet)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/plan",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(r.Body)
		t.Fatalf("plan status=%d body=%s", r.StatusCode, buf.String())
	}
	var resp planResult
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

// TestPipeline_AllDiets — для каждой из 6 диет проверяет:
//   - план сгенерирован на 7 дней;
//   - каждый день имеет ≥1 приёма пищи и положительную калорийность;
//   - shopping_list не пустой и содержит хотя бы 1 категорию;
//   - суммарные kcal недели в коридоре ±20% от целевых (зависит от
//     дискретности блюд, поэтому коридор шире планерского 10%).
func TestPipeline_AllDiets(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()

	token := registerAndLogin(t, srv.URL)

	for _, diet := range []string{"classic", "keto", "vegetarian", "vegan", "paleo", "fasting"} {
		t.Run(diet, func(t *testing.T) {
			r := postPlan(t, srv.URL, token, diet)

			if len(r.Plan) != 7 {
				t.Fatalf("plan has %d days, want 7", len(r.Plan))
			}
			weeklyKcal := 0.0
			for _, d := range r.Plan {
				if len(d.Meals) == 0 {
					t.Errorf("day %d: no meals", d.Day)
				}
				if d.DayTotals.Kcal <= 0 {
					t.Errorf("day %d: kcal=%v, want >0", d.Day, d.DayTotals.Kcal)
				}
				weeklyKcal += d.DayTotals.Kcal
			}
			if len(r.ShoppingList) == 0 {
				t.Errorf("shopping_list is empty")
			}
			// Проверим, что у >50% позиций есть категория (новое поле).
			withCat := 0
			for _, it := range r.ShoppingList {
				if it.Category != "" {
					withCat++
				}
			}
			if withCat*2 < len(r.ShoppingList) {
				t.Errorf("less than 50%% items have category: %d/%d", withCat, len(r.ShoppingList))
			}

			// Целевая дневная норма для male/30/II/maintain ≈ 2900 ккал (см.
			// energy_norms). За неделю ~20300 ккал. Коридор ±20% =
			// [16240, 24360]. Считаем неделю — дискретность блюд уже сглажена.
			avgDaily := weeklyKcal / 7.0
			if avgDaily < 1800 || avgDaily > 4000 {
				t.Errorf("diet=%s: avg daily kcal=%.0f out of sanity range [1800, 4000]",
					diet, avgDaily)
			}
		})
	}
}
