package http

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/identity"
)

const (
	accessCookieName  = "ca_access"
	refreshCookieName = "ca_refresh"
)

// AuthHandlers expone las rutas de identidad.
type AuthHandlers struct {
	Manager *identity.AuthManager
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type selectTenantRequest struct {
	MembershipID string `json:"membership_id"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	res, err := h.Manager.Login(r.Context(), req.Email, req.Password, clientIP(r))
	if err != nil {
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "Autenticación", err.Error(), nil)
		return
	}

	// MFA pendiente: se devuelve el token de segundo factor, sin sesión.
	if res.MfaRequired {
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"mfa_required": true,
			"mfa_token":    res.MfaToken,
		})
		return
	}

	setAuthCookies(w, res.RefreshToken, res.AccessToken)

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"user":         res.User,
		"memberships":  res.Memberships,
		"access_token": res.AccessToken,
	})
}

// MfaSetup inicia MFA para el usuario autenticado: genera un secret TOTP y
// devuelve la URL otpauth para el QR. Requiere access token válido.
func (h *AuthHandlers) MfaSetup(w http.ResponseWriter, r *http.Request) {
	claims, err := h.Manager.VerifyAccessToken(h.bearerOrCookie(r))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	setup, err := h.Manager.SetupMfa(r.Context(), claims.Subject)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, setup)
}

type mfaCodeRequest struct {
	MfaToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

// MfaConfirm habilita MFA validando el código TOTP contra el secret generado.
// Requiere access token válido.
func (h *AuthHandlers) MfaConfirm(w http.ResponseWriter, r *http.Request) {
	var req mfaCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	claims, err := h.Manager.VerifyAccessToken(h.bearerOrCookie(r))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	if err := h.Manager.ConfirmMfa(r.Context(), claims.Subject, req.Code); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_mfa_code", "Código inválido", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MfaVerify completa el login con el segundo factor: valida el mfa_token y el
// código TOTP, y emite refresh + access.
func (h *AuthHandlers) MfaVerify(w http.ResponseWriter, r *http.Request) {
	var req mfaCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}
	res, err := h.Manager.VerifyMfa(r.Context(), req.MfaToken, req.Code, clientIP(r))
	if err != nil {
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "MFA", err.Error(), nil)
		return
	}
	setAuthCookies(w, res.RefreshToken, res.AccessToken)
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"user":         res.User,
		"memberships":  res.Memberships,
		"access_token": res.AccessToken,
	})
}

// MfaDisable desactiva MFA para el usuario autenticado.
func (h *AuthHandlers) MfaDisable(w http.ResponseWriter, r *http.Request) {
	claims, err := h.Manager.VerifyAccessToken(h.bearerOrCookie(r))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	if err := h.Manager.DisableMfa(r.Context(), claims.Subject); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandlers) SelectTenant(w http.ResponseWriter, r *http.Request) {
	var req selectTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	claims, err := h.Manager.VerifyAccessToken(h.bearerOrCookie(r))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}

	me, access, err := h.Manager.SelectTenant(r.Context(), claims.Subject, req.MembershipID)
	if err != nil {
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "Membresía", err.Error(), nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	httpapi.WriteJSON(w, http.StatusOK, me)
}

// setAuthCookies fija refresh (con expiración) y access en cookies HttpOnly.
func setAuthCookies(w http.ResponseWriter, refresh, access string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // local; en producción se fija en true
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(720 * time.Hour),
	})
	if access != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     accessCookieName,
			Value:    access,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

// clientIP devuelve la IP remota para el contador de intentos de login.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		first, _, _ := strings.Cut(ip, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Sesión no iniciada", "falta cookie de refresh", nil)
		return
	}

	access, newRefresh, err := h.Manager.Refresh(r.Context(), cookie.Value)
	if err != nil {
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "Sesión", err.Error(), nil)
		return
	}

	setAuthCookies(w, newRefresh, access)

	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"access_token": access})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil {
		_ = h.Manager.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	me, err := h.Manager.Me(r.Context(), claims)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, me)
}

// Memberships lista las membresías del usuario autenticado.
func (h *AuthHandlers) Memberships(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
		return
	}
	memberships, err := h.Manager.ListMemberships(r.Context(), claims)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, memberships)
}

// bearerOrCookie extrae el access token del header Authorization o de la cookie.
func (h *AuthHandlers) bearerOrCookie(r *http.Request) string {
	if tok := r.Header.Get("Authorization"); tok != "" {
		return tok
	}
	if c, err := r.Cookie(accessCookieName); err == nil {
		return c.Value
	}
	return ""
}

func mapAuthError(err error) int {
	switch {
	case errors.Is(err, identity.ErrInvalidCredentials),
		errors.Is(err, identity.ErrInvalidMfaCode),
		errors.Is(err, identity.ErrInvalidMFAToken):
		return http.StatusUnauthorized
	case errors.Is(err, identity.ErrTooManyAttempts):
		return http.StatusTooManyRequests
	case errors.Is(err, identity.ErrUserDisabled),
		errors.Is(err, identity.ErrMembershipInactive):
		return http.StatusForbidden
	case errors.Is(err, identity.ErrMFANotRequired):
		return http.StatusBadRequest
	case errors.Is(err, identity.ErrInvalidRefreshToken),
		errors.Is(err, identity.ErrRefreshTokenExpired),
		errors.Is(err, identity.ErrRefreshTokenReused):
		return http.StatusUnauthorized
	case errors.Is(err, identity.ErrMembershipNotFound):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func authProblemCode(err error) string {
	switch {
	case errors.Is(err, identity.ErrInvalidCredentials):
		return "invalid_credentials"
	case errors.Is(err, identity.ErrInvalidMfaCode):
		return "invalid_mfa_code"
	case errors.Is(err, identity.ErrInvalidMFAToken):
		return "invalid_mfa_token"
	case errors.Is(err, identity.ErrMFANotRequired):
		return "mfa_not_required"
	case errors.Is(err, identity.ErrTooManyAttempts):
		return "too_many_attempts"
	case errors.Is(err, identity.ErrUserDisabled):
		return "user_disabled"
	case errors.Is(err, identity.ErrRefreshTokenReused):
		return "refresh_token_reused"
	case errors.Is(err, identity.ErrRefreshTokenExpired):
		return "refresh_token_expired"
	case errors.Is(err, identity.ErrInvalidRefreshToken):
		return "invalid_refresh_token"
	case errors.Is(err, identity.ErrMembershipNotFound):
		return "membership_not_found"
	case errors.Is(err, identity.ErrMembershipInactive):
		return "membership_inactive"
	default:
		return "internal_error"
	}
}
