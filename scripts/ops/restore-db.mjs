import { spawnSync } from 'node:child_process'
import { buildRestoreCommand } from './db-backup-restore.config.mjs'

const dryRun = process.argv.includes('--dry-run') || process.env.DRY_RUN === '1'
const backupFile = process.argv.slice(2).find((arg) => !arg.startsWith('--')) || process.env.BACKUP_FILE

if (!backupFile) {
  console.error('Uso: node scripts/ops/restore-db.mjs <backup-file>')
  process.exit(1)
}

const command = buildRestoreCommand({
  composeFile: process.env.COMPOSE_FILE,
  backupFile,
  postgresUser: process.env.POSTGRES_USER,
  postgresDb: process.env.POSTGRES_DB,
})

if (dryRun) {
  console.log(command)
  process.exit(0)
}

const result = spawnSync('sh', ['-lc', command], { stdio: 'inherit' })
process.exit(result.status ?? 1)
