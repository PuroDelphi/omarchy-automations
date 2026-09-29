# Inbound formats

[Español](../es/formats.md) · [Documentation](README.md)

Each entry selects a fixed format. The receiver authenticates original bytes
before parsing and accepts HTTP bodies up to 256 KiB. Invalid input creates no
event. XML and multipart are available in connection forms; outbound formats
remain JSON, URL-encoded form and raw.

## XML

Content types: `application/xml`, `text/xml` and `+xml` suffix types. Parsing uses
UTF-8 and rejects DTDs, entity declarations, processing instructions other than
the XML declaration, and malformed documents. There is no file/network resolver.
Limits: 16 levels, 512 elements, 32 attributes per element and 2048 tokens.

Input:

```xml
<event status="failed"><message>Deployment failed</message></event>
```

Normalized data:

```json
{"xml":{"name":"event","attributes":{"status":"failed"},"text":"","children":[{"name":"message","attributes":{},"text":"Deployment failed","children":[]}]}}
```

Conditions can read `data.xml.attributes.status`; actions can use
`{{data.xml.children.0.text}}`. Array indexes start at zero and accept only
nonnegative integers in ordinary decimal notation. Repeated elements retain order.
For mixed content, `text` concatenates direct element text without retaining its
position relative to children. Comments are omitted. Namespaced names use
`{URI}name`. This is not XPath or a SOAP implementation.

## Multipart

Content type: `multipart/form-data` with an explicit boundary up to 70 bytes.
Limits: 16 parts, eight headers per part and unique names up to 128 bytes.
Duplicate headers and `Content-Transfer-Encoding` are rejected.

- Fields: UTF-8 text up to 16 KiB, under `data.fields.<name>`.
- Files: up to 64 KiB each, under `data.files.<name>`, with `filename`,
  `content_type`, `size` and `base64`.
- Filenames: at most 255 bytes; no path separators, NUL, CR/LF, `.` or `..`.

Files are read into memory and persisted as part of the SQLite event. No temporary
files are created, and received content is neither executed nor decompressed.
Content-type metadata does not verify the file's actual type. Normal event quotas
and retention apply.

Normalized XML/multipart has an additional 240 KiB limit to leave room for the
event envelope. Base64 expansion and JSON escaping count, so a body below 256 KiB
can still be rejected.

## References and tests

Parsing uses Go's standard [encoding/xml](https://pkg.go.dev/encoding/xml) and
[mime/multipart](https://pkg.go.dev/mime/multipart). Additional limits/rejections
are Omarchy Automations policies covered by `internal/core/formats_test.go`.
See [payload recipes](payload-recipes.md) for two examples per input format.
