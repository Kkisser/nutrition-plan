// Package auth — регистрация, вход, JWT-токены и middleware.
// Реализует раздел 1 docs/ФУНКЦИОНАЛ.md.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenLifetime = 30 * 24 * time.Hour // 30 дней
)

// jwtSecret загружается лениво из CORE_JWT_SECRET или генерируется случайно при старте.
// В проде CORE_JWT_SECRET обязателен (иначе токены инвалидируются при рестарте).
var jwtSecret = func() []byte {
	if s := os.Getenv("CORE_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}()

// Claims — содержимое JWT.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

var (
	ErrUserExists      = errors.New("auth: user already exists")
	ErrUserNotFound    = errors.New("auth: user not found")
	ErrInvalidPassword = errors.New("auth: invalid password")
	ErrTokenInvalid    = errors.New("auth: token invalid or expired")
	ErrNotVerified     = errors.New("auth: email not verified")
	ErrBadEmail        = errors.New("auth: invalid email format")
	ErrWeakPassword    = errors.New("auth: password does not meet policy")
)

// HashPassword возвращает bcrypt-хэш с дефолтной стоимостью.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword возвращает nil если пароль совпадает с хэшем.
func CheckPassword(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// GenerateToken создаёт JWT для пользователя.
func GenerateToken(userID uuid.UUID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken проверяет JWT и возвращает claims.
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// NewConfirmToken генерирует одноразовый токен подтверждения email.
func NewConfirmToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidateEmail — формат + базовые правила из docs/ФУНКЦИОНАЛ.md §1.
var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func ValidateEmail(email string) error {
	if !emailRe.MatchString(strings.TrimSpace(email)) {
		return ErrBadEmail
	}
	return nil
}

// ValidatePassword — политика по docs/ФУНКЦИОНАЛ.md §1:
// >= 8 символов, латиница (строчные + прописные), хотя бы одна цифра,
// без пробелов и кириллицы.
func ValidatePassword(pwd string) error {
	if len(pwd) < 8 {
		return fmt.Errorf("%w: too short", ErrWeakPassword)
	}
	if strings.ContainsAny(pwd, " \t\n") {
		return fmt.Errorf("%w: contains whitespace", ErrWeakPassword)
	}
	if regexp.MustCompile(`[А-Яа-яЁё]`).MatchString(pwd) {
		return fmt.Errorf("%w: contains cyrillic", ErrWeakPassword)
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(pwd)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(pwd)
	hasDigit := regexp.MustCompile(`\d`).MatchString(pwd)
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("%w: must include upper, lower latin and a digit", ErrWeakPassword)
	}
	return nil
}

// NormalizeEmail приводит к нижнему регистру + trim.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
