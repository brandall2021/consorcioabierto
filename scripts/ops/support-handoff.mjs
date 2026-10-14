import { buildSupportHandoff } from './support-handoff.config.mjs'

const checklist = buildSupportHandoff(process.env)

console.log('# Support handoff')
for (const item of checklist) {
  console.log(`- [ ] ${item.title}`)
  console.log(`  - Command: \`${item.command}\``)
}
