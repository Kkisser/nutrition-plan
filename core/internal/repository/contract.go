// Package repository описывает контракты доступа к БД.
// Реализация — в подпакете pg/.
package repository

import (
	"context"

	"github.com/google/uuid"

	"nutrition-core/internal/domain"
)

// Repo — единый интерфейс ядра к слою хранения.
//
// Каждая группа методов отделена по доменной области и подключается
// в ядре отдельно. Тесты могут подменить любую реализацию через моки.
type Repo interface {
	GetEnergyNorm(ctx context.Context, sex domain.Sex, age domain.AgeGroup, kfa domain.KfaGroup) (domain.EnergyNorm, error)
	GetDietShares(ctx context.Context, diet domain.DietType) (proteinShare, fatShare, carbShare *float64, err error)
	LoadCatalog(ctx context.Context) ([]domain.Recipe, error)
	GetMicronutrients(ctx context.Context) ([]domain.Micronutrient, error)
	GetMicroNorms(ctx context.Context, sex domain.Sex, age domain.AgeGroup) ([]domain.MicroNorm, error)
	GetCarryover(ctx context.Context, userID uuid.UUID, weekRef string) ([]domain.MicroDeficit, error)
	// Ping — проверка живости хранилища. Используется /health для readiness.
	Ping(ctx context.Context) error
}
