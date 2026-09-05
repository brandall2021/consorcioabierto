package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/documentos"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

// CreateDocumentUploadIntent POST /document-upload-intents (documentos.manage).
// Crea la fila del documento en estado pendiente y devuelve la URL firmada de
// subida directa al bucket (§5.4, [ADR-0008]).
func (h *AuthHandlers) CreateDocumentUploadIntent(w http.ResponseWriter, r *http.Request) {
	var in documentos.CreateUploadIntentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	claims := claimsFrom(r.Context())
	if claims == nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", "", nil)
		return
	}

	svc := documentos.NewService(h.Docs, q)
	res, err := svc.CreateUploadIntent(r.Context(), claims.Tenant, in)
	if err != nil {
		h.writeDocError(w, r, err)
		return
	}

	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "documentos.create",
		RecursoType: "documento",
		RecursoID:   res.DocumentoID,
		Diff:        map[string]any{"tipo": in.Tipo, "nombre": in.Nombre, "size_bytes": in.SizeBytes},
	}))

	httpapi.WriteJSON(w, http.StatusCreated, res)
}

// GetDocumentDownloadUrl GET /documentos/{id}/download-url (documentos.read).
// Autoriza el recurso, escanea (lazy) si está pendiente y devuelve la URL
// firmada breve (§5.4: "nunca aceptar una storage key del cliente").
func (h *AuthHandlers) GetDocumentDownloadUrl(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	svc := documentos.NewService(h.Docs, q)
	res, err := svc.RequestDownloadURL(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeDocError(w, r, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, res)
}

func (h *AuthHandlers) writeDocError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, documentos.ErrDocumentoInvalid):
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", err.Error(), nil)
	case errors.Is(err, documentos.ErrDocumentoTooLarge):
		httpapi.WriteProblem(w, r, http.StatusRequestEntityTooLarge, "entity_too_large", "Documento demasiado grande", err.Error(), nil)
	case errors.Is(err, documentos.ErrDocumentoNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Documento no encontrado", err.Error(), nil)
	case errors.Is(err, documentos.ErrDocumentoCuarentena):
		httpapi.WriteProblem(w, r, http.StatusGone, "gone", "Documento en cuarentena", err.Error(), nil)
	case errors.Is(err, documentos.ErrStorageUnavailable):
		httpapi.WriteProblem(w, r, http.StatusServiceUnavailable, "storage_unavailable", "Storage no disponible", err.Error(), nil)
	default:
		slog.Error("documentos", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}
