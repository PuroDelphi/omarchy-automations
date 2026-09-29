# Formatos de entrada

[English](../en/formats.md) · [Documentación](README.md)

Cada entrada selecciona un formato fijo. El receptor autentica los bytes
originales antes de interpretarlos y admite un cuerpo HTTP de hasta 256 KiB.
Una entrada inválida no crea eventos. XML y multipart están disponibles en el
formulario de conexiones; la salida sigue admitiendo JSON, formulario y raw.

## XML

Tipos de contenido: `application/xml`, `text/xml` y tipos con sufijo `+xml`.
El parser usa UTF-8 y rechaza DTD, declaraciones de entidades, instrucciones
de procesamiento distintas de la declaración XML y documentos malformados.
No tiene un resolvedor de archivos ni de red. La estructura permite hasta
16 niveles, 512 elementos, 32 atributos por elemento y 2048 tokens.

Ejemplo de entrada:

```xml
<event status="failed"><message>Falló el despliegue</message></event>
```

Datos normalizados:

```json
{"xml":{"name":"event","attributes":{"status":"failed"},"text":"","children":[{"name":"message","attributes":{},"text":"Falló el despliegue","children":[]}]}}
```

Una condición puede consultar `data.xml.attributes.status`. Una acción puede
usar `{{data.xml.children.0.text}}`. Los índices de arrays empiezan en cero;
solo se admiten enteros no negativos en su representación decimal habitual.
Los elementos repetidos conservan su orden. En contenido mixto, `text` concatena
el texto directo del elemento; no conserva su posición relativa a los hijos.
Comentarios no se incluyen. Los nombres con namespace usan `{URI}nombre`.
No es una implementación de XPath ni de SOAP.

## Multipart

Tipo de contenido: `multipart/form-data`, con boundary explícito de hasta
70 bytes. Hasta 16 partes, 8 cabeceras por parte y nombres únicos de hasta
128 bytes. Se rechazan cabeceras repetidas y `Content-Transfer-Encoding`.

- Campos: texto UTF-8 de hasta 16 KiB, accesible como `data.fields.<nombre>`.
- Archivos: hasta 64 KiB cada uno, accesibles como `data.files.<nombre>`.
  Incluyen `filename`, `content_type`, `size` y `base64`.
- Nombres de archivo: máximo 255 bytes; sin separadores de ruta, NUL, CR/LF
  ni los nombres `.` y `..`.

Los archivos se leen en memoria y se guardan como parte del evento en SQLite.
No se crean archivos temporales ni se ejecuta o descomprime contenido recibido.
Los metadatos de tipo de contenido no constituyen una verificación del archivo.
Se aplican las mismas cuotas y retención de eventos que a los demás formatos.

XML y multipart normalizados tienen un límite adicional de 240 KiB para dejar
espacio al sobre del evento. La expansión base64 y el escape JSON cuentan en
ese límite; por eso un cuerpo HTTP que cabe en 256 KiB puede ser rechazado.

## Referencias y pruebas

Se utilizan los parsers de la biblioteca estándar de Go:
[encoding/xml](https://pkg.go.dev/encoding/xml) y
[mime/multipart](https://pkg.go.dev/mime/multipart).
Los límites adicionales y los rechazos son políticas propias de Omarchy Automations,
comprobadas en `internal/core/formats_test.go`.
