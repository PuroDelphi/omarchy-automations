# Proveedores de entrada

[English](../en/providers.md) · [Documentación](README.md)

Selecciona la autenticación del proveedor en la entrada y registra su secreto
como referencia desde Seguridad. Una entrada usa un solo formato. Para recibir
JSON y formularios del mismo proveedor, crea entradas independientes.
La exposición pública requiere el proxy TLS del despliegue; el motor conserva
su listener local.

## GitHub

El adaptador `github` valida HMAC-SHA256 de los bytes originales mediante
`X-Hub-Signature-256`. Se incluye el vector de prueba publicado por
[GitHub](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries).

Política de Omarchy Automations: se exige `X-GitHub-Delivery`, pero la identidad se obtiene
del SHA256 del cuerpo autenticado porque esa cabecera no está firmada. Cuerpos
idénticos se consolidan durante la retención de deduplicación. No se usa
`X-GitHub-Event` para decidir permisos o el tipo de evento; filtra por campos
del cuerpo, por ejemplo `data.action` y `data.repository.full_name`.
No hay timestamp firmado en este contrato: la deduplicación retenida no
demuestra frescura indefinida.

## Slack

El adaptador `slack` admite JSON y formulario URL-encoded. Usa el **Signing
Secret**, no un token de bot ni el token de verificación antiguo. Comprueba
`X-Slack-Signature` como HMAC-SHA256 de `v0:<timestamp>:<cuerpo-original>` y
limita la diferencia del timestamp a cinco minutos, siguiendo el
[contrato de firmas de Slack](https://docs.slack.dev/authentication/verifying-requests-from-slack/).

El evento `url_verification` autenticado devuelve HTTP 200 y un objeto JSON
con el `challenge`, conforme al
[contrato de verificación de URL](https://docs.slack.dev/reference/events/url_verification/).
Omarchy Automations limita el desafío a 2048 bytes y no crea ejecuciones para ese intercambio.

Los eventos normales reciben HTTP 200 con `{}` después del commit. El tipo es
el `type` del cuerpo; los eventos internos se consultan mediante
`data.event.type`. Para una entrada JSON puedes filtrar, por ejemplo:

```json
{"field":"data.event.type","op":"eq","value":"app_mention"}
```

La identidad se liga al cuerpo firmado. Un reintento con timestamp nuevo o
cabeceras de reintento distintas no vuelve a ejecutar el mismo cuerpo. Esta
política también consolida cuerpos idénticos legítimos durante la ventana
de deduplicación. Se rechazan firmas inválidas, timestamps ausentes/vencidos
y cabeceras de autenticación repetidas.

El adaptador no responde a `response_url` ni envía mensajes por su cuenta.
Las interacciones encapsuladas en un campo `payload` se conservan como texto
de formulario; no se anuncia interpretación de todos los tipos de interacción.
La validación realizada utiliza fixtures y HTTP local; no se ha conectado
una aplicación de una cuenta real de Slack o GitHub.

## Contrato para añadir un proveedor incorporado

El contrato `inboundAdapter`, versión 1, está en
`internal/core/providers.go`. El registro es estático mediante
`inboundProvider`; no se descarga ni ejecuta código indicado por una petición.

1. Implementar `Verify(request, raw, secret, now)`: validar los bytes originales,
   rechazar ambigüedad, comprobar frescura cuando exista y devolver una identidad
   estable autenticada. No realizar E/S ni guardar credenciales. `now` permite
   pruebas deterministas.
2. Declarar `Formats`, `SuccessStatus` y `EmptyAcknowledgement`. El handler
   aplica autenticación antes de parsear y persistencia antes del éxito normal.
3. Si hace falta, implementar `Challenge(data)`: devuelve respuesta, indicador
   de manejo y error. Solo recibe datos previamente autenticados y parseados.
   El desafío debe ser acotado y no producir efectos de flujos.
4. Añadir el nombre al esquema JSON y al selector QML; comprobar la combinación
   de proveedor/formato en la validación del modelo.
5. Añadir un vector oficial conocido, pruebas de cuerpo alterado, replay,
   cabeceras duplicadas, errores y persistencia fallida. Documentar campos
   firmados, deduplicación y limitaciones de la integración.

Las extensiones externas en procesos separados corresponden a F3.2; este
contrato no las habilita ni sustituye sus permisos ni su aislamiento. Los adaptadores externos implementados se
describen en [ejemplos de código](../es/code-examples.md).
