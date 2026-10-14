import test from 'node:test'
import assert from 'node:assert/strict'
import { buildSmokeOptions } from './load-smoke.config.mjs'

test('buildSmokeOptions uses the default local base url and load thresholds', () => {
  const options = buildSmokeOptions({})

  assert.equal(options.baseUrl, 'http://127.0.0.1:8080')
  assert.deepEqual(Object.keys(options.scenarios).sort(), ['anonymous_health', 'authenticated_flow'])
  assert.equal(options.thresholds['http_req_failed'], 'rate<0.01')
  assert.equal(options.thresholds['http_req_duration'], 'p(95)<800')
})
