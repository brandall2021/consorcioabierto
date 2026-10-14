import { buildGoLiveRehearsalReport } from './go-live-rehearsal.config.mjs'

const report = buildGoLiveRehearsalReport({
  environment: process.env.ENVIRONMENT,
  branch: process.env.BRANCH,
  backupDurationSeconds: process.env.BACKUP_DURATION_SECONDS ? Number(process.env.BACKUP_DURATION_SECONDS) : null,
  migrateDurationSeconds: process.env.MIGRATE_DURATION_SECONDS ? Number(process.env.MIGRATE_DURATION_SECONDS) : null,
  smokeDurationSeconds: process.env.SMOKE_DURATION_SECONDS ? Number(process.env.SMOKE_DURATION_SECONDS) : null,
  restoreDurationSeconds: process.env.RESTORE_DURATION_SECONDS ? Number(process.env.RESTORE_DURATION_SECONDS) : null,
  outcome: process.env.OUTCOME,
})

console.log(report)
