const DEFAULT_COMPOSE_FILE = 'deploy/compose.yaml'
const DEFAULT_BACKUP_DIR = 'backups'
const DEFAULT_POSTGRES_USER = 'consorcio'
const DEFAULT_POSTGRES_DB = 'consorcioabierto'

function quote(value) {
  return `"${value}"`
}

function normalizeTimestamp(timestamp = new Date().toISOString()) {
  return timestamp.replace(/[:.]/g, '-')
}

export function buildBackupCommand({
  composeFile = DEFAULT_COMPOSE_FILE,
  backupDir = DEFAULT_BACKUP_DIR,
  timestamp,
  postgresUser = DEFAULT_POSTGRES_USER,
  postgresDb = DEFAULT_POSTGRES_DB,
} = {}) {
  const file = `${backupDir}/backup-${normalizeTimestamp(timestamp)}.sql`

  return [
    `mkdir -p ${quote(backupDir)}`,
    `docker compose -f ${quote(composeFile)} exec -T postgres pg_dump -U "${'${POSTGRES_USER:-' + postgresUser + '}'}" -d "${'${POSTGRES_DB:-' + postgresDb + '}'}" --no-owner --no-acl > ${quote(file)}`,
  ].join(' && ')
}

export function buildRestoreCommand({
  composeFile = DEFAULT_COMPOSE_FILE,
  backupFile,
  postgresUser = DEFAULT_POSTGRES_USER,
  postgresDb = DEFAULT_POSTGRES_DB,
} = {}) {
  if (!backupFile) {
    throw new Error('backupFile is required')
  }

  return `cat ${quote(backupFile)} | docker compose -f ${quote(composeFile)} exec -T postgres psql -U "${'${POSTGRES_USER:-' + postgresUser + '}'}" -d "${'${POSTGRES_DB:-' + postgresDb + '}'}"`
}
