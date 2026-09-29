package comunicaciones

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateUsesActiveUnitCount(t *testing.T) {
	f := &fakeQueryer{units: []db.Unidade{{Estado: "activa"}, {Estado: "inactiva"}, {Estado: "activa"}}}
	got, err := Create(context.Background(), f, "11111111-1111-4111-8111-111111111111", Input{Titulo: "Aviso", Cuerpo: "Texto"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Destinatarios != 2 {
		t.Fatalf("Destinatarios = %d, want 2", got.Destinatarios)
	}
}

func TestPublishIsIdempotent(t *testing.T) {
	id := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	f := &fakeQueryer{comunicados: []db.Comunicado{{ID: pgtype.UUID{Bytes: id, Valid: true}, Titulo: "Aviso", Estado: "borrador", Destinatarios: 2}}}
	got, err := Publish(context.Background(), f, id.String(), "k-1")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if got.Estado != "publicado" {
		t.Fatalf("Estado = %q, want publicado", got.Estado)
	}
	again, err := Publish(context.Background(), f, id.String(), "k-1")
	if err != nil {
		t.Fatalf("Publish() second call error = %v", err)
	}
	if again != got {
		t.Fatalf("idempotency mismatch: got %+v want %+v", again, got)
	}
}

func TestPublishRejectsConflictingIdempotency(t *testing.T) {
	f := &fakeQueryer{existingIdem: db.IdempotencyKey{RequestHash: requestHash("a", "k"), ResponseJson: []byte(`{"id":"a"}`)}}
	_, err := Publish(context.Background(), f, "b", "k")
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
}

type fakeQueryer struct {
	units        []db.Unidade
	comunicados  []db.Comunicado
	inserted     []db.InsertOutboxEventParams
	idemInserted []db.InsertIdempotencyKeyParams
	idemUpdated  []db.UpdateIdempotencyKeyParams
	existingIdem db.IdempotencyKey
}

func (f *fakeQueryer) ListUnidades(_ context.Context, _ db.ListUnidadesParams) ([]db.Unidade, error) {
	return f.units, nil
}

func (f *fakeQueryer) ListComunicados(_ context.Context, _ db.ListComunicadosParams) ([]db.Comunicado, error) {
	return f.comunicados, nil
}

func (f *fakeQueryer) GetComunicado(_ context.Context, id pgtype.UUID) (db.Comunicado, error) {
	for _, c := range f.comunicados {
		if c.ID == id {
			return c, nil
		}
	}
	return db.Comunicado{}, errors.New("not found")
}

func (f *fakeQueryer) CreateComunicado(_ context.Context, arg db.CreateComunicadoParams) (db.Comunicado, error) {
	c := db.Comunicado{ID: pgtype.UUID{Bytes: uuid.MustParse("33333333-3333-4333-8333-333333333333"), Valid: true}, Titulo: arg.Titulo, Estado: "borrador", Destinatarios: arg.Destinatarios}
	f.comunicados = append(f.comunicados, c)
	return c, nil
}

func (f *fakeQueryer) PublishComunicado(_ context.Context, _ db.PublishComunicadoParams) (db.Comunicado, error) {
	if len(f.comunicados) == 0 {
		return db.Comunicado{}, errors.New("not found")
	}
	f.comunicados[0].Estado = "publicado"
	f.comunicados[0].PublicadoAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return f.comunicados[0], nil
}

func (f *fakeQueryer) InsertOutboxEvent(_ context.Context, arg db.InsertOutboxEventParams) (db.OutboxEvent, error) {
	f.inserted = append(f.inserted, arg)
	return db.OutboxEvent{}, nil
}

func (f *fakeQueryer) GetIdempotencyKey(_ context.Context, _ db.GetIdempotencyKeyParams) (db.IdempotencyKey, error) {
	if f.existingIdem.RequestHash != "" {
		return f.existingIdem, nil
	}
	return db.IdempotencyKey{}, errors.New("not found")
}

func (f *fakeQueryer) InsertIdempotencyKey(_ context.Context, arg db.InsertIdempotencyKeyParams) (int64, error) {
	f.idemInserted = append(f.idemInserted, arg)
	if f.existingIdem.RequestHash != "" {
		return 0, nil
	}
	return 1, nil
}

func (f *fakeQueryer) UpdateIdempotencyKey(_ context.Context, arg db.UpdateIdempotencyKeyParams) error {
	f.idemUpdated = append(f.idemUpdated, arg)
	return nil
}
