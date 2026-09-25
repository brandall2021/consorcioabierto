package portal

import (
	"context"
	"sort"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/consorcios"
	cc "github.com/brandall2021/consorcioabierto/internal/cuenta_corriente"
	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

type TenantConsorcioDTO struct {
	ID                string `json:"id"`
	Nombre            string `json:"nombre"`
	Estado            string `json:"estado"`
	SaldoVencidoCents int64  `json:"saldo_vencido_cents"`
	CargosVencidos    int64  `json:"cargos_vencidos"`
	VencidoDesde      string `json:"vencido_desde"`
}

type TenantReceiptDTO struct {
	ConsorcioID      string `json:"consorcio_id"`
	ConsorcioNombre  string `json:"consorcio_nombre"`
	CobranzaID       string `json:"cobranza_id"`
	Fecha            string `json:"fecha"`
	ImporteCents     int64  `json:"importe_cents"`
	Estado           string `json:"estado"`
	SaldoAFavorCents int64  `json:"saldo_a_favor_cents"`
	CreatedAtISO     string `json:"created_at"`
}

type TenantComunicadoDTO struct {
	ID           string `json:"id"`
	ConsorcioID  string `json:"consorcio_id"`
	ConsorcioNombre string `json:"consorcio_nombre"`
	Titulo       string `json:"titulo"`
	Estado       string `json:"estado"`
	PublicadoAt  string `json:"publicado_at"`
}

type TenantReclamoDTO struct {
	ID          string `json:"id"`
	ConsorcioID string `json:"consorcio_id"`
	UnidadID    string `json:"unidad_id"`
	Categoria   string `json:"categoria"`
	Estado      string `json:"estado"`
	Texto       string `json:"texto"`
	CreatedAt   string `json:"created_at"`
}

type HomeDTO struct {
	Consorcios             []TenantConsorcioDTO   `json:"consorcios"`
	RecibosRecientes       []TenantReceiptDTO     `json:"recibos_recientes"`
	ComunicadosRecientes   []TenantComunicadoDTO  `json:"comunicados_recientes"`
	ReclamosRecientes      []TenantReclamoDTO     `json:"reclamos_recientes"`
	TotalSaldoVencidoCents int64                  `json:"total_saldo_vencido_cents"`
}

func BuildTenantHome(ctx context.Context, q *db.Queries) (HomeDTO, error) {
	items, err := consorcios.List(ctx, q, consorcios.Filter{})
	if err != nil {
		return HomeDTO{}, err
	}

	home := HomeDTO{Consorcios: make([]TenantConsorcioDTO, 0, len(items))}
	var receipts []TenantReceiptDTO
	var comunicados []TenantComunicadoDTO
	var reclamos []TenantReclamoDTO
	for _, c := range items {
		morosidad, err := cc.ListMorosidadByConsorcio(ctx, q, c.ID)
		if err != nil {
			return HomeDTO{}, err
		}
		var saldo int64
		var cargos int64
		var vencidoDesde string
		for _, m := range morosidad {
			saldo += m.SaldoVencidoCents
			cargos += m.CantidadCargos
			if vencidoDesde == "" || (m.VencidoDesde != "" && m.VencidoDesde < vencidoDesde) {
				vencidoDesde = m.VencidoDesde
			}
		}
		home.TotalSaldoVencidoCents += saldo
		home.Consorcios = append(home.Consorcios, TenantConsorcioDTO{
			ID:                c.ID,
			Nombre:            c.Nombre,
			Estado:            c.Estado,
			SaldoVencidoCents: saldo,
			CargosVencidos:    cargos,
			VencidoDesde:      vencidoDesde,
		})

		var cid pgtype.UUID
		if err := cid.Scan(c.ID); err != nil {
			return HomeDTO{}, err
		}
		cobros, err := q.ListCobranzas(ctx, cid)
		if err != nil {
			return HomeDTO{}, err
		}
		for _, cob := range cobros {
			receipts = append(receipts, TenantReceiptDTO{
				ConsorcioID:      c.ID,
				ConsorcioNombre:  c.Nombre,
				CobranzaID:       cob.ID.String(),
				Fecha:            paymentDate(cob.Fecha),
				ImporteCents:     cob.ImporteCents,
				Estado:           cob.Estado,
				SaldoAFavorCents: 0,
				CreatedAtISO:     paymentTime(cob.CreatedAt),
			})
		}

		pubs, err := q.ListComunicados(ctx, cid)
		if err != nil {
			return HomeDTO{}, err
		}
		for _, comm := range pubs {
			if comm.Estado != "publicado" {
				continue
			}
			comunicados = append(comunicados, TenantComunicadoDTO{
				ID:              comm.ID.String(),
				ConsorcioID:      c.ID,
				ConsorcioNombre:  c.Nombre,
				Titulo:          comm.Titulo,
				Estado:          comm.Estado,
				PublicadoAt:     timestampValue(comm.PublicadoAt, comm.CreatedAt),
			})
		}

		itemsReclamos, err := q.ListReclamos(ctx, cid)
		if err != nil {
			return HomeDTO{}, err
		}
		for _, rec := range itemsReclamos {
			reclamos = append(reclamos, TenantReclamoDTO{
				ID:          rec.ID.String(),
				ConsorcioID: c.ID,
				UnidadID:    rec.UnidadID.String(),
				Categoria:   rec.Categoria,
				Estado:      rec.Estado,
				Texto:       rec.Texto,
				CreatedAt:   paymentTime(rec.CreatedAt),
			})
		}
	}

	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].Fecha != receipts[j].Fecha {
			return receipts[i].Fecha > receipts[j].Fecha
		}
		return receipts[i].CreatedAtISO > receipts[j].CreatedAtISO
	})
	if len(receipts) > 5 {
		receipts = receipts[:5]
	}
	sort.Slice(comunicados, func(i, j int) bool {
		return comunicados[i].PublicadoAt > comunicados[j].PublicadoAt
	})
	if len(comunicados) > 5 {
		comunicados = comunicados[:5]
	}
	sort.Slice(reclamos, func(i, j int) bool {
		return reclamos[i].CreatedAt > reclamos[j].CreatedAt
	})
	if len(reclamos) > 5 {
		reclamos = reclamos[:5]
	}
	home.RecibosRecientes = receipts
	home.ComunicadosRecientes = comunicados
	home.ReclamosRecientes = reclamos
	return home, nil
}

func paymentDate(v pgtype.Date) string {
	if !v.Valid {
		return ""
	}
	return v.Time.Format("2006-01-02")
}

func paymentTime(v pgtype.Timestamptz) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339)
}

func timestampValue(primary, fallback pgtype.Timestamptz) string {
	if primary.Valid {
		return primary.Time.UTC().Format(time.RFC3339)
	}
	if fallback.Valid {
		return fallback.Time.UTC().Format(time.RFC3339)
	}
	return ""
}
