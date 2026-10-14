import http from 'k6/http'
import { check, sleep } from 'k6'
import { buildK6Options, buildSmokeOptions } from './load-smoke.config.mjs'

const smoke = buildSmokeOptions(__ENV)

export const options = buildK6Options(__ENV)

const jsonHeaders = { headers: { 'Content-Type': 'application/json' } }

function authHeaders(accessToken) {
  return {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  }
}

function firstConsorcioId(response) {
  const payload = response.json()
  return payload?.data?.[0]?.id ?? null
}

export function anonymousHealth() {
  const res = http.get(`${smoke.baseUrl}/api/v1/health`)

  check(res, {
    'health responds 200': (r) => r.status === 200,
  })

  sleep(1)
}

export function authenticatedFlow() {
  const login = http.post(
    `${smoke.baseUrl}/api/v1/auth/login`,
    JSON.stringify({ email: smoke.email, password: smoke.password }),
    jsonHeaders,
  )

  check(login, {
    'login responds 200': (r) => r.status === 200,
  })

  const loginBody = login.json()
  const accessToken = loginBody?.access_token

  check(loginBody, {
    'login returns access token': (body) => typeof body?.access_token === 'string',
  })

  const auth = authHeaders(accessToken)

  const me = http.get(`${smoke.baseUrl}/api/v1/me`, auth)
  check(me, {
    'me responds 200': (r) => r.status === 200,
  })

  const consorcios = http.get(`${smoke.baseUrl}/api/v1/consorcios`, auth)
  check(consorcios, {
    'consorcios responds 200': (r) => r.status === 200,
  })

  const consorcioId = firstConsorcioId(consorcios)
  if (consorcioId) {
    const base = `${smoke.baseUrl}/api/v1/consorcios/${consorcioId}`
    http.get(base, auth)
    http.get(`${base}/liquidaciones`, auth)
    http.get(`${base}/morosidad`, auth)
  }

  const portal = http.get(`${smoke.baseUrl}/api/v1/portal`, auth)
  check(portal, {
    'portal responds 200': (r) => r.status === 200,
  })

  sleep(1)
}
