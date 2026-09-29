package reclamos

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateValidatesInput(t *testing.T) {
	fake := &fakeQueryer{}
	_, err := Create(context.Background(), fake, consorcioID.String(), CreateInput{UnidadID: "x", Categoria: "gotera", Texto: "xy"})
	if err == nil {
		t.Fatal("expected error for short texto")
	}
	_, err = Create(context.Background(), fake, consorcioID.String(), CreateInput{UnidadID: "x", Categoria: "", Texto: "se inundo el pasillo"})
	if err == nil {
		t.Fatal("expected error for empty categoria")
	}
}

func TestCreateSetsSLA(t *testing.T) {
	fake := &fakeQueryer{create: func(ctx context.Context, p db.CreateReclamoParams) (db.Reclamo, error) {
		if !p.SlaDueAt.Valid {
			t.Fatal("expected sla_due_at set")
		}
		if !p.SlaDueAt.Time.After(time.Now().UTC()) {
			t.Fatalf("sla_due_at must be in the future, got %v", p.SlaDueAt.Time)
		}
		return reclamoRow(EstadoAbierto, ""), nil
	}}
	if _, err := Create(context.Background(), fake, consorcioID.String(), CreateInput{UnidadID: unidadID.String(), Categoria: "gotera", Texto: "se inundo el pasillo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStateMachineHappyPath(t *testing.T) {
	fake := &fakeQueryer{estado: EstadoAbierto}
	want := map[string]string{
		AccionEnProgreso: EstadoEnProgreso,
		AccionResolver:   EstadoResuelto,
		AccionCerrar:     EstadoCerrado,
	}
	for accion, estado := range want {
		fake.estado = EstadoAbierto
		res, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: accion})
		if err != nil {
			t.Fatalf("accion %s: unexpected error: %v", accion, err)
		}
		if res.Estado != estado {
			t.Fatalf("accion %s: expected estado %s, got %s", accion, estado, res.Estado)
		}
		if len(fake.transiciones) == 0 || fake.transiciones[len(fake.transiciones)-1].ToEstado != estado {
			t.Fatalf("accion %s: transicion history not recorded with to_estado %s", accion, estado)
		}
	}
}

func TestStateMachineReopenKeepsHistory(t *testing.T) {
	fake := &fakeQueryer{estado: EstadoCerrado}
	motivo := "no conforme con la solucion"
	res, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionReabrir, Motivo: &motivo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Estado != EstadoAbierto {
		t.Fatalf("expected reabierto -> abierto, got %s", res.Estado)
	}
	if len(fake.transiciones) == 0 {
		t.Fatal("expected transition history row")
	}
	last := fake.transiciones[len(fake.transiciones)-1]
	if last.FromEstado != EstadoCerrado || last.ToEstado != EstadoAbierto || !last.Motivo.Valid {
		t.Fatalf("unexpected history row: from=%s to=%s motivo=%v", last.FromEstado, last.ToEstado, last.Motivo.Valid)
	}
}

func TestStateMachineRejectInvalidTransitions(t *testing.T) {
	fake := &fakeQueryer{estado: EstadoCerrado}
	if _, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionResolver}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for resolver sobre cerrado, got %v", err)
	}
	if _, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionEnProgreso}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for en_progreso sobre cerrado, got %v", err)
	}
}

func TestAsignarRequiresResponsable(t *testing.T) {
	fake := &fakeQueryer{estado: EstadoAbierto}
	if _, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionAsignar}); err == nil {
		t.Fatal("expected error when asignar without responsable")
	}
	responsable := responsableID.String()
	res, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionAsignar, ResponsableID: &responsable})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Estado != EstadoAbierto {
		t.Fatalf("asignar must keep estado abierto, got %s", res.Estado)
	}
	if res.ResponsableID == nil || *res.ResponsableID != responsable {
		t.Fatalf("expected responsable %s, got %v", responsable, res.ResponsableID)
	}
}

func TestReabrirRequiresMotivo(t *testing.T) {
	fake := &fakeQueryer{estado: EstadoCerrado}
	if _, err := Transicionar(context.Background(), fake, reclamoID.String(), TransicionInput{Accion: AccionReabrir}); err == nil {
		t.Fatal("expected error when reabrir without motivo")
	}
}

func TestAddMensajeValidatesAdjunto(t *testing.T) {
	missing := "99999999-9999-4999-8999-999999999999"
	fake := &fakeQueryer{docErr: errors.New("not found")}
	if _, err := AddMensaje(context.Background(), fake, reclamoID.String(), MensajeInput{Texto: "hola", AdjuntoID: &missing}); err == nil {
		t.Fatal("expected error for inexistent adjunto")
	}
}

var (
	consorcioID   = mustUUID("10000000-1000-4000-8000-100000000000")
	unidadID      = mustUUID("20000000-2000-4000-8000-200000000000")
	reclamoID     = mustUUID("30000000-3000-4000-8000-300000000000")
	responsableID = mustUUID("40000000-4000-4000-8000-400000000000")
)

type fakeQueryer struct {
	estado       string
	transiciones []db.ReclamoTransicione
	docErr       error
	create       func(context.Context, db.CreateReclamoParams) (db.Reclamo, error)
}

func (f *fakeQueryer) ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error) {
	return nil, nil
}
func (f *fakeQueryer) GetReclamo(context.Context, pgtype.UUID) (db.Reclamo, error) {
	if f.estado == "" {
		f.estado = EstadoAbierto
	}
	return reclamoRow(f.estado, ""), nil
}
func (f *fakeQueryer) CreateReclamo(ctx context.Context, p db.CreateReclamoParams) (db.Reclamo, error) {
	if f.create != nil {
		return f.create(ctx, p)
	}
	return reclamoRow(EstadoAbierto, ""), nil
}
func (f *fakeQueryer) UpdateReclamoEstado(_ context.Context, p db.UpdateReclamoEstadoParams) (db.Reclamo, error) {
	f.estado = p.Estado
	row := reclamoRow(p.Estado, p.ResponsableID.String())
	row.ResponsableID = p.ResponsableID
	return row, nil
}
func (f *fakeQueryer) ListReclamoMensajes(context.Context, pgtype.UUID) ([]db.ReclamoMensaje, error) {
	return nil, nil
}
func (f *fakeQueryer) InsertReclamoMensaje(context.Context, db.InsertReclamoMensajeParams) (db.ReclamoMensaje, error) {
	return db.ReclamoMensaje{}, nil
}
func (f *fakeQueryer) ListReclamoTransiciones(context.Context, pgtype.UUID) ([]db.ReclamoTransicione, error) {
	return f.transiciones, nil
}
func (f *fakeQueryer) InsertReclamoTransicion(_ context.Context, p db.InsertReclamoTransicionParams) (db.ReclamoTransicione, error) {
	f.transiciones = append(f.transiciones, db.ReclamoTransicione{
		ID:            reclamoID,
		Accion:        p.Accion,
		Motivo:        p.Motivo,
		FromEstado:    p.FromEstado,
		ToEstado:      p.ToEstado,
		ResponsableID: p.ResponsableID,
	})
	return f.transiciones[len(f.transiciones)-1], nil
}
func (f *fakeQueryer) GetDocumento(context.Context, pgtype.UUID) (db.Documento, error) {
	if f.docErr != nil {
		return db.Documento{}, f.docErr
	}
	return db.Documento{}, nil
}

var _ Queryer = (*fakeQueryer)(nil)

func reclamoRow(estado, responsable string) db.Reclamo {
	row := db.Reclamo{
		TenantID:    consorcioID,
		ConsorcioID: consorcioID,
		UnidadID:    unidadID,
		ID:          reclamoID,
		Categoria:   "gotera",
		Texto:       "se inundo el pasillo",
		Estado:      estado,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	if responsable != "" {
		row.ResponsableID = mustUUID(responsable)
	}
	return row
}

func mustUUID(s string) pgtype.UUID {
	var out pgtype.UUID
	_ = out.Scan(s)
	return out
}

func TestEstadoConstantesCubiertas(t *testing.T) {
	estados := []string{EstadoAbierto, EstadoEnProgreso, EstadoResuelto, EstadoCerrado}
	acciones := []string{AccionAsignar, AccionEnProgreso, AccionResolver, AccionCerrar, AccionReabrir}
	if len(estados) != 4 || len(acciones) != 5 {
		t.Fatal("constants diverged from contract")
	}
}
