package documentos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var testSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCreateUploadIntentValidates(t *testing.T) {
	env := func() DocsEnv {
		return DocsEnv{Storage: NewMemoryStorage(), Scanner: NewMockScanner(), MaxUpload: 10, SignedTTL: time.Minute}
	}
	cases := []struct {
		name string
		in   CreateUploadIntentInput
		want error
	}{
		{"tipo vacío", CreateUploadIntentInput{Tipo: " ", Nombre: "a", SHA256: testSHA, SizeBytes: 5}, ErrDocumentoInvalid},
		{"nombre vacío", CreateUploadIntentInput{Tipo: "gasto", Nombre: "", SHA256: testSHA, SizeBytes: 5}, ErrDocumentoInvalid},
		{"sha256 inválido", CreateUploadIntentInput{Tipo: "gasto", Nombre: "a", SHA256: "xyz", SizeBytes: 5}, ErrDocumentoInvalid},
		{"tamaño 0", CreateUploadIntentInput{Tipo: "gasto", Nombre: "a", SHA256: testSHA, SizeBytes: 0}, ErrDocumentoInvalid},
		{"excede máx", CreateUploadIntentInput{Tipo: "gasto", Nombre: "a", SHA256: testSHA, SizeBytes: 11}, ErrDocumentoTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := NewService(env(), &fakeQuerier{})
			_, err := svc.CreateUploadIntent(context.Background(), "tenant-1", c.in)
			if !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

func TestCreateUploadIntentStoresAndPresigns(t *testing.T) {
	mem := NewMemoryStorage()
	env := DocsEnv{Storage: mem, Scanner: NewMockScanner(), MaxUpload: 10, SignedTTL: time.Minute}
	q := &fakeQuerier{}
	svc := NewService(env, q)

	res, err := svc.CreateUploadIntent(context.Background(), "tenant-1", CreateUploadIntentInput{
		Tipo: "gasto", Nombre: "factura.pdf", SHA256: testSHA, SizeBytes: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.DocumentoID == "" || res.UploadURL == "" {
		t.Fatalf("resultado incompleto: %+v", res)
	}
	if len(q.rows) != 1 {
		t.Fatalf("se esperaba 1 fila insertada, hay %d", len(q.rows))
	}
	row := q.rows[0]
	if row.Tipo != "gasto" || row.Nombre != "factura.pdf" || row.SizeBytes != 5 {
		t.Fatalf("fila mal guardada: %+v", row)
	}
	if row.StorageKey != "tenant-1/"+res.DocumentoID {
		t.Fatalf("storage key inesperada: %s", row.StorageKey)
	}
	if row.Antivirus != AntivirusPendiente {
		t.Fatalf("antivirus debería arrancar pendiente: %s", row.Antivirus)
	}
}

func TestRequestDownloadURLScansLazyAndClean(t *testing.T) {
	mem := NewMemoryStorage()
	env := DocsEnv{Storage: mem, Scanner: NewMockScanner(), MaxUpload: 10, SignedTTL: time.Minute}
	q := &fakeQuerier{}

	// Crear intent y simular la subida del cliente.
	svc := NewService(env, q)
	docID := "fb9d8a40-0000-4000-8000-000000000001"
	insert := db.InsertDocumentoParams{
		ID:         pgtype.UUID{Bytes: mustUUID(t, docID), Valid: true},
		Tipo:       "gasto",
		Nombre:     "factura.pdf",
		StorageKey: "tenant-1/" + docID,
		SizeBytes:  5,
		Sha256:     testSHA,
	}
	q.rows = append(q.rows, db.Documento{
		ID: insert.ID, Tipo: insert.Tipo, Nombre: insert.Nombre,
		StorageKey: insert.StorageKey, SizeBytes: insert.SizeBytes, Antivirus: AntivirusPendiente,
	})
	if err := mem.Put(context.Background(), insert.StorageKey, []byte("%PDF-1.4 hello")); err != nil {
		t.Fatalf("put: %v", err)
	}

	res, err := svc.RequestDownloadURL(context.Background(), docID)
	if err != nil {
		t.Fatalf("download url: %v", err)
	}
	if res.URL != "memory://get/"+insert.StorageKey {
		t.Fatalf("url inesperada: %s", res.URL)
	}
	if res.ExpiresAt.IsZero() {
		t.Fatalf("expires_at no establecido")
	}

	// El scan lazy debió persistir antivirus limpio.
	if len(q.rows) == 0 || q.rows[0].Antivirus != AntivirusLimpio {
		t.Fatalf("antivirus debería quedar limpio tras el scan: %+v", q.rows)
	}
}

func TestRequestDownloadURLCuarentena(t *testing.T) {
	mem := NewMemoryStorage()
	env := DocsEnv{Storage: mem, Scanner: NewMockScanner(), MaxUpload: 10, SignedTTL: time.Minute}
	q := &fakeQuerier{}

	docID := "fb9d8a40-0000-4000-8000-000000000002"
	insert := db.InsertDocumentoParams{
		ID:         pgtype.UUID{Bytes: mustUUID(t, docID), Valid: true},
		Tipo:       "gasto",
		Nombre:     "malo.txt",
		StorageKey: "tenant-1/" + docID,
		SizeBytes:  5,
		Sha256:     testSHA,
	}
	q.rows = append(q.rows, db.Documento{
		ID: insert.ID, Tipo: insert.Tipo, Nombre: insert.Nombre,
		StorageKey: insert.StorageKey, SizeBytes: insert.SizeBytes, Antivirus: AntivirusPendiente,
	})
	// Firma EICAR → el mock escanea infectado.
	if err := mem.Put(context.Background(), insert.StorageKey, []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")); err != nil {
		t.Fatalf("put: %v", err)
	}

	_, err := NewService(env, q).RequestDownloadURL(context.Background(), docID)
	if !errors.Is(err, ErrDocumentoCuarentena) {
		t.Fatalf("got %v, want cuarentena", err)
	}
	if q.rows[0].Antivirus != AntivirusCuarentena {
		t.Fatalf("antivirus debería quedar en cuarentena: %+v", q.rows[0])
	}
}

func mustUUID(t *testing.T, s string) [16]byte {
	t.Helper()
	return uuid.MustParse(s)
}

// fakeQuerier es la implementación en memoria del repositorio de documentos.
type fakeQuerier struct {
	rows []db.Documento
}

func (f *fakeQuerier) InsertDocumento(_ context.Context, arg db.InsertDocumentoParams) (db.Documento, error) {
	doc := db.Documento{
		ID: arg.ID, ConsorcioID: arg.ConsorcioID, OwnerType: arg.OwnerType, OwnerID: arg.OwnerID,
		Tipo: arg.Tipo, Nombre: arg.Nombre, StorageKey: arg.StorageKey,
		MimeType: pgtype.Text{Valid: false}, SizeBytes: arg.SizeBytes, Sha256: arg.Sha256,
		Antivirus: AntivirusPendiente,
	}
	f.rows = append(f.rows, doc)
	return doc, nil
}

func (f *fakeQuerier) GetDocumento(_ context.Context, id pgtype.UUID) (db.Documento, error) {
	for _, r := range f.rows {
		if r.ID == id {
			return r, nil
		}
	}
	return db.Documento{}, errors.New("not found")
}

func (f *fakeQuerier) UpdateDocumentoScanResult(_ context.Context, arg db.UpdateDocumentoScanResultParams) (db.Documento, error) {
	for i := range f.rows {
		if f.rows[i].ID == arg.ID {
			f.rows[i].Antivirus = arg.Antivirus
			if arg.MimeType.Valid {
				f.rows[i].MimeType = arg.MimeType
			}
			return f.rows[i], nil
		}
	}
	return db.Documento{}, errors.New("not found")
}