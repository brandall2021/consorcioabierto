import test from 'node:test'
import assert from 'node:assert/strict'
import { buildGoLiveRehearsalReport } from './go-live-rehearsal.config.mjs'

test('go-live rehearsal report includes timings and outcomes', () => {
  const report = buildGoLiveRehearsalReport({
    environment: 'staging',
    branch: 'main',
    backupDurationSeconds: 180,
    migrateDurationSeconds: 42,
    smokeDurationSeconds: 95,
    restoreDurationSeconds: 210,
    outcome: 'ok',
  })

  assert.match(report, /# Go-live rehearsal/)
  assert.match(report, /Environment: staging/)
  assert.match(report, /Branch: main/)
  assert.match(report, /Backup: 180s/)
  assert.match(report, /Migrate: 42s/)
  assert.match(report, /Smoke: 95s/)
  assert.match(report, /Restore: 210s/)
  assert.match(report, /Outcome: ok/)
})
