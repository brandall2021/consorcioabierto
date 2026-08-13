// Feature flags de la Fase 1 (§8.1). Los módulos futuros se ocultan, nunca se
// muestran como páginas vacías. Cada flag define si su navegación está activa.
export const featureFlags = {
  consorcios: true,
  expensas: false,
  administracion: false,
  auditoria: true,
  perfil: true,
} as const

export type FeatureFlag = keyof typeof featureFlags

export function isEnabled(flag: FeatureFlag): boolean {
  return featureFlags[flag]
}
