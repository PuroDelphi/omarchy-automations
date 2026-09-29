# Omarchy Automations — documentación

[English](../en/README.md) · [Proyecto](../../README.md)

La interfaz usa inglés por defecto; puedes seleccionar español y la elección se
guarda por perfil. Estas guías equivalentes describen la versión de desarrollo.
Las verificaciones de entrega pendientes están en el [roadmap](../../ROADMAP.md).

| Guía | Contenido |
|---|---|
| [01](user-guide.md) | Empieza aquí: instalación, uso diario y solución de problemas |
| [02](interface.md) | Interfaz, idioma y capturas |
| [03](options.md) | Cada opción, valores iniciales, límites y ejemplos |
| [04](use-cases.md) | Webhooks, alertas de disco y recordatorios programados |
| [05](code-examples.md) | Scripts tipados y adaptadores de eventos |
| [06](provider-example.md) | Eventos firmados de GitHub |
| [07](service-example.md) | Control autorizado de servicios de usuario |
| [08](recovery-example.md) | Reintentos HTTP y resultados inciertos |
| [09](oauth-example.md) | Configuración y limitaciones de Google OAuth |
| [10](system-integration.md) | Hooks de Omarchy y broker administrativo opcional |
| [11](security.md) | Límites de seguridad y garantías comprobadas |
| [12](performance.md) | Rendimiento medido y escenarios de aceptación |
| [13](github.md) | Iniciar sesión y publicar desde esta máquina |

Empieza por la guía de usuario e importa uno de los ejemplos guiados al borrador.
La simulación no ejecuta acciones. Revisa credenciales, fuentes habilitadas y
permisos antes de activar un flujo real. Los ejemplos contienen marcadores de
posición y sus fuentes y flujos vienen deshabilitados.

La [matriz de cobertura](../option-coverage.md) relaciona campos con ambas
referencias. Cada opción tiene al menos dos valores de ejemplo; las guías
anteriores explican flujos completos y resultados esperados. Los valores de
ejemplo son alternativas, no fragmentos que deban combinarse automáticamente.

[Recetas de monitoreo: dos ejemplos por métrica](monitor-recipes.md).

[Recetas de acciones: todos los tipos y perfiles de comando](action-recipes.md).

[Formatos de datos y condiciones](payload-recipes.md).

[Autenticación de webhooks: HMAC, Slack, GitHub y bearer](authentication-recipes.md).

[Notas de versión de desarrollo](release-notes.md).

- [Pruebas de aceptación con el usuario](external-acceptance.md)

[Instalar sin Go](installation.md).
