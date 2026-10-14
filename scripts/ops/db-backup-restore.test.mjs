import test from 'node:test'
import assert from 'node:assert/strict'
import { buildBackupCommand, buildRestoreCommand } from './db-backup-restore.config.mjs'

test('buildBackupCommand uses compose postgres defaults and writes into backups', () => {
  const cmd = buildBackupCommand({ timestamp: '2026-10-14T12-00-00Z' })

  assert.match(cmd, /docker compose -f "deploy\/compose\.yaml" exec -T postgres pg_dump/)
  assert.match(cmd, /backups\/backup-2026-10-14T12-00-00Z\.sql/)
  assert.match(cmd, /POSTGRES_USER:-consorcio/)
  assert.match(cmd, /POSTGRES_DB:-consorcioabierto/)
})

test('buildRestoreCommand uses the provided backup file path', () => {
  const cmd = buildRestoreCommand({ backupFile: 'backups/backup.sql' })

  assert.match(cmd, /cat "backups\/backup\.sql" \| docker compose -f "deploy\/compose\.yaml" exec -T postgres psql/)
})
