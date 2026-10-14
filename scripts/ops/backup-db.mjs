import { spawnSync } from 'node:child_process'
import { buildBackupCommand } from './db-backup-restore.config.mjs'

const dryRun = process.argv.includes('--dry-run') || process.env.DRY_RUN === '1'
const command = buildBackupCommand({
  composeFile: process.env.COMPOSE_FILE,
  backupDir: process.env.BACKUP_DIR,
  timestamp: process.env.BACKUP_TIMESTAMP,
  postgresUser: process.env.POSTGRES_USER,
  postgresDb: process.env.POSTGRES_DB,
})

if (dryRun) {
  console.log(command)
  process.exit(0)
}

const result = spawnSync('sh', ['-lc', command], { stdio: 'inherit' })
process.exit(result.status ?? 1)
