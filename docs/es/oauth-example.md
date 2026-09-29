# Google OAuth: solicitud autenticada a Calendar

[English](../en/oauth-example.md) · [Guía de usuario](user-guide.md) · [Otros ejemplos](use-cases.md)

El ejemplo envía consulta de disponibilidad sin crear eventos de calendario.
Demuestra POST autenticado de salida. El plugin registra estado de entrega,
**no el cuerpo de respuesta**: no muestra disponibilidad, no pasa datos devueltos
a pasos posteriores ni inspecciona errores por calendario dentro de una respuesta
exitosa. Entrega completada no demuestra éxito a nivel de calendario.

## Preparar tu cliente Google

Usa tu proyecto Google Cloud, habilita Calendar API y configura consentimiento
OAuth para tu cuenta de prueba. Crea credenciales de tipo **Desktop app** y conserva
client ID y client secret localmente. Se aplican restricciones de cuenta/proyecto
y consentimiento. Sigue la [configuración oficial para aplicaciones instaladas](https://developers.google.com/identity/protocols/oauth2/native-app).

No pegues credenciales/tokens en chat, código ni configuración importable. El plugin
no distribuye cliente Google propio. La integración con cuenta real aún no se ha
verificado para esta entrega; las fixtures locales no cierran ese requisito.

## Autorizar desde el plugin

1. Abre Seguridad → Crear o rotar credencial.
2. Selecciona Google OAuth2: autorizar en navegador. Indica ID local `google-calendar`,
   client ID/secret Desktop propios y scope
   `https://www.googleapis.com/auth/calendar.freebusy`.
3. Elige almacén del escritorio si está disponible. El archivo privado alternativo
   está expresamente sin cifrar; no se selecciona automáticamente si falla el otro.
4. Guarda para iniciar sesión de autorización de diez minutos. Pulsa Abrir
   consentimiento en el navegador, elige cuenta de prueba y revisa permisos.
5. Vuelve a Seguridad e inspecciona estado. Actualizar recupera sesión pendiente
   tras recargar panel; reiniciar motor exige otra sesión. Cancelar autorización
   detiene la pendiente pero no revoca conexión ya guardada.

Si ya tienes refresh token adecuado de ese cliente, Google OAuth2: importar token
es otra opción. Guarda client ID, client secret y refresh token sin contactar Google
ni verificar scopes al importar. No sustituyas refresh token por access token.

## Importar, simular y activar

Exporta el borrador que quieras conservar e importa
[google-oauth.json](../../examples/use-cases/google-oauth.json). Contiene destino
`google-freebusy`, autenticación `oauth2-google`, referencia `google-calendar` y
flujo deshabilitado `calendar-query`, origen `local:calendar`.

El destino usa POST `https://www.googleapis.com/calendar/v3/freeBusy`, que admite
el scope anterior. Recibe intervalo e identificadores de calendarios.
[Contrato oficial Freebusy](https://developers.google.com/workspace/calendar/api/v3/reference/freebusy/query).

1. Habilita flujo y guarda. Simula origen `local:calendar`, tipo `test`, con:

   ```json
   {
     "start": "2030-01-07T09:00:00Z",
     "end": "2030-01-07T10:00:00Z",
     "calendar": "calendar-a@example.com"
   }
   ```

   Debe resolver cuerpo con `timeMin`, `timeMax` y un `items[].id`. El calendario
   del ejemplo es ficticio. Simular no necesita token Google.
2. Cambia solo `calendar` a `calendar-b@example.com` para segunda simulación:
   debe resolver identificador nuevo. Comprueba plantilla, no acceso real.
3. Para solicitud real sustituye identificador por uno accesible a tu cuenta de
   prueba y elige intervalo que quieras consultar. Revisa destino Google fijo,
   referencia y capacidad del flujo; después activa.
4. Confirma Ejecutar prueba real. Consulta estado HTTP y finalización en Historial.
   Se descarta cuerpo de respuesta Google; inspecciona resultados específicos de
   API por separado si necesitas confirmar disponibilidad o errores por calendario.

No añadas Authorization manualmente. El motor obtiene access token del almacén
privado elegido y lo añade al destino Google admitido. Este modo solo admite
`www.googleapis.com` y `calendar.googleapis.com` por HTTPS puerto 443, sin excepciones
privadas ni redirecciones. Consentimiento y permiso del flujo son aprobaciones separadas.

## Resolver problemas y desconectar

- Token disponible localmente no prueba validez remota. Estado OAuth2 lee metadatos
  sin contactar Google.
- `reconnect_required` o `scopes_missing` necesitan autorización nueva adecuada;
  comprueba que se concedieran todos los scopes solicitados.
- `client_rejected` exige revisar cliente. Fallos temporales del proveedor pueden
  reintentarse al gestionar entregas pendientes.
- HTTP 401 termina esa entrega sin repetir inmediatamente. Inspecciona conexión
  antes de pedir reenvío HTTP manual elegible.
- Eliminar `google-calendar` impide autenticaciones posteriores con esa referencia,
  pero no revoca consentimiento Google. Retira también el consentimiento del
  proveedor si eso es lo que pretendes.

## Alcance de verificación

`TestDocumentedGoogleOAuth` valida JSON distribuido y simula ambos identificadores
sin credenciales. Con credencial ficticia y transporte controlado comprueba método,
URL exacta, cabecera de access token y cuerpo. Eliminar credencial impide otra llamada
al transporte. No realiza consentimiento real, renovación con Google, comunicación
TLS/DNS ni consulta real de calendario. La prueba integral de cuenta sigue pendiente
bajo F3.3.
