package db

import (
	"strings"
	"testing"
)

// Prueba negativa de aislamiento: las queries del dominio reclamos (H5.3)
// deben estar acotadas por el tenant del contexto de RLS. No es una prueba de
// integración contra Postgres: es una guarda estática sobre el SQL generado.
func TestReclamosQueriesAreTenantScoped(t *testing.T) {
	queries := []string{
		listReclamos,
		getReclamo,
		createReclamo,
		updateReclamoEstado,
		listReclamoMensajes,
		insertReclamoMensaje,
		listReclamoTransiciones,
		insertReclamoTransicion,
	}
	for _, q := range queries {
		if !strings.Contains(q, "app.current_tenant_id()") {
			t.Errorf("query must be tenant-scoped (missing app.current_tenant_id()):\n%s", q)
		}
	}
}
