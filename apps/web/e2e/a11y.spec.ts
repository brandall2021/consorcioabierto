import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { CONSORCIO_ID, LIQUIDACION_ID, stubAnonymous, stubAuthenticated } from './support/api'

const WCAG_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa']
const C = `/app/consorcios/${CONSORCIO_ID}`

async function expectNoViolations(page: Page, name: string) {
  const { violations } = await new AxeBuilder({ page }).withTags(WCAG_TAGS).analyze()
  const summary = violations
    .map((v) => `  - [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} nodo/s)`)
    .join('\n')
  expect(
    violations,
    violations.length ? `Violaciones WCAG 2.2 AA en ${name}:\n${summary}` : undefined,
  ).toEqual([])
}

test.describe('Accesibilidad (WCAG 2.2 AA)', () => {
  test('login (anónimo)', async ({ page }) => {
    await stubAnonymous(page)
    await page.goto('/login')
    await expect(page.getByRole('button', { name: 'Ingresar' })).toBeVisible()
    await expectNoViolations(page, '/login')
  })

  test('dashboard', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto('/app')
    await expect(page.getByRole('heading', { name: 'Inicio', level: 1 })).toBeVisible()
    await expectNoViolations(page, '/app')
  })

  test('consorcios', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto('/app/consorcios')
    await expect(page.getByRole('heading', { name: 'Consorcios', level: 1 })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Consorcio Demo' })).toBeVisible()
    await expectNoViolations(page, '/app/consorcios')
  })

  test('unidades', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/unidades`)
    await expect(page.getByRole('heading', { name: 'Unidades funcionales', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/unidades`)
  })

  test('proveedores', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/proveedores`)
    await expect(page.getByRole('heading', { name: 'Proveedores', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/proveedores`)
  })

  test('liquidaciones', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/liquidaciones`)
    await expect(page.getByRole('heading', { name: 'Liquidaciones', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/liquidaciones`)
  })

  test('liquidación (wizard)', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/liquidaciones/${LIQUIDACION_ID}`)
    await expect(page.getByRole('heading', { name: 'Liquidación', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/liquidaciones/:id`)
  })

  test('reclamos', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/reclamos`)
    await expect(page.getByRole('heading', { name: 'Reclamos', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/reclamos`)
  })

  test('cuenta corriente', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/cuenta-corriente`)
    await expect(page.getByRole('heading', { name: 'Cuenta corriente', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/cuenta-corriente`)
  })

  test('morosidad', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto(`${C}/morosidad`)
    await expect(page.getByRole('heading', { name: 'Morosidad', level: 1 })).toBeVisible()
    await expectNoViolations(page, `${C}/morosidad`)
  })

  test('auditoría', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto('/app/auditoria')
    await expect(page.getByRole('heading', { name: 'Auditoría', level: 1 })).toBeVisible()
    await expectNoViolations(page, '/app/auditoria')
  })

  test('perfil', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto('/app/perfil')
    await expect(page.getByRole('heading', { name: 'Perfil', level: 1 })).toBeVisible()
    await expectNoViolations(page, '/app/perfil')
  })

  test('portal', async ({ page }) => {
    await stubAuthenticated(page)
    await page.goto('/portal')
    await expect(page.getByRole('heading', { name: 'Portal consorcista', level: 1 })).toBeVisible()
    await expectNoViolations(page, '/portal')
  })
})
