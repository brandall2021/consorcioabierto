package portal

import (
	"context"
	"sort"
	"time"

	db "github.com/brandall2021/consorcioabierto/internal/database/gen"
	"github.com/jackc/pgx/v5/pgtype"
)

type PortalConsorcioSummary struct {
	ID                string `json:"id"`
	Nombre            string `json:"nombre"`
	Estado            string `json:"estado"`
	SaldoVencidoCents int64  `json:"saldo_vencido_cents"`
	CargosVencidos    int64  `json:"cargos_vencidos"`
	VencidoDesde      string `json:"vencido_desde"`
}

type PortalReceiptSummary struct {
	ConsorcioID      string    `json:"consorcio_id"`
	ConsorcioNombre  string    `json:"consorcio_nombre"`
	CobranzaID       string    `json:"cobranza_id"`
	Fecha            string    `json:"fecha"`
	ImporteCents     int64     `json:"importe_cents"`
	Estado           string    `json:"estado"`
	SaldoAFavorCents int64     `json:"saldo_a_favor_cents"`
	CreatedAt        time.Time `json:"created_at"`
}

type PortalComunicadoSummary struct {
	ID              string    `json:"id"`
	ConsorcioID     string    `json:"consorcio_id"`
	ConsorcioNombre string    `json:"consorcio_nombre"`
	Titulo          string    `json:"titulo"`
	Estado          string    `json:"estado"`
	PublicadoAt     time.Time `json:"publicado_at"`
}

type PortalReclamoSummary struct {
	ID          string    `json:"id"`
	ConsorcioID string    `json:"consorcio_id"`
	UnidadID    string    `json:"unidad_id"`
	Categoria   string    `json:"categoria"`
	Estado      string    `json:"estado"`
	Texto       string    `json:"texto"`
	CreatedAt   time.Time `json:"created_at"`
}

type PortalHome struct {
	Consorcios             []PortalConsorcioSummary  `json:"consorcios"`
	RecibosRecientes       []PortalReceiptSummary    `json:"recibos_recientes"`
	ComunicadosRecientes   []PortalComunicadoSummary `json:"comunicados_recientes"`
	ReclamosRecientes      []PortalReclamoSummary    `json:"reclamos_recientes"`
	TotalSaldoVencidoCents int64                     `json:"total_saldo_vencido_cents"`
}

type Queryer interface {
	ListConsorcios(context.Context, db.ListConsorciosParams) ([]db.Consorcio, error)
	ListUnidades(context.Context, db.ListUnidadesParams) ([]db.Unidade, error)
	ListUnidadesForCurrentUser(context.Context) ([]db.Unidade, error)
	ListOpenChargesByUnidad(context.Context, db.ListOpenChargesByUnidadParams) ([]db.Charge, error)
	ListComunicados(context.Context, db.ListComunicadosParams) ([]db.Comunicado, error)
	ListReclamos(context.Context, db.ListReclamosParams) ([]db.Reclamo, error)
	ListCobranzas(context.Context, pgtype.UUID) ([]db.Payment, error)
	ListPaymentAllocationsByPayment(context.Context, pgtype.UUID) ([]db.PaymentAllocation, error)
}

func BuildHome(ctx context.Context, q Queryer) (PortalHome, error) {
	consorcios, err := q.ListConsorcios(ctx, db.ListConsorciosParams{})
	if err != nil {
		return PortalHome{}, err
	}

	home := PortalHome{
		Consorcios:           []PortalConsorcioSummary{},
		RecibosRecientes:     []PortalReceiptSummary{},
		ComunicadosRecientes: []PortalComunicadoSummary{},
		ReclamosRecientes:    []PortalReclamoSummary{},
	}
	// El portal es del consorcista: su alcance son las UFs con vinculo vigente.
	// Antes se iteraban todos los consorcios y todas las unidades del tenant,
	// por lo que cada consorcista veia reclamos y deuda de unidades ajenas.
	misUnidades, err := q.ListUnidadesForCurrentUser(ctx)
	if err != nil {
		return PortalHome{}, err
	}
	unitIDs := make(map[string]struct{}, len(misUnidades))
	unitsByConsorcio := make(map[string][]db.Unidade, len(misUnidades))
	consorciosDeUsuario := make([]pgtype.UUID, 0, len(misUnidades))
	seenConsorcio := make(map[string]struct{}, len(misUnidades))
	for _, unit := range misUnidades {
		unitIDs[unit.ID.String()] = struct{}{}
		consorcioKey := unit.ConsorcioID.String()
		unitsByConsorcio[consorcioKey] = append(unitsByConsorcio[consorcioKey], unit)
		if _, ok := seenConsorcio[consorcioKey]; !ok {
			seenConsorcio[consorcioKey] = struct{}{}
			consorciosDeUsuario = append(consorciosDeUsuario, unit.ConsorcioID)
		}
	}
	if len(misUnidades) == 0 {
		return home, nil
	}

	consorciosPorID := make(map[string]db.Consorcio, len(consorcios))
	for _, consorcio := range consorcios {
		consorciosPorID[consorcio.ID.String()] = consorcio
	}
	cutoff := todayUTC()
	for _, consorcioID := range consorciosDeUsuario {
		consorcio, ok := consorciosPorID[consorcioID.String()]
		if !ok {
			continue
		}
		units := unitsByConsorcio[consorcioID.String()]

		var saldoVencido int64
		var cargosVencidos int64
		var vencidoDesde string
		for _, unit := range units {
			charges, err := q.ListOpenChargesByUnidad(ctx, db.ListOpenChargesByUnidadParams{UnidadID: unit.ID, FechaCorte: cutoff})
			if err != nil {
				return PortalHome{}, err
			}
			for _, charge := range charges {
				saldoVencido += charge.SaldoCents
				cargosVencidos++
				if vencidoDesde == "" || charge.DueDate.Time.Format("2006-01-02") < vencidoDesde {
					vencidoDesde = charge.DueDate.Time.Format("2006-01-02")
				}
			}
		}

		home.Consorcios = append(home.Consorcios, PortalConsorcioSummary{
			ID:                consorcio.ID.String(),
			Nombre:            consorcio.Nombre,
			Estado:            consorcio.Estado,
			SaldoVencidoCents: saldoVencido,
			CargosVencidos:    cargosVencidos,
			VencidoDesde:      vencidoDesde,
		})
		home.TotalSaldoVencidoCents += saldoVencido

		payments, err := q.ListCobranzas(ctx, consorcio.ID)
		if err != nil {
			return PortalHome{}, err
		}
		for _, payment := range payments {
			if _, ok := unitIDs[payment.UnidadID.String()]; !ok {
				continue
			}
			if payment.Estado != "acreditado" {
				continue
			}
			allocs, err := q.ListPaymentAllocationsByPayment(ctx, payment.ID)
			if err != nil {
				return PortalHome{}, err
			}
			home.RecibosRecientes = append(home.RecibosRecientes, PortalReceiptSummary{
				ConsorcioID:      consorcio.ID.String(),
				ConsorcioNombre:  consorcio.Nombre,
				CobranzaID:       payment.ID.String(),
				Fecha:            payment.Fecha.Time.Format("2006-01-02"),
				ImporteCents:     payment.ImporteCents,
				Estado:           payment.Estado,
				SaldoAFavorCents: saldoAFavorFromAllocations(payment, allocs),
				CreatedAt:        payment.CreatedAt.Time,
			})
		}

		comunicados, err := q.ListComunicados(ctx, db.ListComunicadosParams{ConsorcioID: consorcio.ID})
		if err != nil {
			return PortalHome{}, err
		}
		for _, comunicado := range comunicados {
			if comunicado.Estado != "publicado" || !comunicado.PublicadoAt.Valid {
				continue
			}
			home.ComunicadosRecientes = append(home.ComunicadosRecientes, PortalComunicadoSummary{
				ID:              comunicado.ID.String(),
				ConsorcioID:     consorcio.ID.String(),
				ConsorcioNombre: consorcio.Nombre,
				Titulo:          comunicado.Titulo,
				Estado:          comunicado.Estado,
				PublicadoAt:     comunicado.PublicadoAt.Time,
			})
		}

		reclamos, err := q.ListReclamos(ctx, db.ListReclamosParams{ConsorcioID: consorcio.ID})
		if err != nil {
			return PortalHome{}, err
		}
		for _, reclamo := range reclamos {
			if _, ok := unitIDs[reclamo.UnidadID.String()]; !ok {
				continue
			}
			home.ReclamosRecientes = append(home.ReclamosRecientes, PortalReclamoSummary{
				ID:          reclamo.ID.String(),
				ConsorcioID: consorcio.ID.String(),
				UnidadID:    reclamo.UnidadID.String(),
				Categoria:   reclamo.Categoria,
				Estado:      reclamo.Estado,
				Texto:       reclamo.Texto,
				CreatedAt:   reclamo.CreatedAt.Time,
			})
		}
	}

	sort.SliceStable(home.RecibosRecientes, func(i, j int) bool {
		return home.RecibosRecientes[i].CreatedAt.After(home.RecibosRecientes[j].CreatedAt)
	})
	sort.SliceStable(home.ComunicadosRecientes, func(i, j int) bool {
		return home.ComunicadosRecientes[i].PublicadoAt.After(home.ComunicadosRecientes[j].PublicadoAt)
	})
	sort.SliceStable(home.ReclamosRecientes, func(i, j int) bool {
		return home.ReclamosRecientes[i].CreatedAt.After(home.ReclamosRecientes[j].CreatedAt)
	})
	if len(home.RecibosRecientes) > 5 {
		home.RecibosRecientes = home.RecibosRecientes[:5]
	}
	if len(home.ComunicadosRecientes) > 5 {
		home.ComunicadosRecientes = home.ComunicadosRecientes[:5]
	}
	if len(home.ReclamosRecientes) > 5 {
		home.ReclamosRecientes = home.ReclamosRecientes[:5]
	}

	return home, nil
}

func todayUTC() pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(time.Now().UTC().Format("2006-01-02"))
	return d
}

func saldoAFavorFromAllocations(payment db.Payment, allocs []db.PaymentAllocation) int64 {
	var allocated int64
	for _, alloc := range allocs {
		allocated += alloc.AmountCents
	}
	remaining := payment.ImporteCents - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}
