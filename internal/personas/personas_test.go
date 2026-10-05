package personas

import (
	"context"
	"errors"
	"testing"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	personaA     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	usuarioA     = "11111111-1111-4111-8111-111111111111"
	usuarioB     = "22222222-2222-4222-8222-222222222222"
	usuarioAjeno = "99999999-9999-4999-8999-999999999999"
)

type fakeQueryer struct {
	members       []db.ListTenantMembersRow
	personas      map[pgtype.UUID]db.PersonaVinculable
	setErr        error
	setCalls      int
	unsetCalls    int
	setLastParams db.SetPersonaUsuarioParams
}

func (f *fakeQueryer) ListTenantMembers(context.Context) ([]db.ListTenantMembersRow, error) {
	return f.members, nil
}

func (f *fakeQueryer) GetPersonaForVinculo(_ context.Context, id pgtype.UUID) (db.PersonaVinculable, error) {
	p, ok := f.personas[id]
	if !ok {
		return db.PersonaVinculable{}, pgx.ErrNoRows
	}
	return p, nil
}

func (f *fakeQueryer) SetPersonaUsuario(_ context.Context, p db.SetPersonaUsuarioParams) error {
	f.setCalls++
	f.setLastParams = p
	return f.setErr
}

func (f *fakeQueryer) UnsetPersonaUsuario(_ context.Context, id pgtype.UUID) error {
	f.unsetCalls++
	p := f.personas[id]
	p.UserID = pgtype.UUID{}
	f.personas[id] = p
	return nil
}

func personaFixture(id string, userID pgtype.UUID) db.PersonaVinculable {
	return db.PersonaVinculable{
		ID:       mustUUID(id),
		TenantID: mustUUID("22222222-2222-4222-8222-222222222222"),
		Nombre:   "María Test",
		UserID:   userID,
	}
}

func membersFixture() []db.ListTenantMembersRow {
	return []db.ListTenantMembersRow{
		{UserID: mustUUID(usuarioA), TenantID: mustUUID("22222222-2222-4222-8222-222222222222"),
			MembershipStatus: "active", EmailNormalized: "a@tenant-a.com", Name: "Ana", UserStatus: "active"},
		{UserID: mustUUID(usuarioB), TenantID: mustUUID("22222222-2222-4222-8222-222222222222"),
			MembershipStatus: "active", EmailNormalized: "b@tenant-a.com", Name: "Beto", UserStatus: "active"},
	}
}

func newFake() *fakeQueryer {
	p := personaFixture(personaA, pgtype.UUID{})
	return &fakeQueryer{members: membersFixture(), personas: map[pgtype.UUID]db.PersonaVinculable{mustUUID(personaA): p}}
}

func mustUUID(s string) pgtype.UUID {
	var out pgtype.UUID
	_ = out.Scan(s)
	return out
}

func TestVincularPersisteElUsuarioElegido(t *testing.T) {
	f := newFake()

	out, err := Vincular(context.Background(), f, personaA, usuarioA)
	if err != nil {
		t.Fatalf("Vincular: %v", err)
	}
	if f.setCalls != 1 {
		t.Fatalf("setCalls = %d, quer 1", f.setCalls)
	}
	if f.setLastParams.UserID != mustUUID(usuarioA) {
		t.Errorf("user_id persistido = %v, quer %s", f.setLastParams.UserID, usuarioA)
	}
	if out.UserID == nil || *out.UserID != usuarioA {
		t.Errorf("DTO user_id = %v, quer %s", out.UserID, usuarioA)
	}
}

// El cliente elige el usuario, nunca el tenant: si el usuario no es miembro
// del tenant activo, la operacion falla sin escribir nada.
func TestVincularRechazaUsuarioDeOtroTenant(t *testing.T) {
	f := newFake()

	_, err := Vincular(context.Background(), f, personaA, usuarioAjeno)
	if !errors.Is(err, ErrUsuarioNotFound) {
		t.Fatalf("err = %v, quer ErrUsuarioNotFound", err)
	}
	if f.setCalls != 0 {
		t.Errorf("setCalls = %d: escribio pese al rechazo", f.setCalls)
	}
}

// Una persona de otro tenant es indistinguible de una inexistente (404).
func TestVincularRechazaPersonaAjena(t *testing.T) {
	f := newFake()
	f.personas = map[pgtype.UUID]db.PersonaVinculable{}

	_, err := Vincular(context.Background(), f, personaA, usuarioA)
	if !errors.Is(err, ErrPersonaNotFound) {
		t.Fatalf("err = %v, quer ErrPersonaNotFound", err)
	}
	if f.setCalls != 0 {
		t.Errorf("setCalls = %d: escribio pese al rechazo", f.setCalls)
	}
}

func TestVincularRespetaIndiceUnico(t *testing.T) {
	f := newFake()
	f.setErr = &pgconn.PgError{Code: "23505"}

	_, err := Vincular(context.Background(), f, personaA, usuarioB)
	if !errors.Is(err, ErrUsuarioYaVinculado) {
		t.Fatalf("err = %v, quer ErrUsuarioYaVinculado", err)
	}
}

func TestVincularRechazaUUIDInvalido(t *testing.T) {
	f := newFake()

	for _, tc := range []struct{ nombre, persona, usuario string }{
		{"persona no uuid", "no-es-uuid", usuarioA},
		{"usuario no uuid", personaA, "no-es-uuid"},
		{"ambos vacios", "", ""},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			_, err := Vincular(context.Background(), f, tc.persona, tc.usuario)
			if !errors.Is(err, ErrVinculoInvalido) {
				t.Fatalf("err = %v, quer ErrVinculoInvalido", err)
			}
			if f.setCalls != 0 {
				t.Errorf("setCalls = %d: escribio pese al rechazo", f.setCalls)
			}
		})
	}
}

func TestDesvincularLimpiaElUsuarioYEsIdempotente(t *testing.T) {
	f := newFake()
	f.personas[mustUUID(personaA)] = personaFixture(personaA, mustUUID(usuarioA))

	out, err := Desvincular(context.Background(), f, personaA)
	if err != nil {
		t.Fatalf("Desvincular: %v", err)
	}
	if out.UserID != nil {
		t.Errorf("user_id = %v, quer nil", out.UserID)
	}
	if f.unsetCalls != 1 {
		t.Errorf("unsetCalls = %d, quer 1", f.unsetCalls)
	}

	// Segunda llamada: ya estaba desvinculada, no vuelve a escribir.
	out, err = Desvincular(context.Background(), f, personaA)
	if err != nil {
		t.Fatalf("Desvincular idempotente: %v", err)
	}
	if f.unsetCalls != 1 {
		t.Errorf("unsetCalls = %d tras idempotente, quer 1", f.unsetCalls)
	}
	if out.UserID != nil {
		t.Errorf("user_id = %v, quer nil", out.UserID)
	}
}

func TestDesvincularRechazaPersonaAjena(t *testing.T) {
	f := newFake()
	f.personas = map[pgtype.UUID]db.PersonaVinculable{}

	_, err := Desvincular(context.Background(), f, personaA)
	if !errors.Is(err, ErrPersonaNotFound) {
		t.Fatalf("err = %v, quer ErrPersonaNotFound", err)
	}
	if f.unsetCalls != 0 {
		t.Errorf("unsetCalls = %d: escribio pese al rechazo", f.unsetCalls)
	}
}

func TestListarMiembrosProyectaElTenantActivo(t *testing.T) {
	f := newFake()

	out, err := ListarMiembros(context.Background(), f)
	if err != nil {
		t.Fatalf("ListarMiembros: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, quer 2", len(out))
	}
	if out[0].UserID != usuarioA || out[0].Email != "a@tenant-a.com" || out[0].Nombre != "Ana" {
		t.Errorf("miembro 0 = %+v", out[0])
	}
	for _, m := range out {
		if m.Vinculado != nil {
			t.Errorf("Vinculado = %v, se espera nil: la vista no trae esa columna", *m.Vinculado)
		}
	}
}
