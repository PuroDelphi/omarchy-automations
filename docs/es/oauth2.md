# Google OAuth2

[English](../en/oauth2.md) · [Documentación](README.md)

Implementación en desarrollo de F3.3. Incluye consentimiento Desktop, importación
de refresh token, renovación, destinos HTTP y UI inglés/español. Las pruebas usan
perfiles temporales, loopback real y proveedor simulado. No se ha verificado una
cuenta Google real; esta comprobación continúa pendiente en F3.3.

Referencias consultadas: [aplicaciones Desktop](https://developers.google.com/identity/protocols/oauth2/native-app)
y [renovación](https://developers.google.com/identity/protocols/oauth2/web-server).

## Conectar desde el panel

1. En Seguridad → Crear o rotar credencial, seleccionar Google OAuth2: autorizar
   en navegador. Utilizar client ID/client secret de un cliente Google **Desktop**
   propio, un ID local de conexión y scopes actuales de las APIs necesarias.
2. Elegir almacén del escritorio o archivo privado sin cifrar explícitamente.
   Guardar inicia una sesión local de diez minutos; no concede permisos a flujos.
3. Pulsar Abrir consentimiento en el navegador y revisar los permisos en Google.
   La contraseña de la cuenta se introduce únicamente en el sitio del proveedor.
4. Volver al panel y comprobar el resultado. Cancelar autorización detiene espera
   o canje. Tras un éxito, la conexión aparece en Credenciales.
5. En un destino permitido, seleccionar `oauth2-google` y referenciar el ID de
   conexión. Revisar y conceder los permisos del flujo por separado.

El modo Google OAuth2: importar token guarda una autorización existente sin
contactar al proveedor. No verifica los scopes de ese token al importarlo.
Campos sensibles se enmascaran y el formulario se limpia al guardar/cancelar.
No se distribuyen credenciales Google propias del proyecto.

## Sesión y recuperación

El protocolo usa PKCE S256, state/verifier independientes de 256 bits y un solo
canje. El receptor escucha exclusivamente en `127.0.0.1` con puerto efímero y ruta
`/oauth/callback`. Comprueba host, ruta, método GET, state, plazo y parámetros
críticos únicos. Una respuesta malformada no consume la sesión. El canje inicial
exige refresh token y todos los scopes solicitados.

El receptor limita cabeceras a 16 KiB, lectura/escritura a tres segundos,
cabeceras a dos y desactiva keep-alive. No acepta cuerpos ni registra URLs/códigos.
La respuesta fija incluye inglés/español, no-store, no-referrer y CSP sin recursos.
Finalizar/cancelar/caducar cierra el puerto; tras un segundo de cierre ordenado
se cierran forzosamente conexiones pendientes.

Recargar el panel no pierde la sesión del motor: actualizar Seguridad recupera
su estado y, solo mientras espera consentimiento, el enlace. Si una consulta
falla, se detiene el sondeo y se indica que hay que actualizar Seguridad.
Reiniciar el motor descarta sesiones: se debe iniciar otra autorización.
Cerrar el motor cancela y espera su sesión antes de cerrar SQLite.

Cancelación y guardado se serializan. Una cancelación aceptada mientras espera o
intercambia impide persistir una respuesta tardía. Antes de guardar se compara la
credencial con la observada al comenzar; si apareció, desapareció o cambió, se
rechaza el resultado. Cancelar no revoca una conexión que ya se guardó.

## Tokens, scopes y almacenamiento

Client ID, client secret, refresh/access tokens, expiración y scopes se guardan
juntos en el backend elegido, dentro del límite de 8192 bytes del almacén. No se
exportan en configuración ni se devuelven por API. Estado devuelve únicamente
metadatos, incluidos expiración y scopes solicitados/concedidos.

Renovaciones se serializan. Cada solicitud relee el almacén, sin caché independiente.
Un token con más de 60 segundos restantes se reutiliza; renovar persiste los tokens
antes de devolver el access token. Cambiar/borrar credenciales no queda bloqueado
por la red: el guardado posterior compara el valor leído y descarta respuestas
obsoletas. No hay fallback automático de backend.

Los scopes se comparan literalmente; no se traducen alias históricos. Consentimiento
parcial o campo scope ausente en el canje inicial termina `scopes_missing`, sin guardar.
Scopes adicionales se conservan como metadatos, sin conceder permisos de flujo.
Una reducción explícita durante renovación bloquea el uso hasta reconectar; si la
renovación omite scopes, conserva los conocidos. El parser admite hasta 8192 bytes,
64 scopes y 512 bytes por scope, sin duplicados ni controles.

## Destinos y fallos

Destinos iniciales: hosts exactos `www.googleapis.com` y `calendar.googleapis.com`,
HTTPS con puerto 443 o implícito, sin userinfo, fragmentos ni excepciones LAN.
La validación se repite al emitir. Transporte público con DNS validado/pinning,
sin proxy ambiental ni redirecciones. El token endpoint es fijo:
`https://oauth2.googleapis.com/token`. Timeout de 15 segundos y respuesta de 32 KiB.

El permiso del flujo liga destino, modo OAuth2 e ID de conexión. El emisor adjunta
solo el access token. Rechaza usar un conjunto OAuth2 válido como Bearer/HMAC genérico.
Errores son estáticos; no devuelve cuerpos del proveedor ni detalles del transporte.
Errores temporales/cambios concurrentes permiten reintentos de la cola; errores
terminales requieren intervención. `invalid_grant` requiere reconectar y no distingue
por sí solo expiración de revocación remota.

Un HTTP 401 termina la entrega sin repetirla. Marca `token_rejected` y borra el access
token solo si coincide la revisión privada usada al enviar. Conserva refresh token
para una entrega posterior. Un rechazo tardío no altera credenciales reemplazadas.
Si guardar el estado falla, el resultado terminal lo indica sin exponer secretos.
Eliminar la credencial local **no revoca** el consentimiento en Google.

Estados de conexión: unavailable, renewal_required, available, expired,
reconnect_required, client_rejected, temporary_error, invalid_response,
token_rejected y scopes_missing. Consultarlos no contacta a Google ni prueba que
un token siga siendo válido remotamente.

## API local

| Operación | Entrada y resultado |
| --- | --- |
| `oauth.google.put` | ID, backend, client_id, client_secret, refresh_token; devuelve ID/proveedor/estado. |
| `oauth.google.status` | ID; estado, expiración y scopes, sin secretos. |
| `oauth.google.begin` | ID, backend, client_id, client_secret, scopes; sesión, ID, estado y URL. |
| `oauth.google.current` | Objeto vacío; sesión actual, o estado none. URL solo durante espera. |
| `oauth.google.session` | session; ID y estado. |
| `oauth.google.cancel` | session; cancela espera/canje y devuelve estado. |
| `secrets.delete` | ID; elimina la conexión local. |

Solo una sesión puede estar en curso. Estados: waiting, exchanging, completed,
cancelled, denied, failed, credential_changed, storage_failed, scopes_missing.
Una nueva sesión puede sustituir la referencia de una finalizada. La URL contiene
client ID, scopes, state y challenge; nunca client secret, verifier, código o tokens.

## Verificación y límites pendientes

La suite cubre PKCE, replay, callbacks, sockets y cierres, canje/renovación,
concurrencia, persistencia, sustitución/borrado, reducción de scopes, límites,
redacción, destinos y envío por worker. QML cubre importación, inicio, recuperación
y cancelación con cliente ficticio, sin abrir navegador.

Falta verificar apertura real del navegador y consentimiento con un cliente/cuenta
Google del usuario; las pruebas simuladas no acreditan ese recorrido. El backend
keyring reutiliza el almacén existente, pero estos recorridos OAuth2 persistentes
se probaron con archivos privados temporales. R1.UI está cerrada; las pruebas reales OAuth/sesión y el cierre estable siguen pendientes. Las evidencias concretas figuran en `docs/validation.md`.
