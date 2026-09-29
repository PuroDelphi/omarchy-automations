# Recuperar una entrega HTTP fallida

[English](../en/recovery-example.md) · [Guía de usuario](user-guide.md) · [Otros ejemplos](use-cases.md)

Usa un receptor que controles, capaz de devolver estados HTTP elegidos y registrar
cuerpo y `Idempotency-Key`. Usa un endpoint de prueba: los reintentos pueden repetir
efectos externos. Este ejemplo no proporciona ni publica un servidor receptor.

## Preparar el flujo

1. Exporta el borrador que quieras conservar e importa
   [http-recovery.json](../../examples/use-cases/http-recovery.json).
2. Sustituye `https://receiver.example.com/events` por URL HTTPS confiable del
   receptor. Crea `receiver-token` en Seguridad con su valor bearer esperado.
   El hostname del ejemplo es un marcador, no una API operativa.
3. Habilita `send-demo` y guarda. Simula origen `local:delivery`, tipo `test` y
   `{"message":"Build finished","request":"demo-1"}`. Debe resolver ambos campos
   en el JSON de salida sin enviar nada.
4. Repite con `{"message":"Backup verified","request":"demo-1"}` para comprobar
   segundo mensaje. Revisa destino, referencia de autenticación y capacidad de
   acción, y activa.

Para receptor privado, configura excepción host:puerto exacta y certificado TLS
confiable; no desactives verificación. Estos ajustes forman parte de la capacidad
revisada.

## Observar un reintento automático

Configura receptor para responder HTTP 503 y confirma prueba real con el primer
cuerpo. La ejecución debe quedar pendiente con datos de intento en Historial.
El motor espera antes de repetir; no envía inmediatamente otro evento. Errores de
transporte, 408, 429 y 5xx son elegibles dentro de límites de ocho intentos y 24 horas.
La espera guardada sobrevive al reinicio del motor.

En este ejercicio, cambia receptor a HTTP 400 antes del próximo intento. Después
debe quedar ejecución fallida terminal. HTTP 400 ordinario no se reintenta
automáticamente. Abre Detalles y revisa estado/intentos; aceptar el evento local
no demuestra éxito de la entrega.

## Solicitar un reenvío manual

1. Corrige receptor para aceptar esta solicitud y responder HTTP 204.
2. Pausa el motor. En Historial desactiva filtro de solo pendientes/inciertos para
   localizar ejecución fallida y pulsa Reenviar HTTP.
3. Revisa confirmación y acepta. La misma ejecución pasa a pendiente; permanece
   pausada hasta Reanudar. Cancelar el diálogo la habría dejado fallida.
4. Reanuda. Debe entregarse correctamente y terminar completada. Compara registros
   del receptor: cuerpo e `Idempotency-Key` deben coincidir con originales.

Reenviar manualmente reinicia contador de intentos y plazo de esa entrega; el
contador no suma todos los ciclos manuales. No genera cuerpo nuevo a partir del
borrador actual. Editar mensaje del borrador no repara el cuerpo ya persistido.
Si el cuerpo está mal, corrige y revisa configuración y envía deliberadamente un
evento nuevo; no supongas que Reenviar HTTP lo reescribe. Revisa primero posibles
efectos anteriores en el receptor.

El receptor debe implementar idempotencia. Reutilizar clave por sí solo no impide
que procese dos veces la operación. Rotar credencial puede corregir autenticación
en un intento posterior, pero no deshace solicitudes enviadas. Cambiar destino/
configuración y permisos puede bloquear trabajo antiguo en lugar de redirigirlo
al destino nuevo.

## Resultados inciertos y acciones no HTTP

Reenviar HTTP solo admite pasos HTTP fallidos/cancelados elegibles. No reintenta
entregas completadas, comandos, scripts, servicios ni resultados inciertos arbitrarios.
Un resultado incierto significa que el efecto pudo ocurrir sin confirmación guardada.
Inspecciona receptor/servicio/archivos implicados antes de decidir iniciar otro evento.
No interpretes incertidumbre como prueba de que nada se ejecutó.

Pausa, cancelación y revocación son diferentes: pausa conserva trabajo; cancelar
no revierte efectos enviados; revocar impide despachos futuros bajo ese permiso.
Cerrar panel no hace ninguna y deja motor funcionando.

## Alcance de verificación

`TestDocumentedHTTPRecovery` carga configuración distribuida y prueba ambos
mensajes simulados. Su emisor controlado responde 503, 400 y 204 mientras worker/
outbox reales gestionan espera, fallo terminal y reenvío manual. Verifica pausa,
cuerpo/clave estables pese a cambios del borrador, finalización y rechazo a repetir
entrega completada. Para evitar espera real adelanta el vencimiento persistido de
su propia prueba. No usa red externa ni acredita idempotencia del receptor o
configuración bearer/TLS del despliegue.
