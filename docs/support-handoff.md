# Support handoff

Checklist mínima para las primeras horas/días post go-live.

## Ejecutar

```sh
make support-handoff
```

## Qué valida

- La checklist de go-live ya se corrió.
- El preflight de seguridad sigue pasando.
- El smoke de carga básico sigue respondiendo.
- El canal de soporte/incidentes está abierto.
- El rollback sigue disponible.

## Regla operativa

Si falla cualquiera de los checks críticos, se corta el despliegue y se vuelve al último backup válido.
