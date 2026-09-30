import { describe, expect, it } from 'vitest'
import {
	accionesDisponibles,
	ESTADOS,
	etiquetaEstado,
	requiereMotivo,
	transicionValida,
	type Accion,
	type EstadoReclamo,
} from './estados'

describe('transicionValida', () => {
	it('permite avanzar un reclamo abierto', () => {
		expect(transicionValida('abierto', 'en_progreso')).toBe(true)
		expect(transicionValida('abierto', 'resolver')).toBe(true)
		expect(transicionValida('abierto', 'cerrar')).toBe(true)
	})

	it('no permite reabrir desde abierto', () => {
		expect(transicionValida('abierto', 'reabrir')).toBe(false)
	})

	it('no permite resolver ni avanzar desde resuelto', () => {
		expect(transicionValida('resuelto', 'resolver')).toBe(false)
		expect(transicionValida('resuelto', 'en_progreso')).toBe(false)
	})

	it('solo reabrir y cerrar son posibles desde resuelto', () => {
		expect(transicionValida('resuelto', 'cerrar')).toBe(true)
		expect(transicionValida('resuelto', 'reabrir')).toBe(true)
	})

	it('desde cerrado solo se puede reabrir', () => {
		expect(transicionValida('cerrado', 'reabrir')).toBe(true)
		expect(transicionValida('cerrado', 'resolver')).toBe(false)
		expect(transicionValida('cerrado', 'cerrar')).toBe(false)
	})
})

describe('requiereMotivo', () => {
	it('solo reabrir exige motivo', () => {
		expect(requiereMotivo('reabrir')).toBe(true)
		expect(requiereMotivo('resolver')).toBe(false)
		expect(requiereMotivo('cerrar')).toBe(false)
		expect(requiereMotivo('en_progreso')).toBe(false)
	})

	it('reabrir solo pide motivo cuando el estado lo permite', () => {
		expect(requiereMotivo('reabrir', 'cerrado')).toBe(true)
		expect(requiereMotivo('reabrir', 'resuelto')).toBe(true)
		expect(requiereMotivo('reabrir', 'abierto')).toBe(false)
	})
})

describe('accionesDisponibles', () => {
	it('ofrece asignar solo mientras el reclamo esta abierto o en progreso', () => {
		const acciones = (estado: EstadoReclamo) =>
			accionesDisponibles(estado).map((a) => a.accion)

		expect(acciones('abierto')).toContain('asignar')
		expect(acciones('en_progreso')).toContain('asignar')
		expect(acciones('resuelto')).not.toContain('asignar')
		expect(acciones('cerrado')).not.toContain('asignar')
	})

	it('no ofrece transiciones invalidas desde un estado terminal', () => {
		expect(accionesDisponibles('cerrado').map((a) => a.accion)).toEqual(['reabrir'])
	})

	it('marca asignar como la unica accion que pide responsable', () => {
		const asignar = accionesDisponibles('abierto').find((a) => a.accion === 'asignar')
		const cerrar = accionesDisponibles('abierto').find((a) => a.accion === 'cerrar')

		expect(asignar?.requiereResponsable).toBe(true)
		expect(cerrar?.requiereResponsable).toBe(false)
	})

	it('un estado desconocido no ofrece acciones', () => {
		expect(accionesDisponibles('inventado' as EstadoReclamo)).toEqual([])
	})
})

describe('etiquetaEstado', () => {
	it('traduce los cuatro estados', () => {
		expect(etiquetaEstado('abierto')).toBe('Abierto')
		expect(etiquetaEstado('en_progreso')).toBe('En progreso')
		expect(etiquetaEstado('resuelto')).toBe('Resuelto')
		expect(etiquetaEstado('cerrado')).toBe('Cerrado')
	})

	it('devuelve el estado crudo si no lo conoce', () => {
		expect(etiquetaEstado('raro' as EstadoReclamo)).toBe('raro')
	})
})

describe('tipos exported', () => {
	it('ESTADOS y Accion quedan alineados con el contrato', () => {
		expect(ESTADOS).toEqual(['abierto', 'en_progreso', 'resuelto', 'cerrado'])
		const acciones: Accion[] = ['asignar', 'en_progreso', 'resolver', 'cerrar', 'reabrir']
		expect(acciones).toHaveLength(5)
	})
})
