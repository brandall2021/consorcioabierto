package expensas

import (
	"errors"
	"math"
	"sort"
)

var (
	ErrCalculoVacio       = errors.New("no hay conceptos ni UFs para distribuir")
	ErrCalculoSinUFs      = errors.New("no hay unidades funcionales")
	ErrCalculoSinCoefCero = errors.New("ninguna UF tiene coeficiente > 0")
)

type ConceptoCalculo struct {
	ConceptoID   string
	Regla        string
	ImporteCents int64
}

type UFCoef struct {
	UnidadID    string
	Codigo      string
	Coeficiente float64
}

type ItemDistribucion struct {
	ConceptoID   string `json:"concepto_id"`
	Regla        string `json:"regla"`
	ImporteCents int64  `json:"importe_cents"`
}

type UnidadItemDistribucion struct {
	ConceptoID   string `json:"concepto_id"`
	ImporteCents int64  `json:"importe_cents"`
}

type UnidadDistribucion struct {
	UnidadID    string                   `json:"unidad_id"`
	Codigo      string                   `json:"codigo"`
	Coeficiente float64                  `json:"coeficiente"`
	TotalCents  int64                    `json:"total_cents"`
	Items       []UnidadItemDistribucion `json:"items"`
}

type DistribucionResult struct {
	Items                 []ItemDistribucion   `json:"items"`
	Unidades              []UnidadDistribucion `json:"unidades"`
	TotalGastosCents      int64                `json:"total_gastos_cents"`
	TotalDistribuidoCents int64                `json:"total_distribuido_cents"`
	DiferenciaCents       int64                `json:"diferencia_cents"`
	UnidadesAlcanzadas    int                  `json:"unidades_alcanzadas"`
	Advertencias          []string             `json:"advertencias"`
}

// Distribuir reparte importes entre UFs por coeficiente con mayor resto
// y tie-break por código UF ascendente (ADR-0006).
func Distribuir(conceptos []ConceptoCalculo, ufs []UFCoef) (DistribucionResult, error) {
	if len(conceptos) == 0 {
		return DistribucionResult{}, ErrCalculoVacio
	}
	if len(ufs) == 0 {
		return DistribucionResult{}, ErrCalculoSinUFs
	}

	// Preparar UFs ordenadas por código (para tie-break determinista)
	sortedUFs := make([]UFCoef, len(ufs))
	copy(sortedUFs, ufs)
	sort.Slice(sortedUFs, func(i, j int) bool {
		return sortedUFs[i].Codigo < sortedUFs[j].Codigo
	})

	// Calcular coeficiente total
	coefTotal := 0.0
	for _, uf := range sortedUFs {
		coefTotal += uf.Coeficiente
	}
	if coefTotal <= 0 {
		return DistribucionResult{}, ErrCalculoSinCoefCero
	}

	var totalGastos int64
	var advertencias []string

	// Inicializar unidades
	unidades := make([]UnidadDistribucion, len(sortedUFs))
	for i, uf := range sortedUFs {
		unidades[i] = UnidadDistribucion{
			UnidadID:    uf.UnidadID,
			Codigo:      uf.Codigo,
			Coeficiente: uf.Coeficiente,
			Items:       make([]UnidadItemDistribucion, 0),
		}
		if uf.Coeficiente == 0 {
			advertencias = append(advertencias, "UF "+uf.Codigo+" tiene coeficiente 0, sin reparto")
		}
	}

	// Distribuir cada concepto
	items := make([]ItemDistribucion, 0, len(conceptos))
	for _, c := range conceptos {
		totalGastos += c.ImporteCents
		items = append(items, ItemDistribucion(c))

		// Distribuir por mayor resto
		remain := c.ImporteCents
		for i := range sortedUFs {
			if coefTotal == 0 || sortedUFs[i].Coeficiente == 0 {
				continue
			}
			exact := float64(c.ImporteCents) * (sortedUFs[i].Coeficiente / coefTotal)
			rounded := int64(math.Round(exact))
			if rounded < 0 {
				rounded = 0
			}
			if rounded > remain {
				rounded = remain
			}
			unidades[i].TotalCents += rounded
			remain -= rounded
			unidades[i].Items = append(unidades[i].Items, UnidadItemDistribucion{
				ConceptoID:   c.ConceptoID,
				ImporteCents: rounded,
			})
		}
		// Residuo va a la primera UF con coeficiente > 0 (ya ordenada por código)
		for i := range sortedUFs {
			if remain <= 0 {
				break
			}
			if sortedUFs[i].Coeficiente > 0 {
				unidades[i].TotalCents += remain
				// El item del concepto actual es el último agregado a esta unidad
				last := len(unidades[i].Items) - 1
				unidades[i].Items[last].ImporteCents += remain
				remain = 0
			}
		}
	}

	var totalDistribuido int64
	unidadesAlcanzadas := 0
	for _, u := range unidades {
		totalDistribuido += u.TotalCents
		if u.TotalCents > 0 {
			unidadesAlcanzadas++
		}
	}

	return DistribucionResult{
		Items:                 items,
		Unidades:              unidades,
		TotalGastosCents:      totalGastos,
		TotalDistribuidoCents: totalDistribuido,
		DiferenciaCents:       totalGastos - totalDistribuido,
		UnidadesAlcanzadas:    unidadesAlcanzadas,
		Advertencias:          advertencias,
	}, nil
}
