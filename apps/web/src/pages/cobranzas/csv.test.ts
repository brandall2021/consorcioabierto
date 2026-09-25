import { describe, expect, it } from 'vitest'
import { parseCobranzasCsv } from './csv'

describe('parseCobranzasCsv', () => {
	it('parsea un archivo simple', () => {
		const rows = parseCobranzasCsv(`codigo_unidad,fecha,canal,amount_cents,referencia
1A,2026-09-08,transferencia,12500,TRX-1
1B,2026-09-08,,5000,
`)
		expect(rows).toHaveLength(2)
		expect(rows[0]).toEqual({
			codigo_unidad: '1A',
			fecha: '2026-09-08',
			canal: 'transferencia',
			amount_cents: 12500,
			referencia: 'TRX-1',
		})
		expect(rows[1]).toEqual({
			codigo_unidad: '1B',
			fecha: '2026-09-08',
			canal: undefined,
			amount_cents: 5000,
			referencia: undefined,
		})
	})
})
