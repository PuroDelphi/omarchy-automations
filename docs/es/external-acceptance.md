# Pruebas de aceptación con el usuario

[English](../en/external-acceptance.md) · [Guía](user-guide.md)

Estas pruebas completan F3.3 y F2.7. Preparar el procedimiento no equivale a
superarlo. Registra fecha, versión, resultado y errores redactados; nunca tokens,
client secret, URL de autorización completa ni contenido del almacén privado.

## Google real

1. En Google Cloud selecciona tu proyecto de prueba y habilita Calendar API.
   Configura Google Auth Platform para tu cuenta; si la aplicación es externa y
   está en pruebas, añade tu cuenta como usuario de prueba.
2. En Google Auth Platform → Clients → Create client elige **Desktop app**.
   Guarda client ID y client secret localmente fuera del repositorio.
   Consulta la [guía oficial](https://developers.google.com/workspace/guides/create-credentials).
3. En el plugin, Security → Create or rotate credential, selecciona autorización
   Google OAuth2 en navegador, ID `google-calendar`, cliente propio y scope
   `https://www.googleapis.com/auth/calendar.freebusy`. Selecciona el almacén
   del escritorio y completa su desbloqueo si lo solicita. Un fallo de este
   almacén no se considera resuelto eligiendo silenciosamente archivo privado.
4. Guarda y abre el consentimiento con el botón del plugin. Inicia sesión,
   comprueba el permiso y vuelve al panel. Espera estado `available` y comprueba
   los scopes concedidos. No basta con tener una sesión de Google en el navegador.
5. Sigue [el ejemplo Calendar](oauth-example.md) para preservar tu borrador,
   importar, simular, revisar y activar el flujo. Ejecuta una consulta de prueba
   con un calendario accesible y verifica entrega completada y estado HTTP.
   El plugin descarta el cuerpo: esto no acredita resultados por calendario.
6. Registra `expires_at` mediante el estado OAuth. Tras la expiración real del
   access token, ejecuta otra consulta. Debe completarse sin nuevo consentimiento
   y presentar una expiración posterior. No adelantes el reloj del sistema ni
   edites el secreto para simular esta comprobación.
7. Tras el ciclo de sesión descrito abajo, comprueba que la conexión persiste y
   que una nueva consulta funciona con el almacén del escritorio desbloqueado.
8. Al terminar, deshabilita el flujo y activa la revisión. Conserva la conexión
   solo si quieres utilizarla; eliminarla localmente no revoca consentimiento
   en Google. La revocación remota se gestiona en tu cuenta Google.

Evidencia mínima: versión, backend elegido, estado/scopes sin secretos, primer
resultado HTTP, expiraciones antes/después, segunda entrega y persistencia tras
sesión. Los ensayos locales de revocación, errores y redacción complementan esta
prueba; no sustituyen el consentimiento y renovación reales.

## Suspensión y cierre de sesión

Realiza cada transición por separado y guarda antes tu trabajo. El cierre de
sesión puede interrumpir esta conversación. No se ejecuta automáticamente.

1. Antes, registra estado del motor, configuración activa y borrador, permisos,
   cola e identificadores de ejecuciones existentes. Exporta el borrador para
   recuperación y conserva la evidencia local fuera del repositorio.
2. Suspende desde el menú habitual de Omarchy y reanuda. Abre el plugin y verifica
   que responde, conserva idioma/configuración/permisos, vuelve a escuchar y
   muestra correctamente las ejecuciones previas. Revisa Historial antes de
   repetir cualquier efecto cuyo resultado aparezca incierto.
3. Ejecuta una nueva prueba identificable de un flujo revisado y comprueba un
   único resultado esperado. Compara la cola con la evidencia anterior. La
   ausencia de duplicados requiere inspeccionar también el receptor si hay HTTP.
4. Guarda de nuevo el trabajo, cierra sesión desde Omarchy e inicia sesión.
   Repite las comprobaciones y una prueba nueva con un identificador diferente.
5. Registra errores de arranque, pérdida de configuración, repeticiones o estados
   inciertos. Una cola inicialmente vacía solo acredita recuperación en reposo;
   no demuestra recuperación de acciones en curso. Las pruebas automatizadas
   de caída cubren otras fronteras, pero deben citarse por separado.

No declares F2.7 superada solo por ver un proceso activo: compara estado antes y
después y resultados de las pruebas. No declares F3.3 superada hasta observar
consentimiento, uso, renovación y persistencia reales.
