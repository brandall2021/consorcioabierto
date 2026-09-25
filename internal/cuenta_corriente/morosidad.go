package cuenta_corriente

import (
	"context"
	"strings"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

type MorosidadDTO struct {
	UnidadID          string `json:"unidad_id"`
	UnidadCodigo      string `json:"unidad_codigo"`
	SaldoVencidoCents int64  `json:"saldo_vencido_cents"`
	CantidadCargos    int64  `json:"cantidad_cargos"`
	VencidoDesde      string `json:"vencido_desde"`
}

func ListMorosidadByConsorcio(ctx context.Context, q *db.Queries, consorcioID string) ([]MorosidadDTO, error) {
	var cid pgtype.UUID
	if err := cid.Scan(strings.TrimSpace(consorcioID)); err != nil {
		return nil, ErrCargoInvalido
	}

	rows, err := q.ListMorosidadByConsorcio(ctx, cid)
	if err != nil {
		return nil, err
	}
	out := make([]MorosidadDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, morosidadDTO(row))
	}
	return out, nil
}

func morosidadDTO(row db.ListMorosidadByConsorcioRow) MorosidadDTO {
	return MorosidadDTO{
		UnidadID:          row.UnidadID.String(),
		UnidadCodigo:      row.UnidadCodigo,
		SaldoVencidoCents: row.SaldoVencidoCents,
		CantidadCargos:    row.CantidadCargos,
		VencidoDesde:      formatMorosidadDate(row.VencidoDesde),
	}
}

func formatMorosidadDate(v any) string {
	switch t := v.(type) {
	case time.Time:
		return t.Format("2006-01-02")
	case pgtype.Date:
		if t.Valid {
			return t.Time.Format("2006-01-02")
		}
		return ""
	default:
		return ""
	}
}
