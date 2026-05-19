package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"nutrition-core/internal/auth"
	"nutrition-core/internal/catalog"
	"nutrition-core/internal/compliance"
	"nutrition-core/internal/config"
	"nutrition-core/internal/domain"
	"nutrition-core/internal/mailer"
	"nutrition-core/internal/micronutrients"
	"nutrition-core/internal/planner"
	"nutrition-core/internal/pricing"
	"nutrition-core/internal/repository"
	"nutrition-core/internal/shopping"
	"nutrition-core/internal/targets"
)

// Handler хранит зависимости HTTP-слоя.
type Handler struct {
	repo            repository.Repo
	cfg             config.Penalty
	pricing         *pricing.Client
	users           *auth.Store
	mailer          mailer.Mailer
	exposeAuthToken bool
}

// NewHandler создаёт обработчик. pricing может быть nil — тогда
// эндпоинт /pricing возвращает 503. users и m обязательны.
// exposeAuthToken=true заставляет /auth/register дополнительно вернуть
// confirm_token в response — это backward-compat дев-режим (на случай,
// когда фронт в e2e-сценариях не имеет доступа к серверным логам).
func NewHandler(
	repo repository.Repo,
	cfg config.Penalty,
	pc *pricing.Client,
	users *auth.Store,
	m mailer.Mailer,
	exposeAuthToken bool,
) *Handler {
	return &Handler{
		repo:            repo,
		cfg:             cfg,
		pricing:         pc,
		users:           users,
		mailer:          m,
		exposeAuthToken: exposeAuthToken,
	}
}

// Health — readiness/liveness проба. Проверяет, что БД отвечает на ping.
// Если БД недоступна — 503 + понятная диагностика (наружу не утекает DSN).
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.repo.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "degraded",
			"db":     "unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"db":     "ok",
	})
}

func (h *Handler) PostPlan(w http.ResponseWriter, r *http.Request) {
	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	// user_id берётся из JWT, а не из тела — нельзя формировать план за чужого.
	if uid, ok := auth.CurrentUser(r.Context()); ok {
		req.UserID = uid
	}

	resp, err := h.process(r.Context(), req)
	if err != nil {
		var notFound notFoundError
		if errors.As(err, &notFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// notFoundError позволяет отделить «не найдено» от внутренней ошибки.
type notFoundError struct{ msg string }

func (e notFoundError) Error() string { return e.msg }

func (h *Handler) process(ctx context.Context, req PlanRequest) (PlanResponse, error) {
	profile, err := decodeProfile(req)
	if err != nil {
		return PlanResponse{}, err
	}
	ageGroup, err := domain.AgeGroupFor(profile.Age)
	if err != nil {
		return PlanResponse{}, err
	}

	norm, err := h.repo.GetEnergyNorm(ctx, profile.Sex, ageGroup, profile.KfaGroup)
	if err != nil {
		return PlanResponse{}, notFoundError{msg: err.Error()}
	}
	pShare, fShare, cShare, err := h.repo.GetDietShares(ctx, profile.DietType)
	if err != nil {
		return PlanResponse{}, err
	}

	daily, err := targets.Calculate(profile, norm, targets.DietShares{
		Protein: pShare, Fat: fShare, Carb: cShare,
	})
	if err != nil {
		return PlanResponse{}, err
	}

	all, err := h.repo.LoadCatalog(ctx)
	if err != nil {
		return PlanResponse{}, err
	}
	recipesByID := make(map[uuid.UUID]domain.Recipe, len(all))
	for _, r := range all {
		recipesByID[r.ID] = r
	}

	excludedDishes := make([]uuid.UUID, len(req.ExcludedDishes))
	copy(excludedDishes, req.ExcludedDishes)
	pinned, pinnedLookup, err := decodePinned(req.PinnedDishes, recipesByID)
	if err != nil {
		return PlanResponse{}, err
	}

	filtered := catalog.Filter(all, catalog.FilterInput{
		Diet:             profile.DietType,
		Allergens:        profile.Allergens,
		ExcludedProducts: profile.ExcludedProducts,
		ExcludedDishes:   excludedDishes,
		ActiveMeals:      profile.Meals,
	})

	plannerInst := planner.New(h.cfg)
	slots, err := plannerInst.PlanAndBalance(planner.Input{
		Daily:        daily,
		ActiveMeals:  profile.Meals,
		Catalog:      filtered,
		Pinned:       pinned,
		PinnedLookup: pinnedLookup,
		TotalDays:    7,
	})
	if err != nil {
		return PlanResponse{}, err
	}

	microNorms, err := h.repo.GetMicroNorms(ctx, profile.Sex, ageGroup)
	if err != nil {
		return PlanResponse{}, err
	}
	deficits := micronutrients.Deficit(slots, 7, microNorms)
	comp := compliance.Check(slots, daily, 7, h.cfg.CorridorRel, microNorms)
	shoppingItems := shopping.Aggregate(slots, profile.Persons)

	return buildResponse(req.UserID, currentWeekRef(time.Now()), slots, shoppingItems, comp, deficits), nil
}

// decodeProfile конвертирует DTO в domain.UserProfile с проверкой enum.
func decodeProfile(req PlanRequest) (domain.UserProfile, error) {
	p := req.Profile
	pr := domain.UserProfile{
		UserID:           req.UserID,
		Sex:              domain.Sex(p.Sex),
		Age:              p.Age,
		HeightCm:         p.HeightCm,
		WeightKg:         p.WeightKg,
		KfaGroup:         domain.KfaGroup(p.KfaGroup),
		Goal:             domain.Goal(p.Goal),
		DietType:         domain.DietType(p.DietType),
		ExcludedProducts: append([]string(nil), p.ExcludedProducts...),
		Persons:          p.Persons,
	}
	if pr.Persons <= 0 {
		pr.Persons = 1
	}
	for _, a := range p.Allergens {
		pr.Allergens = append(pr.Allergens, domain.Allergen(a))
	}
	for _, m := range p.Meals {
		pr.Meals = append(pr.Meals, domain.MealType(m))
	}
	if req.ManualTargetsOverride != nil && req.ManualTargetsOverride.Enabled {
		pr.ManualOverride = &domain.ManualTargets{
			Kcal:     req.ManualTargetsOverride.Kcal,
			ProteinG: req.ManualTargetsOverride.ProteinG,
			FatG:     req.ManualTargetsOverride.FatG,
			CarbG:    req.ManualTargetsOverride.CarbG,
		}
	}
	return pr, nil
}

func decodePinned(
	in []PinnedDishDTO, recipesByID map[uuid.UUID]domain.Recipe,
) ([]domain.PinnedDish, map[uuid.UUID]domain.Recipe, error) {
	pinned := make([]domain.PinnedDish, 0, len(in))
	lookup := make(map[uuid.UUID]domain.Recipe, len(in))
	for _, p := range in {
		r, ok := recipesByID[p.DishID]
		if !ok {
			return nil, nil, notFoundError{msg: fmt.Sprintf("pinned dish %s not in catalog", p.DishID)}
		}
		pinned = append(pinned, domain.PinnedDish{
			Day: p.Day, Meal: domain.MealType(p.Meal), RecipeID: p.DishID,
		})
		lookup[p.DishID] = r
	}
	return pinned, lookup, nil
}

// currentWeekRef — ISO-неделя в формате YYYY-Www.
func currentWeekRef(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
