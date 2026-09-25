export type CobranzaCsvRow = {
	codigo_unidad: string
	fecha: string
	canal?: string
	amount_cents: number
	referencia?: string
}

const expectedHeader = ['codigo_unidad', 'fecha', 'canal', 'amount_cents', 'referencia']

export function parseCobranzasCsv(text: string): CobranzaCsvRow[] {
	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/).filter((line) => line.trim() !== '')
	if (lines.length < 2) {
		throw new Error('El CSV debe tener encabezado y al menos una fila.')
	}
	const header = parseCsvLine(lines[0])
	if (header.length !== expectedHeader.length || header.some((value, i) => value !== expectedHeader[i])) {
		throw new Error('El encabezado debe ser codigo_unidad,fecha,canal,amount_cents,referencia.')
	}

	return lines.slice(1).map((line, index) => {
		const cols = parseCsvLine(line)
		if (cols.length !== expectedHeader.length) {
			throw new Error(`La fila ${index + 2} no tiene 5 columnas.`)
		}
		const amount = Number.parseInt(cols[3], 10)
		if (!Number.isFinite(amount) || amount <= 0) {
			throw new Error(`La fila ${index + 2} tiene amount_cents inválido.`)
		}
		return {
			codigo_unidad: cols[0].trim(),
			fecha: cols[1].trim(),
			canal: cols[2].trim() || undefined,
			amount_cents: amount,
			referencia: cols[4].trim() || undefined,
		}
	})
}

function parseCsvLine(line: string): string[] {
	const out: string[] = []
	let current = ''
	let quoted = false
	for (let i = 0; i < line.length; i += 1) {
		const ch = line[i]
		if (quoted) {
			if (ch === '"' && line[i + 1] === '"') {
				current += '"'
				i += 1
				continue
			}
			if (ch === '"') {
				quoted = false
				continue
			}
			current += ch
			continue
		}
		if (ch === ',') {
			out.push(current)
			current = ''
			continue
		}
		if (ch === '"') {
			quoted = true
			continue
		}
		current += ch
	}
	out.push(current)
	return out
}
