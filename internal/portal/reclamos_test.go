package portal

import (
	"context"
	"errors"
	"testing"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/reclamos"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	miUnidad     = "aaaa1111-1111-4111-8111-111111111111"
	unidadAjena  = "bbbb2222-2222-4222-8222-222222222222"
	miConsorcio  = "11111111-1111-4111-8111-111111111111"
	otroConsorrc = "99999999-9999-4999-8999-999999999999"
)

type unidadesFake struct {
	units []db.Unidade
	err   error
}

func (f *unidadesFake) ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error) {
	return f.units, f.err
}

type reclamosSpy struct {
	creado *db.CreateReclamoParams
}

func (f *reclamosSpy) ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error) {
	return nil, nil
}
func (f *reclamosSpy) GetReclamo(context.Context, pgtype.UUID) (db.Reclamo, error) {
	return db.Reclamo{}, nil
}
func (f *reclamosSpy) CreateReclamo(_ context.Context, p db.CreateReclamoParams) (db.Reclamo, error) {
	f.creado = &p
	return db.Reclamo{
		ID:          mustUUID("cccccccc-3333-4333-8333-333333333333"),
		UnidadID:    p.UnidadID,
		ConsorcioID: p.ConsorcioID,
		Categoria:   p.Categoria,
		Estado:      "abierto",
	}, nil
}
func (f *reclamosSpy) UpdateReclamoEstado(context.Context, db.UpdateReclamoEstadoParams) (db.Reclamo, error) {
	return db.Reclamo{}, nil
}
func (f *reclamosSpy) ListReclamoMensajes(context.Context, pgtype.UUID) ([]db.ReclamoMensaje, error) {
	return nil, nil
}
func (f *reclamosSpy) InsertReclamoMensaje(context.Context, db.InsertReclamoMensajeParams) (db.ReclamoMensaje, error) {
	return db.ReclamoMensaje{}, nil
}
func (f *reclamosSpy) ListReclamoTransiciones(context.Context, pgtype.UUID) ([]db.ReclamoTransicione, error) {
	return nil, nil
}
func (f *reclamosSpy) InsertReclamoTransicion(context.Context, db.InsertReclamoTransicionParams) (db.ReclamoTransicione, error) {
	return db.ReclamoTransicione{}, nil
}
func (f *reclamosSpy) GetDocumento(context.Context, pgtype.UUID) (db.Documento, error) {
	return db.Documento{}, nil
}

var _ reclamos.Queryer = (*reclamosSpy)(nil)

func unidadesDe(mis ...string) *unidadesFake {
	f := &unidadesFake{}
	for _, id := range mis {
		f.units = append(f.units, db.Unidade{
			ID:          mustUUID(id),
			ConsorcioID: mustUUID(miConsorcio),
			Codigo:      "1A",
			Estado:      "activo",
		})
	}
	return f
}

func TestCrearReclamoUsaElConsorcioDeLaUnidadResuelta(t *testing.T) {
	spy := &reclamosSpy{}
	units := unidadesDe(miUnidad)
	// La unidad propia pertenece a otro consorcio que el que el cliente
	// pretenderia: el servidor debe ganar.
	units.units[0].ConsorcioID = mustUUID(otroConsorrc)

	out, err := CrearReclamo(context.Background(), units, spy, reclamos.CreateInput{
		UnidadID:  miUnidad,
		Categoria: "plomería",
		Texto:     "Fuga bajo la mesada",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spy.creado == nil {
		t.Fatal("no se creó el reclamo")
	}
	if got := spy.creado.ConsorcioID.String(); got != otroConsorrc {
		t.Fatalf("consorcio no derivado de la unidad: %s", got)
	}
	if out.Categoria != "plomería" || out.Estado != "abierto" {
		t.Fatalf("reclamo inesperado: %+v", out)
	}
}

func TestCrearReclamoRechazaUnidadAjena(t *testing.T) {
	spy := &reclamosSpy{}
	_, err := CrearReclamo(context.Background(), unidadesDe(miUnidad), spy, reclamos.CreateInput{
		UnidadID:  unidadAjena,
		Categoria: "ascensor",
		Texto:     "Ascensor detenido",
	})
	if !errors.Is(err, ErrSinVinculo) {
		t.Fatalf("se esperaba ErrSinVinculo, hubo %v", err)
	}
	if spy.creado != nil {
		t.Fatal("se creó un reclamo pese a la unidad ajena")
	}
}

func TestCrearReclamoRechazaSinVinculo(t *testing.T) {
	spy := &reclamosSpy{}
	_, err := CrearReclamo(context.Background(), unidadesDe(), spy, reclamos.CreateInput{
		UnidadID:  miUnidad,
		Categoria: "plomería",
		Texto:     "Sin vínculo alguno",
	})
	if !errors.Is(err, ErrSinVinculo) {
		t.Fatalf("se esperaba ErrSinVinculo, hubo %v", err)
	}
	if spy.creado != nil {
		t.Fatal("se creó un reclamo sin vínculo")
	}
}

func TestCrearReclamoRechazaUnidadInvalida(t *testing.T) {
	for _, unidadID := range []string{"", "   ", "no-es-uuid"} {
		spy := &reclamosSpy{}
		_, err := CrearReclamo(context.Background(), unidadesDe(miUnidad), spy, reclamos.CreateInput{
			UnidadID:  unidadID,
			Categoria: "plomería",
			Texto:     "Unidad inválida",
		})
		if !errors.Is(err, ErrSinVinculo) {
			t.Fatalf("unidad %q: se esperaba ErrSinVinculo, hubo %v", unidadID, err)
		}
		if spy.creado != nil {
			t.Fatalf("unidad %q: se creó un reclamo inválido", unidadID)
		}
	}
}
