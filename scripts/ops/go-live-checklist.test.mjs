import test from 'node:test'
import assert from 'node:assert/strict'
import { buildGoLiveChecklist } from './go-live-checklist.config.mjs'

test('go-live checklist includes preflight, backup, migrate and smoke steps', () => {
  const checklist = buildGoLiveChecklist({
    BASE_URL: 'https://staging.example',
    BACKUP_FILE: 'backups/backup.sql',
  })

  assert.deepEqual(
    checklist.map((item) => item.id),
    ['branch', 'preflight', 'backup', 'migrate', 'smoke', 'rollback'],
  )
  assert.match(checklist[2].command, /make backup-db/)
  assert.match(checklist[3].command, /go run \.\/apps\/api migrate up/)
  assert.match(checklist[4].command, /BASE_URL=https:\/\/staging\.example/)
  assert.match(checklist[5].command, /make restore-db BACKUP_FILE=backups\/backup\.sql/)
})
