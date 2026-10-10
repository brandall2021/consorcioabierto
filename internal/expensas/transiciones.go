// Transiciones de liquidaciones (§5.2): calcular → confirmar → publicar, y
// anular desde cualquier estado activo. Todas operan sobre el *db.Queries de
// la transacción del handler (RLS de tenant ya activo); aquí no se abren
// transacciones (Ruling 1): el commit/rollback lo decide el caller.
package expensas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/brandall2021/consorcioabierto/internal/cuenta_corriente"
	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/brandall2021/consorcioabierto/internal/outbox"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrLiquidacionSinGastos = errors.New("no hay gastos registrados para el período")
	ErrLiquidacionSinUFs    = errors.New("no hay unidades activas para el consorcio")
	ErrIdempotencyConflict  = errors.New("Idempotency-Key ya utilizada con otro request")
)

const (
	idempScopeCalcular  = "expensas.calcular"
	idempScopeConfirmar = "expensas.confirmar"
	idempScopePublicar  = "expensas.publicar"
)

// agruparConceptos consolida el importe de los gastos por concepto de expensa.
// La única regla que permite el esquema (CHECK de 00013) es 'coeficiente'.
// Se conserva el orden de primera aparición de cada concepto.
func agruparConceptos(gastos []db.Gasto) []ConceptoCalculo {
	if len(gastos) == 0 {
		return nil
	}
	out := make([]ConceptoCalculo, 0, len(gastos))
	idx := make(map[string]int, len(gastos))
	for _, g := range gastos {
		key := g.ConceptoID.String()
		if i, ok := idx[key]; ok {
			out[i].ImporteCents += g.ImporteCents
			continue
		}
		idx[key] = len(out)
		out = append(out, ConceptoCalculo{
			ConceptoID:   key,
			Regla:        "coeficiente",
			ImporteCents: g.ImporteCents,
		})
	}
	return out
}

// toUnidadCobro convierte el snapshot de unidades del preview en unidades de
// cobro para generar cargos. La fecha de cobro es el vencimiento_1 de la
// liquidación (Ruling 11), no el día 01 del período. Las unidades con total
// <= 0 (p.ej. coeficiente 0) se omiten para no crear cargos vacíos.
func toUnidadCobro(rows []db.GetLiquidacionUnidadesRow, periodo, liquidacionID, vencimiento1 string) []cuenta_corriente.UnidadCobro {
	out := make([]cuenta_corriente.UnidadCobro, 0, len(rows))
	for _, row := range rows {
		if row.TotalCents <= 0 {
			continue
		}
		out = append(out, cuenta_corriente.UnidadCobro{
			UnidadID:      row.UnidadID.String(),
			Concepto:      "Expensa " + periodo,
			DueDate:       vencimiento1,
			TotalCents:    row.TotalCents,
			LiquidacionID: &liquidacionID,
		})
	}
	return out
}

// stampIdempotency reserva la clave de idempotencia del request (misma
// semántica que consorcios/imports.go). Si ya estaba registrada, valida que el
// request hash coincida y devuelve la respuesta cacheada del primer intento.
func stampIdempotency(ctx context.Context, q *db.Queries, scope, liquidacionID, idemKey string) (LiquidacionDTO, bool, error) {
	hash := sha256.Sum256([]byte(liquidacionID + "|" + idemKey))
	requestHash := hex.EncodeToString(hash[:])

	inserted, err := q.InsertIdempotencyKey(ctx, db.InsertIdempotencyKeyParams{
		IdempotencyKey: idemKey,
		Scope:          scope,
		RequestHash:    requestHash,
		ResponseJson:   []byte("{}"),
	})
	if err != nil {
		return LiquidacionDTO{}, false, err
	}
	if inserted == 0 {
		existing, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{
			Scope:          scope,
			IdempotencyKey: idemKey,
		})
		if err != nil {
			return LiquidacionDTO{}, false, err
		}
		if existing.RequestHash != requestHash {
			return LiquidacionDTO{}, false, ErrIdempotencyConflict
		}
		var cached LiquidacionDTO
		if err := json.Unmarshal(existing.ResponseJson, &cached); err != nil {
			return LiquidacionDTO{}, false, err
		}
		return cached, true, nil
	}
	return LiquidacionDTO{}, false, nil
}

// saveIdempotency persiste el resultado exitoso para reintentos idempotentes.
func saveIdempotency(ctx context.Context, q *db.Queries, scope, idemKey string, dto LiquidacionDTO) error {
	if idemKey == "" {
		return nil
	}
	respJSON, err := json.Marshal(dto)
	if err != nil {
		return err
	}
	return q.UpdateIdempotencyKey(ctx, db.UpdateIdempotencyKeyParams{
		Scope:          scope,
		IdempotencyKey: idemKey,
		ResponseJson:   respJSON,
	})
}

// CalcularLiquidacion congela los gastos registrados del período, distribuye
// por coeficiente y persiste el snapshot. Permitido desde 'borrador' (primer
// cálculo) y desde 'calculada' (recálculo: reemplaza el snapshot previo).
func CalcularLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error) {
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	if err := lid.Scan(liquidacionID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}

	liq, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if liq.Estado != "borrador" && liq.Estado != "calculada" {
		return LiquidacionDTO{}, ErrLiquidacionTransicionInvalida
	}

	// Gastos del período; solo los registrados (ListGastos no filtra estado).
	rows, err := q.ListGastos(ctx, db.ListGastosParams{
		ConsorcioID: cid,
		Mes:         liq.Periodo[:4] + "-" + liq.Periodo[4:],
		ProveedorID: pgtype.UUID{},
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	registrados := make([]db.Gasto, 0, len(rows))
	for _, g := range rows {
		if g.Estado == "registrado" {
			registrados = append(registrados, g)
		}
	}
	if len(registrados) == 0 {
		return LiquidacionDTO{}, ErrLiquidacionSinGastos
	}

	ufsRows, err := q.ListUnidades(ctx, db.ListUnidadesParams{ConsorcioID: cid, Estado: "activa"})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if len(ufsRows) == 0 {
		return LiquidacionDTO{}, ErrLiquidacionSinUFs
	}

	// Idempotencia: se sella DESPUÉS de las lecturas de pre-flight y antes de
	// mutar. Sellarla primero cachearía un {} ante un error de negocio (sin
	// gastos/UFs) y el reintento fallaría al deserializar ese JSON.
	if idemKey != "" {
		cached, hit, err := stampIdempotency(ctx, q, idempScopeCalcular, liquidacionID, idemKey)
		if err != nil {
			return LiquidacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	ufs := make([]UFCoef, 0, len(ufsRows))
	for _, u := range ufsRows {
		f, _ := u.Coeficiente.Float64Value()
		ufs = append(ufs, UFCoef{
			UnidadID:    u.ID.String(),
			Codigo:      u.Codigo,
			Coeficiente: f.Float64,
		})
	}

	result, err := Distribuir(agruparConceptos(registrados), ufs)
	if err != nil {
		return LiquidacionDTO{}, err
	}

	// Reemplazar el snapshot anterior (recálculo): gasto, item, unidad.
	if err := q.DeleteLiquidacionGastos(ctx, lid); err != nil {
		return LiquidacionDTO{}, err
	}
	if err := q.DeleteLiquidacionItems(ctx, lid); err != nil {
		return LiquidacionDTO{}, err
	}
	if err := q.DeleteLiquidacionUnidades(ctx, lid); err != nil {
		return LiquidacionDTO{}, err
	}

	for _, g := range registrados {
		if err := q.InsertLiquidacionGasto(ctx, db.InsertLiquidacionGastoParams{
			LiquidacionID: lid,
			GastoID:       g.ID,
			ConceptoID:    g.ConceptoID,
			ImporteCents:  g.ImporteCents,
		}); err != nil {
			return LiquidacionDTO{}, fmt.Errorf("snapshot gastos: %w", err)
		}
	}

	for _, item := range result.Items {
		var concepto pgtype.UUID
		_ = concepto.Scan(item.ConceptoID)
		if err := q.InsertLiquidacionItem(ctx, db.InsertLiquidacionItemParams{
			LiquidacionID: lid,
			ConceptoID:    concepto,
			ReglaAplicada: item.Regla,
			ImporteCents:  item.ImporteCents,
		}); err != nil {
			return LiquidacionDTO{}, fmt.Errorf("snapshot items: %w", err)
		}
	}

	for _, u := range result.Unidades {
		var unidad pgtype.UUID
		_ = unidad.Scan(u.UnidadID)
		var coef pgtype.Numeric
		_ = coef.Scan(u.Coeficiente)
		if err := q.InsertLiquidacionUnidad(ctx, db.InsertLiquidacionUnidadParams{
			LiquidacionID: lid,
			UnidadID:      unidad,
			Codigo:        u.Codigo,
			Coeficiente:   coef,
			TotalCents:    u.TotalCents,
		}); err != nil {
			return LiquidacionDTO{}, fmt.Errorf("snapshot unidades: %w", err)
		}
	}

	if err := q.UpdateLiquidacionCalculo(ctx, db.UpdateLiquidacionCalculoParams{
		TotalGastosCents:      result.TotalGastosCents,
		TotalDistribuidoCents: result.TotalDistribuidoCents,
		UnidadesAlcanzadas:    int32(result.UnidadesAlcanzadas),
		ID:                    lid,
	}); err != nil {
		return LiquidacionDTO{}, err
	}

	n, err := q.TransitionLiquidacion(ctx, db.TransitionLiquidacionParams{
		NuevoEstado:     "calculada",
		ConsorcioID:     cid,
		ID:              lid,
		EstadoActual:    liq.Estado,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if n == 0 {
		return LiquidacionDTO{}, ErrLiquidacionVersionMismatch
	}

	dto, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if err := saveIdempotency(ctx, q, idempScopeCalcular, idemKey, dto); err != nil {
		return LiquidacionDTO{}, err
	}
	return dto, nil
}

// ConfirmarLiquidacion genera los cargos por UF a partir del snapshot
// calculado y transiciona a 'confirmada'. No emite evento outbox (Ruling 12:
// el worker solo procesa 'liquidacion.publicada'; un evento de confirmación
// quedaría como fila venenosa fallida).
func ConfirmarLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error) {
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	if err := lid.Scan(liquidacionID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}

	liq, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if liq.Estado != "calculada" {
		return LiquidacionDTO{}, ErrLiquidacionTransicionInvalida
	}

	if idemKey != "" {
		cached, hit, err := stampIdempotency(ctx, q, idempScopeConfirmar, liquidacionID, idemKey)
		if err != nil {
			return LiquidacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	preview, err := q.GetLiquidacionUnidades(ctx, lid)
	if err != nil {
		return LiquidacionDTO{}, err
	}

	_, _, err = cuenta_corriente.CreateCargos(ctx, q, toUnidadCobro(preview, liq.Periodo, liquidacionID, liq.Vencimiento1), "ARS")
	if err != nil {
		return LiquidacionDTO{}, err
	}

	n, err := q.TransitionLiquidacion(ctx, db.TransitionLiquidacionParams{
		NuevoEstado:     "confirmada",
		ConsorcioID:     cid,
		ID:              lid,
		EstadoActual:    liq.Estado,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if n == 0 {
		return LiquidacionDTO{}, ErrLiquidacionVersionMismatch
	}

	dto, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if err := saveIdempotency(ctx, q, idempScopeConfirmar, idemKey, dto); err != nil {
		return LiquidacionDTO{}, err
	}
	return dto, nil
}

// PublicarLiquidacion encola el evento 'liquidacion.publicada' (lo consume el
// worker de outbox para generar el PDF y avisar a los consorcistas) y
// transiciona a 'publicada'.
func PublicarLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error) {
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	if err := lid.Scan(liquidacionID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}

	liq, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if liq.Estado != "confirmada" {
		return LiquidacionDTO{}, ErrLiquidacionTransicionInvalida
	}

	if idemKey != "" {
		cached, hit, err := stampIdempotency(ctx, q, idempScopePublicar, liquidacionID, idemKey)
		if err != nil {
			return LiquidacionDTO{}, err
		}
		if hit {
			return cached, nil
		}
	}

	payloadBytes, err := json.Marshal(outbox.LiquidacionPayload{
		LiquidacionID: liquidacionID,
		Periodo:       liq.Periodo,
		ConsorcioID:   consorcioID,
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if _, err := q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
		CorrelationID: lid.String(),
		EventType:     outbox.EventPublished,
		Payload:       payloadBytes,
	}); err != nil {
		return LiquidacionDTO{}, err
	}

	n, err := q.TransitionLiquidacion(ctx, db.TransitionLiquidacionParams{
		NuevoEstado:     "publicada",
		ConsorcioID:     cid,
		ID:              lid,
		EstadoActual:    liq.Estado,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if n == 0 {
		return LiquidacionDTO{}, ErrLiquidacionVersionMismatch
	}

	dto, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if err := saveIdempotency(ctx, q, idempScopePublicar, idemKey, dto); err != nil {
		return LiquidacionDTO{}, err
	}
	return dto, nil
}

// AnularLiquidacion revierte los cargos pendientes (si había) y transiciona a
// 'anulada'. El motivo de la anulación (Ruling 10, spec H3.2) es informativo
// aquí: se audita en el handler (Task 10), no se persiste en la liquidación.
func AnularLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, motivo string) (LiquidacionDTO, error) {
	var cid, lid pgtype.UUID
	if err := cid.Scan(consorcioID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionInvalid
	}
	if err := lid.Scan(liquidacionID); err != nil {
		return LiquidacionDTO{}, ErrLiquidacionNotFound
	}

	liq, err := GetLiquidacion(ctx, q, consorcioID, liquidacionID)
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if liq.Estado == "publicada" || liq.Estado == "anulada" || liq.Estado == "cerrada" {
		return LiquidacionDTO{}, ErrLiquidacionTransicionInvalida
	}

	// Si la liquidación generó cargos, reversar los que sigan con saldo
	// pendiente. Los errores NO se tragan: romper una invariante de dinero
	// (R3/R4) no debe dejarse pasar en silencio.
	if liq.Estado == "confirmada" || liq.Estado == "publicada" {
		charges, err := q.GetChargesByLiquidacion(ctx, lid)
		if err != nil {
			return LiquidacionDTO{}, err
		}
		for _, ch := range charges {
			if ch.SaldoCents > 0 {
				if _, err := cuenta_corriente.RevertirCargo(ctx, q, ch.ID.String(), ch.UnidadID.String(), "ARS"); err != nil {
					return LiquidacionDTO{}, fmt.Errorf("reversa cargo %s: %w", ch.ID.String(), err)
				}
			}
		}
	}

	n, err := q.AnularLiquidacion(ctx, db.AnularLiquidacionParams{
		ID:              lid,
		EstadoActual:    liq.Estado,
		ExpectedVersion: int32(expectedVersion),
	})
	if err != nil {
		return LiquidacionDTO{}, err
	}
	if n == 0 {
		return LiquidacionDTO{}, ErrLiquidacionVersionMismatch
	}
	return GetLiquidacion(ctx, q, consorcioID, liquidacionID)
}
