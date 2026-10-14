import test from 'node:test'
import assert from 'node:assert/strict'
import { buildSecurityPreflightReport } from './security-preflight.config.mjs'

test('production security preflight fails for simulated drivers and console tracing', () => {
  const report = buildSecurityPreflightReport({
    APP_ENV: 'production',
    STORAGE_DRIVER: 'minio',
    MAIL_DRIVER: 'mailpit',
    PSP_DRIVER: 'mock',
    SCAN_DRIVER: 'mock',
    OTEL_EXPORTER: 'console',
  })

  assert.equal(report.exitCode, 1)
  assert.equal(report.checks.every((check) => check.ok), false)
  assert.match(report.checks.map((check) => check.message).join('\n'), /STORAGE_DRIVER|MAIL_DRIVER|PSP_DRIVER|SCAN_DRIVER|OTEL_EXPORTER/)
})

test('production security preflight passes for real drivers and otlp tracing', () => {
  const report = buildSecurityPreflightReport({
    APP_ENV: 'production',
    STORAGE_DRIVER: 's3',
    MAIL_DRIVER: 'smtp',
    PSP_DRIVER: 'mercadopago',
    SCAN_DRIVER: 'none',
    OTEL_EXPORTER: 'otlp',
    OTEL_EXPORTER_OTLP_ENDPOINT: 'http://otel:4318',
  })

  assert.equal(report.exitCode, 0)
  assert.equal(report.checks.every((check) => check.ok), true)
})
