import { describe, expect, it } from 'vitest'
import { parseCobranzasCsv } from './csv'

describe('parseCobranzasCsv', () => {
	it('parsea filas simples', () => {
		const rows = parseCobranzasCsv(`codigo_unidad,fecha,canal,amount_cents,referencia
1A,2026-09-28,efectivo,15000,REC-1
1B,2026-09-28,,20000,
`)
		expect(rows).toEqual([
			{ codigo_unidad: '1A', fecha: '2026-09-28', canal: 'efectivo', amount_cents: 15000, referencia: 'REC-1' },
			{ codigo_unidad: '1B', fecha: '2026-09-28', canal: undefined, amount_cents: 20000, referencia: undefined },
		])
	})

	it('soporta comillas y comas en campos', () => {
		const rows = parseCobranzasCsv(`codigo_unidad,fecha,canal,amount_cents,referencia
"1, A",2026-09-28,otros,15000,"Pago, septiembre"
`)
		expect(rows[0]).toEqual({
			codigo_unidad: '1, A',
			fecha: '2026-09-28',
			canal: 'otros',
			amount_cents: 15000,
			referencia: 'Pago, septiembre',
		})
	})
})
