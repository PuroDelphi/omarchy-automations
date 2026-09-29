# Recetas de autenticación de webhooks

[English](../en/authentication-recipes.md) · [Guía de usuario](user-guide.md) · [Formatos](payload-recipes.md)

Crea una credencial genérica en Seguridad y usa su ID como referencia de secreto
en la entrada. Todas las entradas exigen autenticación. Usa un valor privado para
integraciones reales; los [ocho vectores de prueba](../../examples/authentication-vectors.json)
usan un valor público intencional y un reloj fijo. No son credenciales reales ni
configuración importable. Cada proveedor tiene dos cuerpos firmados o autenticados:
una compilación fallida y otra correcta.

## HMAC genérico

Ejemplo A: entrada `deploy`, credencial `deploy-signing`, cuerpo JSON
`{"type":"build","state":"failed"}`. Ejemplo B: entrada `backup`, credencial
`backup-signing`, cuerpo JSON `{"type":"backup","state":"passed"}`.
Para ambas entradas, el emisor calcula HMAC-SHA256 sobre estos bytes exactos:

```text
ASCII(timestamp) + "." + delivery_id + "." + cuerpo_HTTP_original
```

Envía `X-Quatrro-Timestamp` como segundos Unix canónicos, `X-Quatrro-Delivery`
como identificador no vacío y `X-Quatrro-Signature` como `sha256=` más el HMAC
hexadecimal. El timestamp debe estar a un máximo de cinco minutos del reloj receptor.
Al reintentar el mismo evento conserva su ID; usa uno nuevo para un evento
realmente nuevo. El ID está firmado. Cambiar el cuerpo o ID sin recalcular la
firma falla. La ventana de deduplicación retenida es finita; la autenticación
no promete entrega exactamente una vez de forma ilimitada.

## Slack

Usa el Signing Secret de la app, no un token de bot. Ejemplo A: entrada `mentions`,
JSON, condición `data.event.type eq app_mention`. Ejemplo B: entrada `commands`,
formulario, condición `data.command eq /backup` (Texto). Usa entradas separadas
por formato. El verificador de Slack solo admite JSON/formulario y comprueba:

```text
X-Slack-Request-Timestamp: segundos Unix canónicos
X-Slack-Signature: v0= + hex(HMAC-SHA256(secreto, "v0:" + timestamp + ":" + cuerpo_original))
```

Aplica la misma tolerancia de cinco minutos. Una petición JSON firmada
`url_verification` devuelve su challenge (máximo 2048 bytes) sin ejecutar flujos.
Los eventos normales reciben HTTP 200 con `{}` después de persistir. Los campos
del evento JSON interno quedan bajo `data.event`; una interacción en el campo
`payload` de formulario permanece como texto, sin decodificación JSON automática.
No se envía respuesta implícita a `response_url`. Un destino outbound debe estar
registrado y autorizado.

Cuerpos Slack firmados idénticos se deduplican incluso con timestamp nuevo o
cabeceras de reintento distintas. Esto también consolida cuerpos legítimos
idénticos durante la retención. Los vectores prueban la implementación local;
no se ha verificado una integración con un workspace real de Slack.

## GitHub y bearer

Para GitHub, el ejemplo A filtra repositorio y acción de una release; el B filtra
estado y entorno de un despliegue. Consulta el [ejemplo completo de release firmada](provider-example.md).
Envía `X-Hub-Signature-256: sha256=<HMAC hexadecimal del cuerpo original>` y un
`X-GitHub-Delivery` no vacío. Las cabeceras de evento/entrega no están firmadas
con el cuerpo; el plugin usa identidad del cuerpo y condiciones sobre sus campos
autenticados. Este contrato no incluye timestamp firmado, por lo que la
deduplicación retenida no demuestra frescura indefinida.

Para bearer, el ejemplo A es un emisor CI a la entrada `builds`; el B, un trabajo
local a `backups`. Cada uno envía `Authorization: Bearer <su valor privado>`.
`X-Quatrro-Delivery` opcional indica el ID de entrega. Bearer no incluye frescura
firmada del reloj ni vinculación al cuerpo; usa TLS para tráfico remoto y protege
la credencial. No es OAuth solo porque la cabecera diga Bearer.

## Verificar el límite de autenticación

Configura un flujo para la fuente exacta de la entrada y habilítalo/revísalo solo
cuando quieras efectos reales. Como prueba positiva envía el cuerpo firmado sin
cambios; como negativa altera un byte sin actualizar la firma. HMAC, Slack y
GitHub deben rechazarlo. Para bearer prueba un token incorrecto: bearer no firma
el cuerpo. Las cabeceras de autenticación repetidas se rechazan en lugar de elegir
una ambiguamente. La aceptación normal fuera de Slack es HTTP 202 después de
persistir; Historial informa resultados posteriores de acciones.

La prueba de vectores usa un reloj fijo y no envía HTTP. Los emisores reales deben
generar timestamps actuales y firmas privadas localmente. Simular prueba condiciones
y plantillas, no firmas ni transporte de red.
