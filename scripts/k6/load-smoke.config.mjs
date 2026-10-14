const DEFAULT_BASE_URL = 'http://127.0.0.1:8080'
const DEFAULT_EMAIL = 'admin@tenant-a.com'
const DEFAULT_PASSWORD = 'Admin.123'

export function buildSmokeOptions(env = {}) {
  return {
    baseUrl: env.BASE_URL ?? DEFAULT_BASE_URL,
    email: env.LOADTEST_EMAIL ?? DEFAULT_EMAIL,
    password: env.LOADTEST_PASSWORD ?? DEFAULT_PASSWORD,
    thresholds: {
      http_req_failed: 'rate<0.01',
      http_req_duration: 'p(95)<800',
    },
    scenarios: {
      anonymous_health: {
        executor: 'constant-vus',
        vus: 1,
        duration: '20s',
        exec: 'anonymousHealth',
      },
      authenticated_flow: {
        executor: 'constant-vus',
        vus: 1,
        duration: '20s',
        startTime: '2s',
        exec: 'authenticatedFlow',
      },
    },
  }
}

export function buildK6Options(env = {}) {
  const smoke = buildSmokeOptions(env)

  return {
    scenarios: smoke.scenarios,
    thresholds: {
      http_req_failed: [smoke.thresholds.http_req_failed],
      http_req_duration: [smoke.thresholds.http_req_duration],
    },
  }
}
