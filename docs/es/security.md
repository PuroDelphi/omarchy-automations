# Límites de seguridad y verificación

[English](../en/security.md) · [Guía de usuario](user-guide.md) · [Permisos y opciones](options.md)

Los datos remotos no conceden permisos. Activar vincula permisos con recursos y
revisiones revisados; ejecutar vuelve a comprobarlos. Conserva privadas las
credenciales de entrada y revisa qué puede hacer cada flujo habilitado. Un emisor
autenticado puede desencadenar los efectos autorizados para sus flujos coincidentes.

El socket de control local se restringe al mismo usuario Linux. No protege frente
a malware que ya se ejecuta como ese usuario, un administrador, kernel comprometido
o código malicioso que apruebes explícitamente. Scripts y comandos usan aislamiento
comprobado en el host; los permisos de directorios exponen todo el árbol elegido,
incluidos los sockets que contenga.

| Límite | Evidencia verificada |
|---|---|
| Firmas y replay | HMAC vincula cuerpo/timestamp/entrega; vectores GitHub y Slack, manipulación, cabeceras duplicadas, timestamps vencidos donde corresponda y deduplicación persistida. La identidad GitHub deriva de bytes firmados, no de su cabecera de entrega sin firmar. |
| Direcciones de salida | Restricciones IPv4/IPv6, rechazo de DNS mixto público/privado, excepciones exactas host:puerto, conexión directa a dirección validada, revalidación de respuestas cambiantes, sin seguir redirecciones ni usar proxies del entorno. |
| Inyección | Plantillas sin evaluación de código; escape JSON/form, argumentos literales tipados, separación de opciones y markup escapado en avisos; rechazo de configuración ambigua/no admitida. |
| Revocación | Cambiar revisión exige revisar. Revocar acciones, monitores, temporizadores y destinos privados bloquea su trabajo futuro, incluidos casos de cola/reintento comprobados. |
| Archivos y socket | Directorios privados, socket y credenciales con modo 0600, comprobación UID del peer; rechazo de symlinks, archivos especiales, modos inseguros y valores excesivos. FIFO se abre sin bloqueo antes de validar tipo. |
| Aislamiento efectivo | Pruebas reales systemd/Bubblewrap de límites cgroup CPU, memoria y procesos; restricciones de archivos/red/entorno; timeout y cancelación de descendientes. |
| Divulgación | Valores de credenciales excluidos de exportación de configuración, diagnóstico y vistas operativas; fallos OAuth controlados ocultan detalles del proveedor/transporte; scripts no heredan secretos fixture del entorno. |
| Broker administrativo opcional | Política exacta unidad/operación/UID, identidad del kernel, Polkit y autorización repetida; sin interfaz de comandos root arbitrarios. Pruebas separadas de systemd/Polkit aislados en el registro de validación. |

Familias representativas: `ingress_test.go`, `providers_test.go`, `egress_test.go`,
`lan_test.go`, `config_test.go`, `events_test.go`, `secrets_test.go`,
`diagnostics_test.go`, `scripts_test.go`, `isolation_host_test.go`,
`internal/local` e `internal/broker`. El [registro de validación](../validation.md)
recoge comandos, resultados y alcance de las integraciones.

## Límites prácticos

- Una firma GitHub no aporta timestamp confiable del evento. La deduplicación
  tiene retención; no garantiza frescura indefinida ni efectos externos exactamente
  una vez. El receptor debe implementar idempotencia cuando se necesite.
- Las excepciones HTTPS privadas autorizan un host y puerto, con validación de
  direcciones actuales. Un hostname permitido no garantiza que su aplicación
  remota u operador DNS sean confiables.
- El almacén explícito en archivo guarda texto plano protegido por permisos.
  El keyring puede fallar si no está disponible o está bloqueado; no debe pasar
  silenciosamente a texto plano. La integración real de cuenta Google OAuth/keyring
  sigue siendo una validación separada. No incrustes secretos en código o payloads.
- La configuración exportada incluye texto ordinario y código aprobado aunque
  excluya valores del almacén de credenciales. Revisa antes de compartir.
- Revocar y cancelar no deshace efectos ya aceptados fuera del motor. Investiga
  resultados inciertos antes de repetir trabajo.

## Reproducir desde el código fuente

```bash
go test -race ./...
QUATRRO_HOST_TEST=1 go test ./internal/core -run '^TestHostIsolation' -count=1 -v
```

El segundo comando crea unidades de usuario temporales y ejercita límites reales;
requiere el entorno systemd/Bubblewrap admitido. Las pruebas ordinarias omiten
integraciones que requieren activación explícita. La prueba de DNS cambiante usa
resolver controlado y receptor TLS local, no un servicio DNS público.
Son pruebas del proyecto y revisión de código, no una auditoría de intrusión
independiente ni garantía de haber eliminado toda vulnerabilidad posible.
