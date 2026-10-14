import { buildGoLiveChecklist } from './go-live-checklist.config.mjs'

const checklist = buildGoLiveChecklist(process.env)

console.log('# Go-live checklist')
for (const item of checklist) {
  console.log(`- [ ] ${item.title}`)
  console.log(`  - Command: \`${item.command}\``)
}
