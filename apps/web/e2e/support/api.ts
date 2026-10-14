import type { Route } from '@playwright/test'
import type { Page } from '@playwright/test'

const UUID = '00000000-0000-0000-0000-000000000000'
const ISO = '2026-01-01T00:00:00Z'

export const CONSORCIO_ID = '11111111-1111-1111-1111-111111111111'
export const LIQUIDACION_ID = '22222222-2222-2222-2222-222222222222'

const meFixture = {
  user: { id: UUID, email: 'admin@tenant-a.com', name: 'Admin Tenant A', status: 'active', mfa_enabled: false },
  membership: {
    id: UUID,
    tenant_id: UUID,
    tenant_name: 'Consorcio Demo S.A.',
    tenant_status: 'active',
    roles: ['admin'],
    scopes: [],
  },
  permissions: [
    'consorcios.read',
    'consorcios.write',
    'auditoria.read',
    'ufs.manage',
    'liquidaciones.read',
    'reclamos.read',
    'cobranzas.read',
    'personas.read',
  ],
}

const consorcioFixture = {
  id: CONSORCIO_ID,
  tenant_id: UUID,
  nombre: 'Consorcio Demo',
  cuit: '30712345678',
  domicilio: 'Av. Siempre Viva 742',
  tipo: 'edificio',
  estado: 'activo',
  created_at: ISO,
}

const listFixture = { data: [consorcioFixture], meta: { total: 1 } }

const liquidacionFixture = {
  id: LIQUIDACION_ID,
  consorcio_id: CONSORCIO_ID,
  periodo: '202601',
  vencimiento_1: '2026-02-10',
  vencimiento_2: null,
  estado: 'borrador',
  version: 1,
  total_gastos_cents: 0,
  total_distribuido_cents: 0,
  unidades_alcanzadas: 0,
  created_at: ISO,
  updated_at: ISO,
}

const emptyList = { data: [], meta: {} }

function bodyFor(path: string): unknown {
  if (path === '/me') return meFixture
  if (path === '/consorcios') return listFixture
  if (path === `/consorcios/${CONSORCIO_ID}`) return consorcioFixture
  if (path === '/portal') {
    return {
      data: {
        unidades: [],
        consorcios: [],
        recibos_recientes: [],
        comunicados_recientes: [],
        reclamos_recientes: [],
        total_saldo_vencido_cents: 0,
      },
    }
  }
  if (path === '/portal/notificaciones') return { data: [], meta: { no_leidas: 0 } }
  if (/^\/consorcios\/[^/]+\/morosidad$/.test(path)) {
    return { data: [], meta: { total_saldo_cents: 0, total_cargos_vencidos: 0 } }
  }
  if (/^\/consorcios\/[^/]+\/liquidaciones\/[^/]+$/.test(path)) return liquidacionFixture
  return emptyList
}

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({
    status,
    contentType: status === 401 ? 'application/problem+json' : 'application/json',
    body: JSON.stringify(body),
  })
}

export async function stubAnonymous(page: Page) {
  await page.route('**/api/v1/**', (route) => {
    const path = new URL(route.request().url()).pathname.replace(/^\/api\/v1/, '')
    if (path === '/me') {
      return json(route, 401, { type: 'about:blank', title: 'Unauthorized', status: 401 })
    }
    return json(route, 200, emptyList)
  })
}

export async function stubAuthenticated(page: Page) {
  await page.route('**/api/v1/**', (route) => {
    const path = new URL(route.request().url()).pathname.replace(/^\/api\/v1/, '')
    return json(route, 200, bodyFor(path))
  })
}
