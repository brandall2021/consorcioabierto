import { buildSecurityPreflightReport } from './security-preflight.config.mjs'

const report = buildSecurityPreflightReport(process.env)

for (const item of report.checks) {
  const prefix = item.ok ? '[x]' : '[ ]'
  console.log(`${prefix} ${item.name}: ${item.message}`)
}

process.exit(report.exitCode)
