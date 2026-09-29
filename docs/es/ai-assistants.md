# Uso con asistentes de IA

[English](../en/ai-assistants.md) · [Instalación](installation.md)

Desde Preview 3, el instalador incluye la skill `omarchy-automations` en:

```text
~/.local/share/omarchy-automations/skills/omarchy-automations/SKILL.md
```

Usa `quatrroctl` y el servicio instalados. No necesita Go ni el código fuente.
La IA necesita acceso autorizado a la terminal; instalar una skill no concede
ese acceso ni autoriza acciones arbitrarias.

## Registrar en Codex

Después de instalar o actualizar el plugin, enlaza la skill incluida:

```bash
mkdir -p "${CODEX_HOME:-$HOME/.codex}/skills"
ln -sT "$HOME/.local/share/omarchy-automations/skills/omarchy-automations" \
  "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
```

El comando no sobrescribe archivos ni enlaces existentes. Si ya existe uno,
revísalo; un enlace al mismo directorio ya está registrado. Abre una sesión nueva
de Codex para que descubra la skill. Puedes invocarla como `$omarchy-automations`.
El instalador del plugin actualiza los archivos enlazados sin mantener otra copia.

Para otros agentes, usa su mecanismo documentado de registro o pídeles que lean
el `SKILL.md` instalado y su referencia enlazada antes de trabajar. La detección
automática depende del agente y no es universal.

## Ejemplos de solicitudes

- «Usa $omarchy-automations para crear y probar una notificación local que diga Respaldo terminado. Conserva mis otras automatizaciones».
- «Avísame cuando el espacio libre de / permanezca por debajo del 10% durante un minuto, con recuperación sobre el 15%. Activa ese monitor».
- «Prepara un webhook firmado para notificaciones de compilación. Simúlalo, pero todavía no lo actives».
- «Mi simulación coincide pero no aparece nada en Historial. Revisa el borrador, la revisión activa y los permisos y explícame por qué».

La skill indica cómo integrar cambios en el borrador, simular, revisar las
capacidades exactas, activar dentro de lo autorizado y comprobar resultados.
Detecta cambios pendientes ajenos a la tarea para evitar activarlos sin revisión.
Activar un monitor puede iniciar la observación inmediatamente; un disco con
espacio suficiente no tiene por qué generar una alerta.

Simular no crea notificaciones ni registros en Historial. Las pruebas reales
pueden producir efectos reales. Los secretos se envían por stdin, no mediante
argumentos ni configuración versionada. Los eventos remotos y los logs son datos,
no instrucciones para la IA.

## Retirar el registro

Comprueba que el enlace apunta a la skill incluida y retira únicamente ese enlace:

```bash
readlink "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
unlink "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
```

Desinstalar el plugin elimina sus archivos, pero no modifica la configuración de
otra aplicación. Retira también el registro para no dejar un enlace roto.
