export function buildGoLiveRehearsalReport({
  environment = 'staging',
  branch = 'main',
  backupDurationSeconds = null,
  migrateDurationSeconds = null,
  smokeDurationSeconds = null,
  restoreDurationSeconds = null,
  outcome = 'pending',
} = {}) {
  const formatSeconds = (value) => (value == null ? 'n/a' : `${value}s`)

  return [
    '# Go-live rehearsal',
    `Environment: ${environment}`,
    `Branch: ${branch}`,
    `Backup: ${formatSeconds(backupDurationSeconds)}`,
    `Migrate: ${formatSeconds(migrateDurationSeconds)}`,
    `Smoke: ${formatSeconds(smokeDurationSeconds)}`,
    `Restore: ${formatSeconds(restoreDurationSeconds)}`,
    `Outcome: ${outcome}`,
  ].join('\n')
}
