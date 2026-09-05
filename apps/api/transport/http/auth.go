package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/documentos"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/brandall2021/consorcioabierto/internal/identity"
	"github.com/brandall2021/consorcioabierto/internal/tenancy"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	accessCookieName  = "ca_access"
	refreshCookieName = "ca_refresh"
)

// AuthHandlers expone las rutas de identidad.
type AuthHandlers struct {
	Manager *identity.AuthManager
	Audit   *audit.Recorder
	Docs     documentos.DocsEnv
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
		h.recordAudit(r, audit.Event{
			Accion:      "auth.login.failed",
			RecursoType: "user",
			RecursoID:   req.Email,
			Diff:        map[string]any{"reason": authProblemCode(err)},
		})
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "Autenticación", err.Error(), nil)
		return
	}

	// MFA pendiente: se devuelve el token de segundo factor, sin sesión.
	if res.MfaRequired {
		h.recordAudit(r, audit.Event{
			ActorID:     res.User.ID,
			Accion:      "auth.login.mfa_pending",
			RecursoType: "user",
			RecursoID:   res.User.ID,
		})
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"mfa_required": true,
			"mfa_token":    res.MfaToken,
		})
		return
	}

	setAuthCookies(w, res.RefreshToken, res.AccessToken)

	h.recordAudit(r, audit.Event{
		ActorID:     res.User.ID,
		Accion:      "auth.login.success",
		RecursoType: "user",
		RecursoID:   res.User.ID,
	})

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
	h.recordAudit(r, audit.Event{
		ActorID:     claims.Subject,
		Accion:      "auth.mfa.setup",
		RecursoType: "user",
		RecursoID:   claims.Subject,
	})
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
	h.recordAudit(r, audit.Event{
		ActorID:     claims.Subject,
		Accion:      "auth.mfa.confirm",
		RecursoType: "user",
		RecursoID:   claims.Subject,
	})
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
		h.recordAudit(r, audit.Event{
			Accion:      "auth.mfa.verify.failed",
			RecursoType: "user",
			Diff:        map[string]any{"reason": authProblemCode(err)},
		})
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "MFA", err.Error(), nil)
		return
	}
	setAuthCookies(w, res.RefreshToken, res.AccessToken)
	h.recordAudit(r, audit.Event{
		ActorID:     res.User.ID,
		Accion:      "auth.mfa.verify",
		RecursoType: "user",
		RecursoID:   res.User.ID,
	})
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
	h.recordAudit(r, audit.Event{
		ActorID:     claims.Subject,
		Accion:      "auth.mfa.disable",
		RecursoType: "user",
		RecursoID:   claims.Subject,
	})
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

	h.recordAudit(r, audit.Event{
		ActorID:         claims.Subject,
		TenantID:        me.Membership.TenantID,
		ActorMembership: me.Membership.ID,
		Accion:          "auth.select_tenant",
		RecursoType:     "membership",
		RecursoID:       me.Membership.ID,
	})

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
		h.recordAudit(r, audit.Event{
			Accion:      "auth.refresh.failed",
			RecursoType: "session",
			Diff:        map[string]any{"reason": authProblemCode(err)},
		})
		httpapi.WriteProblem(w, r, mapAuthError(err), authProblemCode(err), "Sesión", err.Error(), nil)
		return
	}

	setAuthCookies(w, newRefresh, access)

	h.recordAudit(r, audit.Event{
		Accion:      "auth.refresh",
		RecursoType: "session",
	})

	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"access_token": access})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil {
		_ = h.Manager.Logout(r.Context(), cookie.Value)
		h.recordAudit(r, audit.Event{
			Accion:      "auth.logout",
			RecursoType: "session",
		})
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

// listAuditEventsHandler devuelve la auditoría append-only del tenant activo
// (permiso auditoria.read), filtrada y paginada por cursor keyset.
func (h *AuthHandlers) listAuditEventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		if claims == nil {
			httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "claims ausentes", nil)
			return
		}

		f := audit.Filter{Limit: 50}
		if cursor := r.URL.Query().Get("cursor"); cursor != "" {
			c, err := audit.DecodeCursor(cursor)
			if err != nil {
				httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_cursor", "Cursor inválido", err.Error(), nil)
				return
			}
			f.Cursor = c
		}
		f.Accion = r.URL.Query().Get("accion")
		f.RecursoType = r.URL.Query().Get("recurso_type")
		if s := r.URL.Query().Get("desde"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "desde inválido", err.Error(), nil)
				return
			}
			f.Desde = &t
		}
		if s := r.URL.Query().Get("hasta"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "hasta inválido", err.Error(), nil)
				return
			}
			f.Hasta = &t
		}

		ctx := r.Context()
		pool := h.Manager.Pool()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := tenancy.SetContext(ctx, tx, claims.Subject, claims.Tenant); err != nil {
			httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
			return
		}

		events, err := h.Audit.List(ctx, db.New(tx), f)
		if err != nil {
			httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
			return
		}

		// next_cursor = clave keyset del último evento (si se trajo una página completa).
		var nextCursor any
		if len(events) == f.Limit {
			last := events[len(events)-1]
			nextCursor = audit.EncodeCursor(&audit.Cursor{CreatedAt: last.CreatedAt.Time, ID: last.ID.String()})
		}

		dto := make([]audit.EventDTO, 0, len(events))
		for _, e := range events {
			dto = append(dto, audit.ToDTO(e))
		}

		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"data": dto,
			"meta": map[string]any{
				"request_id":  middleware.GetReqID(ctx),
				"next_cursor": nextCursor,
			},
		})
	}
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

// recordAudit registra un evento de auditoría best-effort: un fallo de
// auditoría nunca debe romper la operación de negocio que la originó.
func (h *AuthHandlers) recordAudit(r *http.Request, e audit.Event) {
	if h.Audit == nil {
		return
	}
	e.RequestID = middleware.GetReqID(r.Context())
	e.IP = clientIP(r)
	e.UserAgent = r.UserAgent()
	if err := h.Audit.Record(r.Context(), e); err != nil {
		slog.Debug("auditoría", "error", err)
	}
}
