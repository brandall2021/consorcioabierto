package documentos

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Querier es la porción del repositorio (db.Queries) que consume el servicio de
// documentos. Se define acá para que el servicio sea testeable sin DB real; el
// adaptador concreto es `db.New(tx)` desde el handler.
type Querier interface {
	InsertDocumento(ctx context.Context, arg db.InsertDocumentoParams) (db.Documento, error)
	GetDocumento(ctx context.Context, id pgtype.UUID) (db.Documento, error)
	UpdateDocumentoScanResult(ctx context.Context, arg db.UpdateDocumentoScanResultParams) (db.Documento, error)
}

// CreateUploadIntentInput es el request body del intent de subida (openapi: tipo,
// nombre, size_bytes, sha256; owner_type/owner_id son opcionales).
type CreateUploadIntentInput struct {
	ConsorcioID *string `json:"consorcio_id"`
	OwnerType   *string `json:"owner_type"`
	OwnerID     *string `json:"owner_id"`
	Tipo        string  `json:"tipo"`
	Nombre      string  `json:"nombre"`
	SizeBytes   int64   `json:"size_bytes"`
	SHA256      string  `json:"sha256"`
}

// CreateUploadIntentResult es la respuesta del intent (openapi: documento_id +
// upload_url).
type CreateUploadIntentResult struct {
	DocumentoID string `json:"documento_id"`
	UploadURL   string `json:"upload_url"`
}

// DownloadURLResult es la respuesta de la URL firmada de descarga (openapi:
// DocumentDownload { url, expires_at }).
type DownloadURLResult struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service orquesta los casos de uso de documentos (upload intent y descarga)
// sobre un storage S3-compatible y un antivirus, contra el repositorio de la
// tenancy activa. Se construye por request: el handler lo instancia con el
// *db.Queries acotado por el tenant del token.
type Service struct {
	Q         Querier
	Storage   Storage
	Scanner   Scanner
	MaxUpload int64
	SignedTTL time.Duration
}

// DocsEnv agrupa las dependencias ambientales de documentos (storage, scanner
// y límites) que el handler inyecta por request en el Service.
type DocsEnv struct {
	Storage   Storage
	Scanner   Scanner
	MaxUpload int64
	SignedTTL time.Duration
}

// NewService construye el servicio con las dependencias ambientales y el
// repositorio acotado por tenant de la request.
func NewService(env DocsEnv, q Querier) *Service {
	return &Service{
		Q:         q,
		Storage:   env.Storage,
		Scanner:   env.Scanner,
		MaxUpload: env.MaxUpload,
		SignedTTL: env.SignedTTL,
	}
}

// CreateUploadIntent crea la fila del documento en estado pendiente y devuelve
// la URL firmada de subida. La storage key se genera como {tenant_id}/{id} y
// el cliente sube directo al bucket. ([ADR-0008], §5.4).
func (s *Service) CreateUploadIntent(
	ctx context.Context,
	tenantID string,
	in CreateUploadIntentInput,
) (CreateUploadIntentResult, error) {
	// --- validación (rechaza args del cliente → 400/413) ---
	in.Tipo = strings.TrimSpace(in.Tipo)
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.SHA256 != "" {
		in.SHA256 = strings.TrimSpace(in.SHA256)
	}

	if in.Tipo == "" || len(in.Tipo) > 50 {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: tipo requerido (max 50)", ErrDocumentoInvalid)
	}
	if in.Nombre == "" || len(in.Nombre) > 255 {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: nombre requerido (max 255)", ErrDocumentoInvalid)
	}
	if !sha256HexRe.MatchString(in.SHA256) {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: sha256 inválido (64 hex)", ErrDocumentoInvalid)
	}
	if in.SizeBytes <= 0 {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: tamaño inválido", ErrDocumentoInvalid)
	}
	if in.SizeBytes > s.MaxUpload {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: %d bytes", ErrDocumentoTooLarge, in.SizeBytes)
	}

	// --- generar id y storage key ---
	id := uuid.NewString()
	storageKey := tenantID + "/" + id

	// --- insertar fila (pendiente) ---
	params := db.InsertDocumentoParams{
		ID:         pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true},
		Tipo:       in.Tipo,
		Nombre:     in.Nombre,
		StorageKey: storageKey,
		SizeBytes:  in.SizeBytes,
		Sha256:     in.SHA256,
	}
	if in.ConsorcioID != nil && *in.ConsorcioID != "" {
		params.ConsorcioID = pgtype.UUID{Bytes: uuid.MustParse(*in.ConsorcioID), Valid: true}
	}
	if in.OwnerType != nil && *in.OwnerType != "" {
		params.OwnerType = pgtype.Text{String: *in.OwnerType, Valid: true}
	}
	if in.OwnerID != nil && *in.OwnerID != "" {
		params.OwnerID = pgtype.UUID{Bytes: uuid.MustParse(*in.OwnerID), Valid: true}
	}

	if _, err := s.Q.InsertDocumento(ctx, params); err != nil {
		return CreateUploadIntentResult{}, fmt.Errorf("insert documento: %w", err)
	}

	// --- presign PUT ---
	uploadURL, err := s.Storage.PresignPut(ctx, storageKey, in.SizeBytes, s.SignedTTL)
	if err != nil {
		return CreateUploadIntentResult{}, fmt.Errorf("%w: presign put", err)
	}

	return CreateUploadIntentResult{
		DocumentoID: id,
		UploadURL:   uploadURL,
	}, nil
}

// RequestDownloadURL autoriza la descarga del documento y devuelve una URL
// firmada breve. Si el antivirus está pendiente, realiza el escaneo lazy
// (determinista en mock; no-op en none) y guarda el resultado. ([ADR-0008],
// §5.4: "La descarga usa URL firmada breve después de autorizar el recurso").
func (s *Service) RequestDownloadURL(
	ctx context.Context,
	docID string,
) (DownloadURLResult, error) {
	doc, err := s.Q.GetDocumento(ctx, pgtype.UUID{Bytes: uuid.MustParse(docID), Valid: true})
	if err != nil {
		return DownloadURLResult{}, fmt.Errorf("%w: get documento", ErrDocumentoNotFound)
	}

	if doc.Antivirus == AntivirusCuarentena {
		return DownloadURLResult{}, ErrDocumentoCuarentena
	}

	// --- scan lazy: solo si antivirus sigue en pendiente ---
	if doc.Antivirus == AntivirusPendiente {
		if err := s.scanAndUpdate(ctx, doc); err != nil {
			return DownloadURLResult{}, err
		}
		// Recargar después del update para leer el nuevo estado.
		doc, err = s.Q.GetDocumento(ctx, pgtype.UUID{Bytes: uuid.MustParse(docID), Valid: true})
		if err != nil {
			return DownloadURLResult{}, fmt.Errorf("%w: re-leer documento", ErrDocumentoNotFound)
		}
	}

	if doc.Antivirus == AntivirusCuarentena {
		return DownloadURLResult{}, ErrDocumentoCuarentena
	}

	// --- presign GET ---
	url, err := s.Storage.PresignGet(ctx, doc.StorageKey, s.SignedTTL)
	if err != nil {
		return DownloadURLResult{}, fmt.Errorf("%w: presign get", ErrStorageUnavailable)
	}
	return DownloadURLResult{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(s.SignedTTL),
	}, nil
}

func (s *Service) scanAndUpdate(
	ctx context.Context,
	doc db.Documento,
) error {
	r, err := s.Storage.Get(ctx, doc.StorageKey)
	if err != nil {
		return fmt.Errorf("%w: objeto no encontrado en storage", ErrDocumentoNotFound)
	}
	defer func() { _ = r.Close() }()

	result, err := s.Scanner.Scan(ctx, r)
	if err != nil {
		return fmt.Errorf("escaneo: %w", err)
	}

	antivirus := AntivirusLimpio
	if !result.Limpio {
		antivirus = AntivirusCuarentena
	}

	mimeType := pgtype.Text{Valid: false}
	if result.Mime != "" {
		mimeType = pgtype.Text{String: result.Mime, Valid: true}
	}

	if _, err := s.Q.UpdateDocumentoScanResult(ctx, db.UpdateDocumentoScanResultParams{
		Antivirus: antivirus,
		MimeType:  mimeType,
		ID:        doc.ID,
	}); err != nil {
		return fmt.Errorf("actualizar scan result: %w", err)
	}
	return nil
}