package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/audit"
	"github.com/brandall2021/consorcioabierto/internal/consorcios"
	"github.com/brandall2021/consorcioabierto/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

// maxImportFileBytes limita el tamaño de la plantilla CSV (10 MB).
const maxImportFileBytes int64 = 10 << 20

// CreateImportJob POST /consorcios/{id}/unidades/import-jobs (permiso ufs.manage).
// Multipart {archivo, modo}. Valida la plantilla y crea el job en `listo`
// (preview) sin escribir UFs; el contenido se aplica recién en confirm.
func (h *AuthHandlers) CreateImportJob(w http.ResponseWriter, r *http.Request) {
	consorcioID := chi.URLParam(r, "id")
	if consorcioID == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", "consorcio_id requerido", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileBytes)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Solicitud inválida", "multipart inválido", nil)
		return
	}
	modo := r.FormValue("modo")
	if modo == "" {
		httpapi.WriteProblem(w, r, http.StatusUnprocessableEntity, "invalid_request", "Solicitud inválida", "campo modo requerido", nil)
		return
	}
	file, _, err := r.FormFile("archivo")
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnprocessableEntity, "invalid_request", "Solicitud inválida", "campo archivo requerido", nil)
		return
	}
	defer func() { _ = file.Close() }()

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	job, err := consorcios.CreateImportJob(r.Context(), q, consorcioID, modo, file)
	if err != nil {
		h.writeImportJobError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "import_jobs.create",
		RecursoType: "import_job",
		RecursoID:   job.ID,
		Diff:        map[string]any{"consorcio_id": consorcioID, "modo": modo, "total_filas": job.TotalFilas},
	}))

	httpapi.WriteJSON(w, http.StatusAccepted, job)
}

// GetImportJob GET /import-jobs/{id} (permiso ufs.read). Devuelve el preview
// (errores de validación) o el resumen final tras confirmar.
func (h *AuthHandlers) GetImportJob(w http.ResponseWriter, r *http.Request) {
	q, _, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer rollbackOnly(rollback)()

	job, err := consorcios.GetImportJob(r.Context(), q, chi.URLParam(r, "id"))
	if err != nil {
		h.writeImportJobError(w, r, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, job)
}

// ConfirmImportJob POST /import-jobs/{id}/confirm (permiso ufs.manage).
// Aplica el modo del job en transacción; idempotente con Idempotency-Key.
func (h *AuthHandlers) ConfirmImportJob(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key requerida", "confirmar importación es idempotente (ADR-0004)", nil)
		return
	}

	q, commit, rollback, err := h.txQueries(r)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Token inválido", err.Error(), nil)
		return
	}
	defer func() { _ = rollback() }()

	job, err := consorcios.ConfirmImportJob(r.Context(), q, chi.URLParam(r, "id"), idemKey)
	if err != nil {
		h.writeImportJobError(w, r, err)
		return
	}
	if err := commit(); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
		return
	}

	h.recordAudit(r, h.consorcioAuditEvent(r, audit.Event{
		Accion:      "import_jobs.confirm",
		RecursoType: "import_job",
		RecursoID:   job.ID,
		Diff: map[string]any{
			"creados":      job.Creados,
			"actualizados": job.Actualizados,
			"rechazados":   job.Rechazados,
		},
	}))

	httpapi.WriteJSON(w, http.StatusOK, job)
}

func (h *AuthHandlers) writeImportJobError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, consorcios.ErrImportInvalid):
		httpapi.WriteProblem(w, r, http.StatusUnprocessableEntity, "import_invalid", "Importación inválida", err.Error(), nil)
	case errors.Is(err, consorcios.ErrImportJobNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Job de importación no encontrado", err.Error(), nil)
	case errors.Is(err, consorcios.ErrImportJobEstado),
		errors.Is(err, consorcios.ErrIdempotencyConflict):
		httpapi.WriteProblem(w, r, http.StatusConflict, "conflict", "Conflicto", err.Error(), nil)
	case errors.Is(err, consorcios.ErrNotFound):
		httpapi.WriteProblem(w, r, http.StatusNotFound, "not_found", "Consorcio no encontrado", err.Error(), nil)
	default:
		slog.Error("import_jobs", "error", err)
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Error interno", err.Error(), nil)
	}
}