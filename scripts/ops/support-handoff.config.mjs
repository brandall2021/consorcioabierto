const DEFAULT_BACKUP_FILE = 'backups/backup-latest.sql'
const DEFAULT_INCIDENT_CHANNEL = '#support'

export function buildSupportHandoff(env = {}) {
  const baseUrl = env.BASE_URL ?? 'http://127.0.0.1:8080'
  const incidentChannel = env.INCIDENT_CHANNEL ?? DEFAULT_INCIDENT_CHANNEL
  const backupFile = env.BACKUP_FILE ?? DEFAULT_BACKUP_FILE

  return [
    {
      id: 'handoff',
      title: 'Cerrar el go-live',
      command: 'make go-live-checklist',
    },
    {
      id: 'verify',
      title: 'Confirmar salud del entorno',
      command: 'make security-preflight',
    },
    {
      id: 'monitor',
      title: 'Monitorear smoke post-deploy',
      command: `BASE_URL=${baseUrl} make loadtest`,
    },
    {
      id: 'escalate',
      title: 'Abrir canal de soporte',
      command: `incidente en ${incidentChannel} con captura de logs y métricas`,
    },
    {
      id: 'rollback',
      title: 'Mantener rollback disponible',
      command: `make restore-db BACKUP_FILE=${backupFile}`,
    },
  ]
}
