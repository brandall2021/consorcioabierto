// Importación CSV de UFs (H2.3, §6.1).
//
// Flujo: createImportJob lee la plantilla `ufs-v1`, valida encabezado, tipos,
// duplicados y referencias SIN escribir nada (preview). El confirm aplica el
// modo explícito (crear|actualizar|crear_y_actualizar) en una transacción y es
// idempotente con Idempotency-Key (ADR-0004). Nunca hay upsert silencioso.
package consorcios

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// PlantillaUFSCSV es la plantilla versionada de UFs. Cambios de formato
	// incrementan la versión; el job la persiste para trazabilidad.
	PlantillaUFSCSV = "ufs-v1"

	idempotencyScope = "import_jobs.confirm"
)

var (
	// ErrImportInvalid: archivo no parseable, encabezado ausente o modo inválido.
	ErrImportInvalid = errors.New("import inválido")
	// ErrImportJobNotFound: el job no existe en el tenant (o no es visible).
	ErrImportJobNotFound = errors.New("job de importación no encontrado")
	// ErrImportJobEstado: el job no está en un estado accionable.
	ErrImportJobEstado = errors.New("job de importación en estado inválido")
	// ErrIdempotencyConflict: la clave Idempotency-Key ya se usó con otro request.
	ErrIdempotencyConflict = errors.New("Idempotency-Key ya utilizada con otro request")
)

// encabezadoUFSCSV define las columnas de la plantilla ufs-v1 en orden.
var encabezadoUFSCSV = []string{
	"codigo", "tipo", "superficie", "coeficiente", "nombre", "documento", "email", "telefono", "vinculo", "porcentaje",
}

type campoErrorDTO struct {
	Campo string `json:"campo"`
	Error string `json:"error"`
}

type ImportErrorDTO struct {
	Fila   int             `json:"fila"`
	Campos []campoErrorDTO `json:"campos"`
}

// ImportJobDTO es el contrato HTTP del job.
type ImportJobDTO struct {
	ID               string           `json:"id"`
	Estado           string           `json:"estado"`
	TotalFilas       int              `json:"total_filas"`
	Creados          int              `json:"creados"`
	Actualizados     int              `json:"actualizados"`
	Rechazados       int              `json:"rechazados"`
	Errores          []ImportErrorDTO `json:"errores"`
	ArchivoErroresURL *string          `json:"archivo_errores_url"`
}

// filaImportCSV representa una fila pendiente de confirmar. Si Valida=false se
// rechaza y su error ya quedó en Errores. Se persiste en JSONB.
type filaImportCSV struct {
	Fila        int      `json:"fila"`
	Codigo      string   `json:"codigo"`
	Tipo        string   `json:"tipo"`
	Superficie  *float64 `json:"superficie,omitempty"`
	Coeficiente string   `json:"coeficiente"`
	Nombre      string   `json:"nombre,omitempty"`
	Documento   *string  `json:"documento,omitempty"`
	Email       *string  `json:"email,omitempty"`
	Telefono    *string  `json:"telefono,omitempty"`
	Vinculo     string   `json:"vinculo,omitempty"`
	Porcentaje  *string  `json:"porcentaje,omitempty"`
	Valida      bool     `json:"valida"`
}

// importJob is the DB row plus parsed JSONB, used internally.
type importJob struct {
	job    db.ImportJob
	filas  []filaImportCSV
	errores []ImportErrorDTO
}

func importJobDTO(j *importJob) ImportJobDTO {
	dto := ImportJobDTO{
		ID:           j.job.ID.String(),
		Estado:       j.job.Estado,
		TotalFilas:   int(j.job.TotalFilas),
		Creados:      int(j.job.Creados),
		Actualizados: int(j.job.Actualizados),
		Rechazados:   int(j.job.Rechazados),
		Errores:      j.errores,
	}
	if j.job.ArchivoErroresUrl.Valid {
		s := j.job.ArchivoErroresUrl.String
		dto.ArchivoErroresURL = &s
	}
	return dto
}

// CreateImportJob valida la plantilla y persiste el job en estado `listo` sin
// escribir UFs. Devuelve el DTO listo para preview.
func CreateImportJob(ctx context.Context, q *db.Queries, consorcioID, modo string, r io.Reader) (ImportJobDTO, error) {
	if modo != "crear" && modo != "actualizar" && modo != "crear_y_actualizar" {
		return ImportJobDTO{}, ErrImportInvalid
	}
	var cid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return ImportJobDTO{}, ErrImportInvalid
	}
	// Referencia: el consorcio debe existir en el tenant (RLS).
	if _, err := q.GetConsorcio(ctx, cid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ImportJobDTO{}, ErrNotFound
		}
		return ImportJobDTO{}, err
	}

	filas, errores, err := parsePlantillaUFSCSV(r)
	if err != nil {
		return ImportJobDTO{}, err
	}
	filasJSON, err := json.Marshal(filas)
	if err != nil {
		return ImportJobDTO{}, err
	}
	erroresJSON, err := json.Marshal(errores)
	if err != nil {
		return ImportJobDTO{}, err
	}

	created, err := q.CreateImportJob(ctx, db.CreateImportJobParams{
		ConsorcioID:      cid,
		Modo:             modo,
		PlantillaVersion: PlantillaUFSCSV,
		TotalFilas:       int32(len(filas)),
		Filas:            filasJSON,
		Errores:          erroresJSON,
	})
	if err != nil {
		return ImportJobDTO{}, err
	}
	if err := q.SetImportJobEstado(ctx, db.SetImportJobEstadoParams{ID: created.ID, Estado: "listo"}); err != nil {
		return ImportJobDTO{}, err
	}
	created.Estado = "listo"

	j, err := parseImportJob(created)
	if err != nil {
		return ImportJobDTO{}, err
	}
	return importJobDTO(j), nil
}

// GetImportJob devuelve el job con sus errores (preview).
func GetImportJob(ctx context.Context, q *db.Queries, jobID string) (ImportJobDTO, error) {
	var jid pgtype.UUID
	if err := jid.Scan(jobID); err != nil {
		return ImportJobDTO{}, ErrImportJobNotFound
	}
	row, err := q.GetImportJob(ctx, jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImportJobDTO{}, ErrImportJobNotFound
	}
	if err != nil {
		return ImportJobDTO{}, err
	}
	j, err := parseImportJob(row)
	if err != nil {
		return ImportJobDTO{}, err
	}
	return importJobDTO(j), nil
}

// parseImportJob deserializa las columnas JSONB de una fila import_jobs.
func parseImportJob(row db.ImportJob) (*importJob, error) {
	j := &importJob{job: row}
	if len(row.Filas) > 0 {
		if err := json.Unmarshal(row.Filas, &j.filas); err != nil {
			return nil, err
		}
	}
	if len(row.Errores) > 0 {
		if err := json.Unmarshal(row.Errores, &j.errores); err != nil {
			return nil, err
		}
	}
	return j, nil
}

// parsePlantillaUFSCSV lee y valida la plantilla sin escribir nada. Cada fila se
// valida campo a campo; una fila con errores queda marcada como no válida.
func parsePlantillaUFSCSV(r io.Reader) ([]filaImportCSV, []ImportErrorDTO, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = len(encabezadoUFSCSV)
	cr.ReuseRecord = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: CSV no parseable: %v", ErrImportInvalid, err)
	}
	if len(records) < 2 {
		return nil, nil, fmt.Errorf("%w: archivo vacío (solo encabezado)", ErrImportInvalid)
	}
	header := normalizeCSVRecord(records[0])
	if len(header) != len(encabezadoUFSCSV) {
		return nil, nil, fmt.Errorf("%w: encabezado no coincide con plantilla %s", ErrImportInvalid, PlantillaUFSCSV)
	}
	for i, h := range header {
		if h != encabezadoUFSCSV[i] {
			return nil, nil, fmt.Errorf("%w: encabezado no coincide con plantilla %s (columna %d)", ErrImportInvalid, PlantillaUFSCSV, i+1)
		}
	}

	seenCodigo := map[string]bool{}
	filas := make([]filaImportCSV, 0, len(records)-1)
	totalErrores := make([]ImportErrorDTO, 0)

	for idx, rec := range records[1:] {
		filaNum := idx + 2 // fila 1 es el encabezado
		rec = normalizeCSVRecord(rec)
		f := filaImportCSV{Fila: filaNum, Coeficiente: rec[3], Valida: true}
		var campos []campoErrorDTO

		// codigo: obligatorio, único dentro del archivo.
		f.Codigo = strings.TrimSpace(rec[0])
		if f.Codigo == "" || len(f.Codigo) > 50 {
			campos = append(campos, campoErrorDTO{Campo: "codigo", Error: "código obligatorio (máx 50 caracteres)"})
		} else if seenCodigo[f.Codigo] {
			campos = append(campos, campoErrorDTO{Campo: "codigo", Error: "código duplicado en el archivo"})
		}
		seenCodigo[f.Codigo] = true

		// tipo: enum.
		f.Tipo = strings.TrimSpace(rec[1])
		if !validTiposUF[f.Tipo] {
			campos = append(campos, campoErrorDTO{Campo: "tipo", Error: "tipo inválido (departamento|cochera|local|unidad_edificio|otros)"})
		}

		// superficie: opcional, >= 0, numérico.
		if s := strings.TrimSpace(rec[2]); s != "" {
			v, perr := strconv.ParseFloat(s, 64)
			if perr != nil || v < 0 {
				campos = append(campos, campoErrorDTO{Campo: "superficie", Error: "superficie inválida (numérico >= 0)"})
			} else {
				f.Superficie = &v
			}
		}

		// coeficiente: obligatorio salvo default 0; mismo formato que el API.
		if c := strings.TrimSpace(rec[3]); c != "" && !coefRe.MatchString(c) {
			campos = append(campos, campoErrorDTO{Campo: "coeficiente", Error: "coeficiente inválido (hasta 8 decimales)"})
		} else {
			f.Coeficiente = strings.TrimSpace(c)
		}

		// vínculo con persona (opcional salvo si hay persona).
		f.Nombre = strings.TrimSpace(rec[4])
		f.Vinculo = strings.TrimSpace(rec[8])
		if f.Vinculo == "" {
			f.Vinculo = "propietario"
		}
		f.Documento = optionalStr(rec[5])
		f.Email = optionalStr(rec[6])
		f.Telefono = optionalStr(rec[7])
		f.Porcentaje = optionalStr(rec[9])

		hasPersona := f.Nombre != "" || f.Documento != nil || f.Email != nil || f.Telefono != nil
		if hasPersona {
			if f.Nombre == "" || len(f.Nombre) > 200 {
				campos = append(campos, campoErrorDTO{Campo: "nombre", Error: "nombre obligatorio si se declara una persona (máx 200)"})
			}
			if !validVinculos[f.Vinculo] {
				campos = append(campos, campoErrorDTO{Campo: "vinculo", Error: "vínculo inválido (propietario|inquilino|apoderado)"})
			}
			if f.Porcentaje != nil && (!pctRe.MatchString(*f.Porcentaje) || pctGt100(*f.Porcentaje)) {
				campos = append(campos, campoErrorDTO{Campo: "porcentaje", Error: "porcentaje inválido (0-100, hasta 2 decimales)"})
			}
		} else {
			f.Vinculo = ""
			f.Porcentaje = nil
		}

		if len(campos) > 0 {
			f.Valida = false
			totalErrores = append(totalErrores, ImportErrorDTO{Fila: filaNum, Campos: campos})
		}
		filas = append(filas, f)
	}
	return filas, totalErrores, nil
}

func pctGt100(s string) bool {
	v, err := strconv.ParseFloat(s, 64)
	return err != nil || v > 100
}

func normalizeCSVRecord(rec []string) []string {
	out := make([]string, len(rec))
	for i, c := range rec {
		out[i] = strings.TrimSpace(strings.TrimPrefix(c, "\ufeff"))
	}
	return out
}

func optionalStr(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

// ConfirmImportJob aplica el modo del job a las filas válidas en la misma
// transacción y es idempotente con Idempotency-Key (ADR-0004): la primera
// ejecución persiste el resultado; reintentos con la misma clave devuelven el
// resultado previo sin re-ejecutar efectos.
func ConfirmImportJob(ctx context.Context, q *db.Queries, jobID, idemKey string) (ImportJobDTO, error) {
	var jid pgtype.UUID
	if err := jid.Scan(jobID); err != nil {
		return ImportJobDTO{}, ErrImportJobNotFound
	}
	row, err := q.GetImportJob(ctx, jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImportJobDTO{}, ErrImportJobNotFound
	}
	if err != nil {
		return ImportJobDTO{}, err
	}
	j, err := parseImportJob(row)
	if err != nil {
		return ImportJobDTO{}, err
	}
	if j.job.Estado == "procesado" {
		return importJobDTO(j), nil
	}
	if j.job.Estado != "listo" {
		return ImportJobDTO{}, ErrImportJobEstado
	}

	hash := sha256.Sum256([]byte(jobID + "|" + idemKey))
	requestHash := hex.EncodeToString(hash[:])

	// Idempotencia: primer intento inserta la clave; si ya existía, se devuelve
	// el resultado almacenado (mismo request) o 409 (clave reutilizada).
	inserted, err := q.InsertIdempotencyKey(ctx, db.InsertIdempotencyKeyParams{
		IdempotencyKey: idemKey,
		Scope:          idempotencyScope,
		RequestHash:    requestHash,
		ResponseJson:   []byte("{}"),
	})
	if err != nil {
		return ImportJobDTO{}, err
	}
	if inserted == 0 {
		existing, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{
			Scope:          idempotencyScope,
			IdempotencyKey: idemKey,
		})
		if err != nil {
			return ImportJobDTO{}, err
		}
		if existing.RequestHash != requestHash {
			return ImportJobDTO{}, ErrIdempotencyConflict
		}
		var dto ImportJobDTO
		if err := json.Unmarshal(existing.ResponseJson, &dto); err != nil {
			return ImportJobDTO{}, err
		}
		return dto, nil
	}

	// Estado transitorio confirmando y procesamiento.
	if err := q.SetImportJobEstado(ctx, db.SetImportJobEstadoParams{ID: jid, Estado: "confirmando"}); err != nil {
		return ImportJobDTO{}, err
	}

	creados, actualizados, erroresFinal, err := procesarFilas(ctx, q, j)
	if err != nil {
		return ImportJobDTO{}, err
	}
	rechazados := int(j.job.TotalFilas) - creados - actualizados
	erroresJSON, err := json.Marshal(erroresFinal)
	if err != nil {
		return ImportJobDTO{}, err
	}

	finished, err := q.FinishImportJob(ctx, db.FinishImportJobParams{
		ID:         jid,
		Estado:     "procesado",
		Creados:    int32(creados),
		Actualizados: int32(actualizados),
		Rechazados: int32(rechazados),
		Errores:    erroresJSON,
	})
	if err != nil {
		return ImportJobDTO{}, err
	}
	jf, err := parseImportJob(finished)
	if err != nil {
		return ImportJobDTO{}, err
	}
	dto := importJobDTO(jf)

	// Persistir el resultado idempotente junto a la operación.
	respJSON, err := json.Marshal(dto)
	if err != nil {
		return ImportJobDTO{}, err
	}
	if err := q.UpdateIdempotencyKey(ctx, db.UpdateIdempotencyKeyParams{
		Scope:          idempotencyScope,
		IdempotencyKey: idemKey,
		ResponseJson:   respJSON,
	}); err != nil {
		return ImportJobDTO{}, err
	}
	return dto, nil
}

// procesarFilas recorre las filas válidas según el modo. Las filas inválidas ya
// quedaron registradas como errores de preview; los errores de runtime (por
// ejemplo, código ya existe al confirmar en modo crear) se agregan al final.
func procesarFilas(ctx context.Context, q *db.Queries, j *importJob) (int, int, []ImportErrorDTO, error) {
	creados := 0
	actualizados := 0
	errores := append([]ImportErrorDTO{}, j.errores...)

	for _, f := range j.filas {
		if !f.Valida {
			continue
		}
		switch j.job.Modo {
		case "crear":
			if _, err := q.GetUnidadByCodigo(ctx, db.GetUnidadByCodigoParams{
				ConsorcioID: j.job.ConsorcioID,
				Codigo:      f.Codigo,
			}); err == nil {
				errores = append(errores, ImportErrorDTO{Fila: f.Fila, Campos: []campoErrorDTO{{Campo: "codigo", Error: "código ya existe"}}})
				continue
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return creados, actualizados, errores, err
			}
			if _, err := crearUnidadDesdeFila(ctx, q, j.job.ConsorcioID, f); err != nil {
				return creados, actualizados, errores, err
			}
			creados++
		case "actualizar", "crear_y_actualizar":
			existente, err := q.GetUnidadByCodigo(ctx, db.GetUnidadByCodigoParams{
				ConsorcioID: j.job.ConsorcioID,
				Codigo:      f.Codigo,
			})
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				if j.job.Modo == "actualizar" {
					errores = append(errores, ImportErrorDTO{Fila: f.Fila, Campos: []campoErrorDTO{{Campo: "codigo", Error: "código no existe"}}})
					continue
				}
				if _, err := crearUnidadDesdeFila(ctx, q, j.job.ConsorcioID, f); err != nil {
					return creados, actualizados, errores, err
				}
				creados++
			case err != nil:
				return creados, actualizados, errores, err
			default:
				if err := actualizarUnidadDesdeFila(ctx, q, j.job.ConsorcioID, existente.ID, f); err != nil {
					return creados, actualizados, errores, err
				}
				actualizados++
			}
		}
	}
	return creados, actualizados, errores, nil
}

// crearUnidadDesdeFila crea una UF con su vínculo inicial (si la fila declara persona).
func crearUnidadDesdeFila(ctx context.Context, q *db.Queries, consorcioID pgtype.UUID, f filaImportCSV) (*db.Unidade, error) {
	var superficie pgtype.Numeric
	if f.Superficie != nil {
		superficie = floatToNumeric(*f.Superficie)
	}
	coef, err := parseNumeric(f.Coeficiente)
	if err != nil {
		coef = pgtype.Numeric{} // validado en preview; no debería llegar acá
	}
	u, err := q.CreateUnidad(ctx, db.CreateUnidadParams{
		ConsorcioID: consorcioID,
		Codigo:      f.Codigo,
		Tipo:        f.Tipo,
		Superficie:  superficie,
		Coeficiente: coef,
	})
	if err != nil {
		return nil, err
	}
	if f.TienePersona() {
		v, err := filaToValidatedVinculo(f)
		if err != nil {
			return nil, err
		}
		if err := attachPersona(ctx, q, u.ID, v); err != nil {
			return nil, err
		}
	}
	return &u, nil
}

// actualizarUnidadDesdeFila actualiza los campos editables y reemplaza los
// vínculos vigentes de la UF por el de la fila (si declara persona).
func actualizarUnidadDesdeFila(ctx context.Context, q *db.Queries, consorcioID pgtype.UUID, unidadID pgtype.UUID, f filaImportCSV) error {
	var superficie pgtype.Numeric
	if f.Superficie != nil {
		superficie = floatToNumeric(*f.Superficie)
	}
	coef, err := parseNumeric(f.Coeficiente)
	if err != nil {
		coef = pgtype.Numeric{}
	}
	if _, err := q.UpdateUnidad(ctx, db.UpdateUnidadParams{
		ConsorcioID: consorcioID,
		ID:          unidadID,
		Tipo:        pgtype.Text{String: f.Tipo, Valid: true},
		Superficie:  superficie,
		Coeficiente: coef,
	}); err != nil {
		return err
	}

	if !f.TienePersona() {
		return nil
	}
	v, err := filaToValidatedVinculo(f)
	if err != nil {
		return err
	}
	// Cerrar vínculos vigentes actuales y crear el nuevo (histórico).
	if _, err := q.CloseAllVinculosVigentes(ctx, db.CloseAllVinculosVigentesParams{
		UnidadID: unidadID,
		ValidTo:  pgtype.Date{Time: time.Now(), Valid: true},
	}); err != nil {
		return err
	}
	return attachPersona(ctx, q, unidadID, v)
}

// TienePersona informa si la fila declara una persona para vincular.
func (f filaImportCSV) TienePersona() bool {
	return f.Nombre != "" || f.Documento != nil || f.Email != nil || f.Telefono != nil
}

// filaToValidatedVinculo convierte una fila con persona al vínculo validado.
func filaToValidatedVinculo(f filaImportCSV) (validatedVinculo, error) {
	return validateVinculo(PersonaVinculoInput{
		Persona: PersonaInput{
			Nombre:    f.Nombre,
			Documento: f.Documento,
			Email:     f.Email,
			Telefono:  f.Telefono,
		},
		Vinculo:    f.Vinculo,
		Porcentaje: f.Porcentaje,
	})
}