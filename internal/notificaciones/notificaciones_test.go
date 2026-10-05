package notificaciones

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
)

// dobles: el dominio se prueba sin Postgres, igual que internal/personas.
type fake struct {
	destinatarios    []pgtype.UUID
	insertados       []db.InsertNotificacionParams
	insertErr        error
	listados         []db.Notificacion
	listadoParams    db.ListNotificacionesForCurrentUserParams
	noLeidas         int64
	marcarErr        error
	listErr          error
	destinatariosErr error
}

func (f *fake) ListDestinatariosNotificacion(context.Context, pgtype.UUID) ([]pgtype.UUID, error) {
	return f.destinatarios, f.destinatariosErr
}

func (f *fake) InsertNotificacion(_ context.Context, arg db.InsertNotificacionParams) (db.Notificacion, error) {
	f.insertados = append(f.insertados, arg)
	if f.insertErr != nil {
		return db.Notificacion{}, f.insertErr
	}
	return notif(), nil
}

func (f *fake) ListNotificacionesForCurrentUser(_ context.Context, arg db.ListNotificacionesForCurrentUserParams) ([]db.Notificacion, error) {
	f.listadoParams = arg
	return f.listados, f.listErr
}

func (f *fake) MarcarNotificacionLeida(context.Context, pgtype.UUID) (db.Notificacion, error) {
	if f.marcarErr != nil {
		return db.Notificacion{}, f.marcarErr
	}
	n := notif()
	n.LeidaAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return n, nil
}

func (f *fake) CountNotificacionesNoLeidas(context.Context) (int64, error) {
	return f.noLeidas, nil
}

func pgid() pgtype.UUID {
	return pgtype.UUID{Bytes: uuid.New(), Valid: true}
}

func notif() db.Notificacion {
	return db.Notificacion{
		TenantID:    pgtype.UUID{Bytes: uuid.New(), Valid: true},
		ID:          pgtype.UUID{Bytes: uuid.MustParse("11111111-1111-4111-8111-111111111111"), Valid: true},
		UserID:      pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Tipo:        TipoComunicado,
		Titulo:      "Corte de agua",
		Cuerpo:      "De 9 a 13 hs.",
		RecursoType: RecursoComunicado,
		CreatedAt:   pgtype.Timestamptz{Time: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), Valid: true},
	}
}

func TestNotificarComunicadoCreaUnaPorDestinatario(t *testing.T) {
	f := &fake{destinatarios: []pgtype.UUID{pgid(), pgid(), pgid()}}
	res, err := NotificarComunicado(context.Background(), f, uuid.NewString(), uuid.NewString(), "Corte de agua", "De 9 a 13 hs.")
	if err != nil {
		t.Fatalf("no error: %v", err)
	}
	if res.Destinatarios != 3 || res.Creadas != 3 || res.Omitidas != 0 {
		t.Fatalf("esperaba 3/3/0, obtuve %+v", res)
	}
	for _, ins := range f.insertados {
		if ins.Tipo != TipoComunicado || ins.RecursoType != RecursoComunicado {
			t.Fatalf("tipo/recurso inesperados: %+v", ins)
		}
		if ins.Titulo != "Corte de agua" {
			t.Fatalf("titulo no propagado: %q", ins.Titulo)
		}
		if !ins.UserID.Valid {
			t.Fatal("user_id sin resolver")
		}
	}
}

func TestNotificarComunicadoEsIdempotenteEnElReintento(t *testing.T) {
	// El worker reintenta los eventos fallidos; un duplicado por conflicto del
	// indice unico no es error y no debe sumar Creadas.
	f := &fake{destinatarios: []pgtype.UUID{pgid(), pgid()}, insertErr: pgx.ErrNoRows}
	res, err := NotificarComunicado(context.Background(), f, uuid.NewString(), uuid.NewString(), "Titulo", "")
	if err != nil {
		t.Fatalf("un duplicado no debe ser error: %v", err)
	}
	if res.Creadas != 0 || res.Omitidas != 2 {
		t.Fatalf("esperaba 0 creadas / 2 omitidas, obtuve %+v", res)
	}
}

func TestNotificarComunicadoSinDestinatarios(t *testing.T) {
	f := &fake{destinatarios: []pgtype.UUID{}}
	res, err := NotificarComunicado(context.Background(), f, uuid.NewString(), uuid.NewString(), "Titulo", "")
	if err != nil {
		t.Fatalf("sin destinatarios no hay error: %v", err)
	}
	if res.Destinatarios != 0 || res.Creadas != 0 || len(f.insertados) != 0 {
		t.Fatalf("no debia insertar nada: %+v", res)
	}
}

func TestNotificarComunicadoRechazaUUIDInvalido(t *testing.T) {
	for _, caso := range []struct{ consorcio, comunicado string }{
		{"no-es-uuid", uuid.NewString()},
		{uuid.NewString(), "no-es-uuid"},
	} {
		f := &fake{}
		if _, err := NotificarComunicado(context.Background(), f, caso.consorcio, caso.comunicado, "T", ""); !errors.Is(err, ErrLimiteInvalido) {
			t.Fatalf("consorcio=%q comunicado=%q esperaba ErrLimiteInvalido, obtuve %v", caso.consorcio, caso.comunicado, err)
		}
		if len(f.insertados) != 0 {
			t.Fatal("no debe insertar con entrada invalida")
		}
	}
}

func TestNotificarComunicadoRechazaTituloVacio(t *testing.T) {
	f := &fake{}
	if _, err := NotificarComunicado(context.Background(), f, uuid.NewString(), uuid.NewString(), "   ", ""); !errors.Is(err, ErrLimiteInvalido) {
		t.Fatalf("esperaba ErrLimiteInvalido, obtuve %v", err)
	}
}

func TestListarAcotaElLimiteYDevuelveElContador(t *testing.T) {
	f := &fake{listados: []db.Notificacion{notif()}, noLeidas: 3}
	out, err := Listar(context.Background(), f, true, 10)
	if err != nil {
		t.Fatalf("no error: %v", err)
	}
	if f.listadoParams.Limite != 10 || !f.listadoParams.SoloNoLeidas {
		t.Fatalf("params no propagados: %+v", f.listadoParams)
	}
	if out.NoLeidas != 3 {
		t.Fatalf("esperaba 3 no leidas, obtuve %d", out.NoLeidas)
	}
	if len(out.Data) != 1 || out.Data[0].LeidaAt != nil {
		t.Fatalf("no leida deberia venir sin leida_at: %+v", out.Data[0])
	}
}

func TestListarAplicaLimitesPorDefectoYMximo(t *testing.T) {
	f := &fake{}
	if _, err := Listar(context.Background(), f, false, 0); err != nil {
		t.Fatal(err)
	}
	if f.listadoParams.Limite != limitePorDefecto {
		t.Fatalf("limite 0 deberia usar el defecto, obtuve %d", f.listadoParams.Limite)
	}
	f2 := &fake{}
	if _, err := Listar(context.Background(), f2, false, 100000); err != nil {
		t.Fatal(err)
	}
	if f2.listadoParams.Limite != limiteMaximo {
		t.Fatalf("limite enorme deberia recortarse, obtuve %d", f2.listadoParams.Limite)
	}
}

func TestListarDevuelveListaVaciaYNoNil(t *testing.T) {
	f := &fake{}
	out, err := Listar(context.Background(), f, false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if out.Data == nil {
		t.Fatal("la lista vacia debe ir inicializada para que el JSON sea [] y no null")
	}
}

func TestMarcarLeida(t *testing.T) {
	f := &fake{}
	dto, err := MarcarLeida(context.Background(), f, "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("no error: %v", err)
	}
	if dto.LeidaAt == nil {
		t.Fatal("deberia traer leida_at")
	}
}

func TestMarcarLeidaIdNotFoundODeOtroUsuario(t *testing.T) {
	// La query filtra por app.current_user_id(); otra notificacion, de otro
	// usuario o ya leida devuelven cero filas y 404 indistinguible.
	f := &fake{marcarErr: pgx.ErrNoRows}
	if _, err := MarcarLeida(context.Background(), f, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotificacionNotFound) {
		t.Fatalf("esperaba ErrNotificacionNotFound, obtuve %v", err)
	}
}

func TestMarcarLeidaRechazaUUIDInvalido(t *testing.T) {
	f := &fake{}
	if _, err := MarcarLeida(context.Background(), f, "xyz"); !errors.Is(err, ErrLimiteInvalido) {
		t.Fatalf("esperaba ErrLimiteInvalido, obtuve %v", err)
	}
}
