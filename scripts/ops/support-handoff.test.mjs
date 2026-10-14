import test from 'node:test'
import assert from 'node:assert/strict'
import { buildSupportHandoff } from './support-handoff.config.mjs'

test('support handoff includes go-live verification and first-response items', () => {
  const checklist = buildSupportHandoff({
    BASE_URL: 'https://staging.example',
    INCIDENT_CHANNEL: '#ops',
  })

  assert.deepEqual(
    checklist.map((item) => item.id),
    ['handoff', 'verify', 'monitor', 'escalate', 'rollback'],
  )
  assert.match(checklist[0].command, /make go-live-checklist/)
  assert.match(checklist[1].command, /make security-preflight/)
  assert.match(checklist[2].command, /make loadtest/)
  assert.match(checklist[3].command, /#ops/)
  assert.match(checklist[4].command, /make restore-db BACKUP_FILE=/)
})
