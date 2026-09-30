export type EstadoReclamo = 'abierto' | 'en_progreso' | 'resuelto' | 'cerrado'

export type Accion = 'asignar' | 'en_progreso' | 'resolver' | 'cerrar' | 'reabrir'

export const ESTADOS: EstadoReclamo[] = ['abierto', 'en_progreso', 'resuelto', 'cerrado']

const ETIQUETAS: Record<EstadoReclamo, string> = {
	abierto: 'Abierto',
	en_progreso: 'En progreso',
	resuelto: 'Resuelto',
	cerrado: 'Cerrado',
}

// Espejo de la tabla `transiciones` de internal/reclamos/reclamos.go. El backend
// sigue siendo la autoridad: esto solo evita ofrecer botones que darían 409.
const TRANSICIONES: Record<EstadoReclamo, Accion[]> = {
	abierto: ['en_progreso', 'resolver', 'cerrar'],
	en_progreso: ['resolver', 'cerrar'],
	resuelto: ['cerrar', 'reabrir'],
	cerrado: ['reabrir'],
}

// Espejo de `accionesReasignacion`: `asignar` no cambia el estado.
const REASIGNABLES: EstadoReclamo[] = ['abierto', 'en_progreso']

export function transicionValida(estado: EstadoReclamo, accion: Accion): boolean {
	if (accion === 'asignar') return REASIGNABLES.includes(estado)
	return TRANSICIONES[estado]?.includes(accion) ?? false
}

export function requiereMotivo(accion: Accion, estado?: EstadoReclamo): boolean {
	if (accion !== 'reabrir') return false
	return estado === undefined ? true : transicionValida(estado, accion)
}

export function etiquetaEstado(estado: EstadoReclamo): string {
	return ETIQUETAS[estado] ?? estado
}

export type AccionDisponible = {
	accion: Accion
	etiqueta: string
	requiereMotivo: boolean
	requiereResponsable: boolean
}

const ETIQUETAS_ACCION: Record<Accion, string> = {
	asignar: 'Asignar',
	en_progreso: 'Marcar en progreso',
	resolver: 'Resolver',
	cerrar: 'Cerrar',
	reabrir: 'Reabrir',
}

export function accionesDisponibles(estado: EstadoReclamo): AccionDisponible[] {
	const acciones: Accion[] = estado === 'abierto' || estado === 'en_progreso' ? ['asignar'] : []
	for (const accion of TRANSICIONES[estado] ?? []) acciones.push(accion)

	return acciones.map((accion) => ({
		accion,
		etiqueta: ETIQUETAS_ACCION[accion],
		requiereMotivo: requiereMotivo(accion, estado),
		requiereResponsable: accion === 'asignar',
	}))
}
