package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"nutrition-core/internal/auth"
	"nutrition-core/internal/mailer"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	UserID          string `json:"user_id"`
	Email           string `json:"email"`
	ConfirmToken    string `json:"confirm_token,omitempty"` // только если exposeAuthToken=true
	ConfirmRequired bool   `json:"confirm_required"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	Confirmed bool   `json:"email_confirmed"`
}

type VerifyRequest struct {
	Token string `json:"token"`
}

// PostAuthRegister — регистрация. По docs/ФУНКЦИОНАЛ.md §1.
func (h *Handler) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	email := auth.NormalizeEmail(req.Email)
	if err := auth.ValidateEmail(email); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	confirmToken, err := auth.NewConfirmToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	user, err := h.users.Create(r.Context(), email, hash, confirmToken)
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if h.mailer != nil {
		_ = h.mailer.Send(r.Context(), mailer.Message{
			To:      user.Email,
			Subject: "Подтверждение регистрации — План питания",
			Body: fmt.Sprintf(
				"Здравствуйте!\n\nДля завершения регистрации подтвердите email "+
					"токеном:\n\n  %s\n\nЕсли вы не регистрировались — проигнорируйте письмо.\n",
				confirmToken,
			),
		})
	}

	resp := RegisterResponse{
		UserID:          user.UserID.String(),
		Email:           user.Email,
		ConfirmRequired: true,
	}
	if h.exposeAuthToken {
		resp.ConfirmToken = confirmToken
	}
	writeJSON(w, http.StatusCreated, resp)
}

// PostAuthLogin — логин с возвратом JWT.
func (h *Handler) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	email := auth.NormalizeEmail(req.Email)

	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("неверный email или пароль"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("неверный email или пароль"))
		return
	}

	token, err := auth.GenerateToken(user.UserID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{
		UserID:    user.UserID.String(),
		Email:     user.Email,
		Token:     token,
		Confirmed: user.Confirmed,
	})
}

// PostAuthVerify — подтверждение email по токену.
func (h *Handler) PostAuthVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	email, err := h.users.Verify(r.Context(), req.Token)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":     email,
		"confirmed": true,
	})
}

// GetAuthMe — данные текущего пользователя из токена.
func (h *Handler) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.CurrentClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("not authenticated"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": c.UserID.String(),
		"email":   c.Email,
	})
}
