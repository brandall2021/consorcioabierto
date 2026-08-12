import { describe, expect, it } from 'vitest'
import { formatCents } from './format'

describe('formatCents', () => {
  it('formatea centavos en pesos con separador de miles', () => {
    expect(formatCents(123456)).toContain('1.234,56')
  })

  it('muestra montos negativos', () => {
    expect(formatCents(-100)).toContain('-')
  })

  it('muestra cero con dos decimales', () => {
    expect(formatCents(0)).toContain('0,00')
  })

  it('no trunca centavos', () => {
    expect(formatCents(123457)).toContain('1.234,57')
  })
})
