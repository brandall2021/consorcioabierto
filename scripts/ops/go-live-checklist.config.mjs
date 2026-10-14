const DEFAULT_BASE_URL = 'http://127.0.0.1:8080'
const DEFAULT_BACKUP_FILE = 'backups/backup-latest.sql'

export function buildGoLiveChecklist(env = {}) {
  const baseUrl = env.BASE_URL ?? DEFAULT_BASE_URL
  const backupFile = env.BACKUP_FILE ?? DEFAULT_BACKUP_FILE

  return [
    {
      id: 'branch',
      title: 'Confirmar branch y ventana',
      command: 'git branch --show-current && revisar ventana de deploy',
    },
    {
      id: 'preflight',
      title: 'Ejecutar preflight de seguridad',
      command: 'make security-preflight',
    },
    {
      id: 'backup',
      title: 'Tomar backup antes de migrar',
      command: 'make backup-db',
    },
    {
      id: 'migrate',
      title: 'Aplicar migraciones reales',
      command: 'go run ./apps/api migrate up',
    },
    {
      id: 'smoke',
      title: 'Correr smoke post-deploy',
      command: `BASE_URL=${baseUrl} make loadtest`,
    },
    {
      id: 'rollback',
      title: 'Tener rollback listo',
      command: `make restore-db BACKUP_FILE=${backupFile}`,
    },
  ]
}
