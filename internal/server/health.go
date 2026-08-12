package server

import (
	"net/http"

	"github.com/brandall2021/consorcioabierto/internal/httpapi"
)

// handleHealth expone el estado del API (liveness). El readiness contra la base
// se agrega en H1.2 cuando exista el repositorio de salud.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
