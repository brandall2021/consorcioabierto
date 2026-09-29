package cobranzas

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

type CuentaCorrienteMovimientoDTO struct {
	ID            string    `json:"id"`
	UnidadID      string    `json:"unidad_id"`
	Tipo          string    `json:"tipo"`
	FechaEfectiva string    `json:"fecha_efectiva"`
	DebitCents    int64     `json:"debit_cents"`
	CreditCents   int64     `json:"credit_cents"`
	Currency      string    `json:"currency"`
	Referencia    *string   `json:"referencia"`
	CreatedAt     time.Time `json:"-"`
}

type CuentaCorrienteMetaDTO struct {
	RequestID  string `json:"request_id"`
	SaldoCents int64  `json:"saldo_cents"`
}

type CuentaCorrienteDTO struct {
	Data []CuentaCorrienteMovimientoDTO `json:"data"`
	Meta CuentaCorrienteMetaDTO         `json:"meta"`
}

func GetCuentaCorriente(ctx context.Context, q *db.Queries, consorcioID, unidadID string) (CuentaCorrienteDTO, error) {
	var cid, uid pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return CuentaCorrienteDTO{}, ErrCobranzaInvalid
	}
	if err := uid.Scan(strings.TrimSpace(unidadID)); err != nil {
		return CuentaCorrienteDTO{}, ErrCobranzaInvalid
	}

	charges, err := q.ListChargesByUnidad(ctx, uid)
	if err != nil {
		return CuentaCorrienteDTO{}, err
	}
	payments, err := q.ListCobranzas(ctx, cid)
	if err != nil {
		return CuentaCorrienteDTO{}, err
	}

	movements := make([]CuentaCorrienteMovimientoDTO, 0, len(charges)+len(payments))
	for _, charge := range charges {
		ref := charge.Concepto
		movements = append(movements, CuentaCorrienteMovimientoDTO{
			ID:            charge.ID.String(),
			UnidadID:      charge.UnidadID.String(),
			Tipo:          "cargo",
			FechaEfectiva: charge.DueDate.Time.Format("2006-01-02"),
			DebitCents:    charge.TotalCents,
			CreditCents:   0,
			Currency:      "ARS",
			Referencia:    &ref,
			CreatedAt:     charge.CreatedAt.Time,
		})
	}
	for _, payment := range payments {
		if payment.UnidadID.String() != uid.String() || payment.Estado != "acreditado" {
			continue
		}
		ref := payment.Referencia.String
		if !payment.Referencia.Valid {
			ref = payment.Canal
		}
		movements = append(movements, CuentaCorrienteMovimientoDTO{
			ID:            payment.ID.String(),
			UnidadID:      payment.UnidadID.String(),
			Tipo:          "cobranza",
			FechaEfectiva: payment.Fecha.Time.Format("2006-01-02"),
			DebitCents:    0,
			CreditCents:   payment.ImporteCents,
			Currency:      "ARS",
			Referencia:    &ref,
			CreatedAt:     payment.CreatedAt.Time,
		})
	}

	sort.SliceStable(movements, func(i, j int) bool {
		if movements[i].FechaEfectiva != movements[j].FechaEfectiva {
			return movements[i].FechaEfectiva < movements[j].FechaEfectiva
		}
		if !movements[i].CreatedAt.Equal(movements[j].CreatedAt) {
			return movements[i].CreatedAt.Before(movements[j].CreatedAt)
		}
		return movements[i].ID < movements[j].ID
	})

	var saldo int64
	for i := range movements {
		saldo += movements[i].DebitCents - movements[i].CreditCents
	}

	for i := range movements {
		movements[i].CreatedAt = time.Time{}
	}

	return CuentaCorrienteDTO{
		Data: movements,
		Meta: CuentaCorrienteMetaDTO{SaldoCents: saldo},
	}, nil
}

func BuildReciboPDF(detalle AcreditacionDTO) []byte {
	lines := []string{
		"ConsorcioAbierto",
		"Recibo de cobranza",
		fmt.Sprintf("Cobranza: %s", detalle.Cobranza.ID),
		fmt.Sprintf("Unidad: %s", detalle.Cobranza.UnidadID),
		fmt.Sprintf("Fecha: %s", detalle.Cobranza.Fecha),
		fmt.Sprintf("Canal: %s", detalle.Cobranza.Canal),
		fmt.Sprintf("Importe: %s", moneyToString(detalle.Cobranza.Importe.AmountCents)),
	}
	for _, a := range detalle.Asignaciones {
		lines = append(lines, fmt.Sprintf("Asignacion %s -> %s", a.ChargeID, moneyToString(a.AmountCents)))
	}
	lines = append(lines, fmt.Sprintf("Saldo a favor: %s", moneyToString(detalle.SaldoAFavorCents)))
	return buildSimplePDF(lines)
}

func moneyToString(cents int64) string {
	if cents < 0 {
		return fmt.Sprintf("-ARS %d.%02d", -cents/100, -cents%100)
	}
	return fmt.Sprintf("ARS %d.%02d", cents/100, cents%100)
}

func buildSimplePDF(lines []string) []byte {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 12 Tf\n72 760 Td\n")
	for i, line := range lines {
		if i > 0 {
			content.WriteString("0 -16 Td\n")
		}
		content.WriteString("(")
		content.WriteString(escapePDFText(line))
		content.WriteString(") Tj\n")
	}
	content.WriteString("ET\n")

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()),
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefStart := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefStart)
	return out.Bytes()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
