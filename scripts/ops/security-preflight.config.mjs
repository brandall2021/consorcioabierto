function check(name, ok, message) {
  return { name, ok, message }
}

function isProduction(env) {
  return (env.APP_ENV ?? 'local') === 'production'
}

function requiresOtlp(env) {
  return (env.OTEL_EXPORTER ?? 'console') === 'otlp'
}

export function buildSecurityPreflightReport(env = {}) {
  if (!isProduction(env)) {
    return {
      exitCode: 0,
      checks: [check('app_env', true, 'no-op fuera de production')],
    }
  }

  const checks = [
    check(
      'STORAGE_DRIVER',
      (env.STORAGE_DRIVER ?? '') === 's3',
      'STORAGE_DRIVER debe ser s3 en production',
    ),
    check(
      'MAIL_DRIVER',
      (env.MAIL_DRIVER ?? '') === 'smtp',
      'MAIL_DRIVER debe ser smtp en production',
    ),
    check(
      'PSP_DRIVER',
      (env.PSP_DRIVER ?? '') !== 'mock',
      'PSP_DRIVER no puede ser mock en production',
    ),
    check(
      'SCAN_DRIVER',
      (env.SCAN_DRIVER ?? '') !== 'mock',
      'SCAN_DRIVER no puede ser mock en production',
    ),
    check(
      'OTEL_EXPORTER',
      ['otlp', 'none'].includes(env.OTEL_EXPORTER ?? 'console'),
      'OTEL_EXPORTER debe ser otlp o none en production',
    ),
    check(
      'OTEL_EXPORTER_OTLP_ENDPOINT',
      !requiresOtlp(env) || Boolean((env.OTEL_EXPORTER_OTLP_ENDPOINT ?? '').trim()),
      'OTEL_EXPORTER_OTLP_ENDPOINT es obligatorio cuando OTEL_EXPORTER=otlp',
    ),
  ]

  return {
    exitCode: checks.every((item) => item.ok) ? 0 : 1,
    checks,
  }
}
