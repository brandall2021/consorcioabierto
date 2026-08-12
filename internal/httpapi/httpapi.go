// Package httpapi provee helpers HTTP: JSON y errores RFC 9457
// (application/problem+json, convenciones §7.1 de la especificación).
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// WriteJSON serializa v como JSON con el status dado.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// FieldError describe un error por campo (p. ej. validación).
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// Problem es la representación de error RFC 9457.
type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Code      string       `json:"code"`
	Detail    string       `json:"detail"`
	Instance  string       `json:"instance"`
	RequestID string       `json:"request_id"`
	Errors    []FieldError `json:"errors,omitempty"`
}

// WriteProblem emite un error RFC 9457 con request_id tomado del contexto.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string, errors []FieldError) {
	WriteJSON(w, status, Problem{
		Type:      "https://api.consorcioabierto.local/problems/" + code,
		Title:     title,
		Status:    status,
		Code:      code,
		Detail:    detail,
		Instance:  r.URL.Path,
		RequestID: middleware.GetReqID(r.Context()),
		Errors:    errors,
	})
}
