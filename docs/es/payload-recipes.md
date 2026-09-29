# Formatos de datos y condiciones

[English](../en/payload-recipes.md) · [Guía de usuario](user-guide.md) · [Opciones](options.md)

Cada entrada tiene un formato fijo. La autenticación comprueba los bytes HTTP
originales antes de interpretarlos. Elige el Content-Type correspondiente y un
flujo para `entry:TU-ID`. Los cuerpos siguientes son alternativas, no configuraciones
completas. Los [ejemplos exactos de peticiones](../../examples/payload-formats.json)
contienen los diez cuerpos, tipos de contenido y campos esperados. Este archivo
sirve de referencia/prueba; no lo importes como configuración del plugin.

| Entrada | Ejemplo A | Ejemplo B | Campos disponibles para condiciones/acciones |
|---|---|---|---|
| JSON | `{"state":"failed","count":3}` | `{"enabled":true,"message":"Ready"}` | `data.count` es número 3; `data.enabled` es booleano true. |
| Formulario | `state=failed&count=3` | `message=Build+ready&project=alpha` | `data.count` es texto `3`; `data.message` es texto `Build ready`. |
| Raw | `Build ready` con text/plain | Un cuerpo CSV con text/csv | `data.body` contiene texto; `data.content_type` es el tipo interpretado sin parámetros. |
| XML | `<event state="failed"><message>Build failed</message></event>` | `<event><value>10</value><value>20</value></event>` | `data.xml.attributes.state` es `failed`; `data.xml.children.1.text` es texto `20`. |
| Multipart | Parte de texto `state`, valor `failed` | Parte de archivo `report`, nombre report.txt, contenido `OK` | `data.fields.state` es `failed`; `data.files.report.base64` es `T0s=` y su tamaño es 2 bytes. |

JSON exige un objeto y application/json o un tipo +json. Formulario exige
application/x-www-form-urlencoded; rechaza nombres de campo duplicados y admite
hasta 128 campos. Raw también exige un Content-Type sintácticamente válido. No
interpreta CSV como columnas automáticamente. XML requiere application/xml,
text/xml o +xml; multipart requiere multipart/form-data con su boundary. No firmes
un cuerpo para después cambiar sus espacios o codificación antes de enviarlo.

El límite HTTP es 256 KiB. XML rechaza DTD/declaraciones de entidades e instrucciones
de procesamiento arbitrarias; limita a 16 niveles, 512 elementos y 32 atributos
por elemento. Multipart admite hasta 16 partes, texto de hasta 16 KiB y archivos
de hasta 64 KiB cada uno, con nombres únicos. Los nombres de archivo no pueden
contener separadores de ruta. El JSON normalizado XML/multipart tiene además un
límite de 240 KiB, incluida la expansión base64. Los archivos recibidos permanecen
como datos del evento; no se escriben ni ejecutan como archivos del host. El texto
XML sigue siendo texto: `"20"` no es el número 20.

## Ejemplos de condiciones

Todas las condiciones de un flujo deben coincidir. La fuente debe coincidir
exactamente y el flujo debe estar habilitado. Estos ejemplos suponen que existen
los campos del evento indicados:

| Operador | Ejemplo A | Ejemplo B |
|---|---|---|
| `eq` | `data.state`, Texto, `failed` | `data.enabled`, Booleano, `true` |
| `ne` | `data.state`, Texto, `passed` | `data.enabled`, Booleano, `false` |
| `gt` | `data.count`, Número, `2` | `data.temperature`, Número, `80` |
| `lt` | `data.free`, Número, `10` | `data.age`, Número, `3600` |
| `contains` | `data.message`, Texto, `Build` | `data.project`, Texto, `release-` |

`contains` busca una subcadena distinguiendo mayúsculas; no es regex ni pertenencia
a un array. Un campo ausente falla para todos los operadores, incluido `ne`.
Las comparaciones numéricas son estrictas; igualdad no satisface `gt` ni `lt`.
Elige Número solo para campos numéricos: formulario `count=3` no coincide con
`gt 2` numérico. Usa números JSON o un adaptador aprobado explícitamente si necesitas
convertir tipos. Las rutas con puntos permiten índices ordinarios (`data.items.0.name`),
no `01`, negativos ni expresiones arbitrarias. Esta sintaxis no permite escapar
puntos dentro de claves; normaliza esos datos con un adaptador si es necesario.

Para comprobar sin efectos, habilita el flujo en el borrador y simula su fuente/tipo
exactos con datos JSON normalizados. Para XML A usa
`{"xml":{"attributes":{"state":"failed"}}}` al probar esa condición. Simular
comprueba condiciones/plantillas, no el análisis XML ni la autenticación. Las
peticiones HTTP reales prueban esos límites y pueden ejecutar flujos activos.

## Formatos de salida

Registra un destino con POST, PUT o PATCH según su API. Configura el cuerpo de la
acción HTTP para el formato elegido en ese destino:

| Salida | Ejemplo A | Ejemplo B |
|---|---|---|
| JSON | `{"state":"{{type}}"}` | `{"count":"{{data.count}}"}` |
| Formulario | `{"project":"{{data.project}}"}` | `{"enabled":true,"count":3}` |
| Raw | `Build: {{data.message}}` | `State: {{type}}` |

Una sustitución exacta de campo JSON conserva el tipo: count numérico sigue siendo
numérico. La salida de formulario primero resuelve un objeto JSON y después
codifica sus campos escalares como URL; rechaza arrays/objetos como valores.
Raw usa text/plain. Ningún formato acepta una URL remota arbitraria desde datos
del evento: siguen aplicándose el destino registrado y sus permisos. Comprueba
el cuerpo resuelto en simulación y el receptor/Historial para entregas reales;
una vista previa local no demuestra entrega.
