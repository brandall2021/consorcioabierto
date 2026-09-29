export type CobranzaCsvRow = {
	codigo_unidad: string
	fecha: string
	canal?: string
	amount_cents: number
	referencia?: string
}

const expectedHeader = ['codigo_unidad', 'fecha', 'canal', 'amount_cents', 'referencia']

export function parseCobranzasCsv(input: string): CobranzaCsvRow[] {
	const lines = input
		.split(/\r?\n/)
		.map((line) => line.trim())
		.filter(Boolean)

	if (lines.length < 2) {
		throw new Error('El CSV debe incluir encabezado y al menos una fila')
	}

	const header = parseCsvLine(lines[0]).map((col) => col.trim())
	if (header.length !== expectedHeader.length || !header.every((col, index) => col === expectedHeader[index])) {
		throw new Error(`Encabezado inválido. Se esperaba: ${expectedHeader.join(',')}`)
	}

	return lines.slice(1).map((line, index) => {
		const cols = parseCsvLine(line)
		if (cols.length !== expectedHeader.length) {
			throw new Error(`Fila ${index + 2}: cantidad de columnas inválida`)
		}
		const amountCents = Number.parseInt(cols[3]?.trim() ?? '', 10)
		if (!Number.isFinite(amountCents)) {
			throw new Error(`Fila ${index + 2}: amount_cents inválido`)
		}
		return {
			codigo_unidad: cols[0].trim(),
			fecha: cols[1].trim(),
			canal: normalizeOptional(cols[2]),
			amount_cents: amountCents,
			referencia: normalizeOptional(cols[4]),
		}
	})
}

function normalizeOptional(value: string): string | undefined {
	const trimmed = value.trim()
	return trimmed === '' ? undefined : trimmed
}

function parseCsvLine(line: string): string[] {
	const values: string[] = []
	let current = ''
	let inQuotes = false

	for (let i = 0; i < line.length; i++) {
		const char = line[i]
		if (char === '"') {
			if (inQuotes && line[i + 1] === '"') {
				current += '"'
				i++
				continue
			}
			inQuotes = !inQuotes
			continue
		}
		if (char === ',' && !inQuotes) {
			values.push(current)
			current = ''
			continue
		}
		current += char
	}

	values.push(current)
	return values
}
