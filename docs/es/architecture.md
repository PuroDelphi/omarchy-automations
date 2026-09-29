# Arquitectura y fronteras de confianza

[English](../en/architecture.md) · [Documentación](README.md) · [Protocolo](protocol.md)

Describe la versión de desarrollo implementada. La [propuesta original](../DESIGN.md)
es un documento histórico; sus afirmaciones en futuro no representan el estado
actual. Consulta el roadmap para evidencias y pruebas aplazadas.

## Componentes

```text
Webhooks autenticados ─┐
Monitores / horarios ──┼─> eventos validados ─> SQLite ─> flujos revisados
Hooks Omarchy / local ─┘                                  │
                                             acciones tipadas / HTTPS
                                                         │
                                                historial de ejecución

Panel QML <─> quatrroctl <─> socket Unix privado <─> quatrrod
                                                      │
                                  broker privilegiado opcional separado
```

El panel QML nativo edita borradores, simula entradas y muestra capacidades,
credenciales, historial y estado. No aloja el receptor HTTP. `quatrrod` es un
servicio de usuario separado: ocultar o recargar el panel no detiene los flujos.
`quatrroctl` negocia el protocolo local y envía una operación por conexión.
El motor usa Go y SQLite, con dependencias fijadas en `go.mod`/`go.sum`.

Los plugins Omarchy ejecutan código del usuario dentro del shell. El panel no es
un sandbox; componentes con el mismo UID no crean una frontera fuerte contra
software local malicioso. El aislamiento restringe trabajos despachados, no
convierte todo el escritorio en confiable.

## Configuración y ejecución

Las entradas describen autenticación y formato. Los destinos definen HTTPS y
referencias de credenciales. Los flujos seleccionan origen, exigen todas sus
condiciones y ejecutan pasos ordenados. Monitores y programaciones generan
eventos. Scripts/adaptadores son código local revisado con revisiones fijadas.

Guardar valida un borrador. Simular evalúa condiciones y plantillas sin efectos.
Activar exige hash y capacidades exactas revisadas. Cada ejecución conserva su
revisión; los permisos se comprueban otra vez antes de despachar. Revocar bloquea
efectos futuros, incluido trabajo pendiente. Los datos remotos no seleccionan
libremente ejecutables, servicios, credenciales ni URL.

Las plantillas conservan tipos escalares completos y codifican la salida
estructurada, evitando que datos creen campos JSON/form extra rompiendo comillas.
Las condiciones son comparaciones declarativas acotadas, no JavaScript o shell.

## Entrada y salida

HTTP escucha en loopback. Recibir remotamente requiere proxy/túnel TLS explícito
y confiable; ni setup ni el túnel sustituyen la autenticación. El socket Unix
administrativo no debe exponerse junto al receptor. La autenticación comprueba
bytes originales antes de normalizar/persistir. Aceptar HTTP confirma persistencia,
no acciones completadas.

Los destinos salientes son HTTPS registrados, sin redirecciones. Se resuelven,
validan y fijan direcciones al conectar manteniendo verificación TLS del hostname.
Las excepciones privadas exactas son capacidades explícitas. Los cuerpos quedan
persistidos para reintentos; el receptor debe implementar idempotencia. No se
prometen efectos exactamente una vez entre servicios independientes.

## Estado y recuperación

SQLite conserva inbox, ejecuciones, pasos, outbox, estado de monitores/horarios,
borradores, revisiones y permisos. Las migraciones son transaccionales y rechazan
esquemas futuros. Cuotas/retención acotan el trabajo admitido. No se elimina
silenciosamente trabajo pendiente o incierto para admitir eventos nuevos.

La recuperación distingue efectos completados, fallidos, cancelados e inciertos.
Un comando incierto podría haberse ejecutado; no se repite a ciegas. Reintentar
HTTP conserva cuerpo/clave; no renderiza el borrador nuevo. No existe rollback
universal para notificaciones, comandos ni solicitudes externas. Pausa/admisión,
cancelación y revocación son operaciones distintas.

## Secretos y acciones del sistema

Las credenciales usan Secret Service o archivo privado elegido explícitamente.
Los archivos tienen permisos restrictivos, no cifrado; no hay fallback silencioso.
La configuración almacena referencias. Secretos pegados manualmente en cabeceras
o cuerpos normales siguen siendo configuración exportable y deben evitarse.

Comandos/scripts usan systemd de usuario y Bubblewrap con límites de recursos,
tiempo, archivos y red. Si el aislamiento no funciona se rechaza la activación.
Los directorios aprobados tienen identidad y modo verificados. Los scripts
se ejecutan desde contenido aprobado, no desde una ruta fuente mutable.

Las acciones de servicio de usuario seleccionan unidades/operaciones exactas.
Servicios del sistema requieren broker separado, política administrativa y
Polkit. El motor no se convierte en shell root genérico. Setup no instala el broker.

## Propiedad de la instalación

En el modo recomendado, Omarchy posee el checkout Git con QML y scripts setup.
El instalador posee solo dos binarios y una unidad de usuario, con hashes en un
recibo privado. Compara UI del checkout y paquete seleccionado antes de instalar.
Retira primero el runtime y después el checkout mediante Omarchy.

La alternativa desde fuentes/archivo también posee la UI instalada, usando
directorios de revisión inmutables para evitar cachés QML antiguos. Ambos modos
conservan datos/credenciales al retirar, rechazan archivos propios modificados y
no se mezclan sobre un recibo existente. Hooks y políticas del broker tienen
ciclos independientes.

Rutas predeterminadas (los overrides XDG se aplican a las rutas del backend):

| Propósito | Ubicación |
|---|---|
| Panel Omarchy | `~/.config/omarchy/plugins/quatrro.automations/` |
| Configuración del motor | `$XDG_CONFIG_HOME/quatrro/` |
| SQLite e historial | `$XDG_STATE_HOME/quatrro/` |
| Socket local | `$XDG_RUNTIME_DIR/quatrro/control.sock` |
| Servicio de usuario | `$XDG_CONFIG_HOME/systemd/user/quatrrod.service` |
| Runtime | `~/.local/bin/quatrrod`, `~/.local/bin/quatrroctl` |
| Recibo/backups | `$XDG_STATE_HOME/quatrro-install/` |

`QUATRRO_PROFILE` aísla datos de desarrollo. No convierte acciones reales en
simulaciones. Consulta [seguridad](security.md), [instalación](installation.md) y
[aceptación externa](external-acceptance.md) para sus límites operativos.
