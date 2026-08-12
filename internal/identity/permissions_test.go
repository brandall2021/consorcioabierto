package identity

import "testing"

func TestPermissionsForRolesTenantAdmin(t *testing.T) {
	perms := PermissionsForRoles([]string{"tenant_admin"})
	want := map[string]bool{
		"tenant.users.read": false, "consorcios.manage": false,
		"expensas.publish": false, "finanzas.reconcile": false,
	}
	for _, p := range perms {
		want[p] = true
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("tenant_admin debería tener %s", k)
		}
	}
}

func TestPermissionsConsorcistaLimitado(t *testing.T) {
	perms := PermissionsForRoles([]string{"consorcista"})
	for _, p := range perms {
		switch p {
		case "finanzas.read", "gastos.read", "consorcios.manage":
			t.Errorf("consorcista no debería tener %s", p)
		}
	}
}

func TestPermissionsUnionDeRoles(t *testing.T) {
	perms := PermissionsForRoles([]string{"auditor", "tesorero"})
	set := map[string]bool{}
	for _, p := range perms {
		set[p] = true
	}
	if !set["finanzas.reconcile"] {
		t.Error("tesorero debería aportar finanzas.reconcile")
	}
	if !set["auditoria.read"] {
		t.Error("auditor debería aportar auditoria.read")
	}
}
