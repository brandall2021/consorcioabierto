package identity

// Permisos por rol según docs/permission-matrix.md. La autorización final se
// resuelve en backend con permiso + tenant + scope ([ADR-0009]).
var rolePermissions = map[string][]string{
	"platform_admin": {
		"tenants.read", "tenants.manage", "plan_codes.manage",
	},
	"tenant_admin": {
		"tenant.users.read", "tenant.users.manage",
		"consorcios.read", "consorcios.manage",
		"ufs.read", "ufs.manage",
		"expensas.read", "expensas.create", "expensas.confirm", "expensas.publish",
		"gastos.read", "gastos.manage",
		"cobranzas.read", "cobranzas.manage",
		"finanzas.read", "finanzas.reconcile",
		"proveedores.read", "proveedores.manage",
		"documentos.read", "documentos.manage",
		"comunicaciones.read", "comunicaciones.send",
		"reclamos.read", "reclamos.manage",
		"auditoria.read",
	},
	"consorcio_admin": {
		"consorcios.read", "consorcios.manage",
		"ufs.read", "ufs.manage",
		"expensas.read", "expensas.create", "expensas.confirm", "expensas.publish",
		"gastos.read", "gastos.manage",
		"cobranzas.read", "cobranzas.manage",
		"finanzas.read", "finanzas.reconcile",
		"proveedores.read", "proveedores.manage",
		"documentos.read", "documentos.manage",
		"comunicaciones.read", "comunicaciones.send",
		"reclamos.read", "reclamos.manage",
		"auditoria.read",
	},
	"tesorero": {
		"consorcios.read",
		"ufs.read",
		"expensas.read",
		"gastos.read",
		"cobranzas.read", "cobranzas.manage",
		"finanzas.read", "finanzas.reconcile",
		"proveedores.read",
		"documentos.read",
		"comunicaciones.read",
		"reclamos.read", "reclamos.manage",
	},
	"auditor": {
		"consorcios.read",
		"ufs.read",
		"expensas.read",
		"gastos.read",
		"cobranzas.read",
		"finanzas.read",
		"proveedores.read",
		"documentos.read",
		"comunicaciones.read",
		"reclamos.read",
		"auditoria.read",
	},
	"consorcista": {
		"ufs.read", "expensas.read", "cobranzas.read",
		"documentos.read", "comunicaciones.read", "reclamos.read",
	},
}

// PermissionsForRoles devuelve la unión de permisos para los roles dados.
func PermissionsForRoles(roles []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range roles {
		for _, p := range rolePermissions[r] {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}
