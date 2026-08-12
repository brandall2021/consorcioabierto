// Package tenancy agrupa helpers de aislamiento multi-tenant (RLS) y RBAC.
package tenancy

import (
	"context"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
)

// SetUser fija el contexto de usuario de RLS (app.set_app_user_id) para la
// transacción/conexión actual. Debe ejecutarse antes de cualquier consulta
// protegida por políticas que dependan de app.current_user_id().
func SetUser(ctx context.Context, q db.DBTX, userID string) error {
	_, err := q.Exec(ctx, "SELECT app.set_app_user_id($1)", userID)
	return err
}

// SetTenant fija el contexto de tenant de RLS (app.set_app_tenant_id).
func SetTenant(ctx context.Context, q db.DBTX, tenantID string) error {
	_, err := q.Exec(ctx, "SELECT app.set_app_tenant_id($1)", tenantID)
	return err
}

// SetContext fija usuario y tenant en una sola operación.
func SetContext(ctx context.Context, q db.DBTX, userID, tenantID string) error {
	if err := SetUser(ctx, q, userID); err != nil {
		return err
	}
	return SetTenant(ctx, q, tenantID)
}
