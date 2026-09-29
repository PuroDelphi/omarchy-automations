# Payload formats and conditions

[Español](../es/payload-recipes.md) · [User guide](user-guide.md) · [Options](options.md)

An entry has one fixed format. Authentication verifies the original HTTP bytes
before parsing. Choose the matching Content-Type and configure a flow for
`entry:YOUR-ID`. The following bodies are alternatives, not full configurations.
[Exact request fixtures](../../examples/payload-formats.json) contain all ten
bodies, Content-Type values and expected parsed fields. This fixture file is for
reference/testing; do not import it as plugin configuration.

| Input | Example A | Example B | Fields available to conditions/actions |
|---|---|---|---|
| JSON | `{"state":"failed","count":3}` | `{"enabled":true,"message":"Ready"}` | `data.count` is number 3; `data.enabled` is boolean true. |
| Form | `state=failed&count=3` | `message=Build+ready&project=alpha` | `data.count` is text `3`; `data.message` is text `Build ready`. |
| Raw | `Build ready` with text/plain | A CSV body with text/csv | `data.body` contains text; `data.content_type` is the parsed media type without parameters. |
| XML | `<event state="failed"><message>Build failed</message></event>` | `<event><value>10</value><value>20</value></event>` | `data.xml.attributes.state` is `failed`; `data.xml.children.1.text` is text `20`. |
| Multipart | Text part named `state`, value `failed` | File part named `report`, filename report.txt, contents `OK` | `data.fields.state` is `failed`; `data.files.report.base64` is `T0s=` and size is 2 bytes. |

JSON requires an object and application/json or a +json type. Form requires
application/x-www-form-urlencoded; duplicate field names are rejected, with at
most 128 fields. Raw still requires a syntactically valid Content-Type. CSV is
not parsed into columns automatically. XML needs application/xml, text/xml or
+xml; multipart needs multipart/form-data with its boundary. Do not sign a body
and then change its whitespace or encoding before sending it.

The HTTP body limit is 256 KiB. XML rejects DTD/entity declarations and arbitrary
processing instructions; limits include 16 levels, 512 elements and 32 attributes
per element. Multipart permits at most 16 parts, text parts up to 16 KiB and files
up to 64 KiB each, with unique names. Filenames cannot contain path separators.
XML/multipart normalized JSON is additionally capped at 240 KiB, including base64
expansion. Received files remain event data; they are not written or executed as
host files. XML text remains text: `"20"` is not numeric 20.

## Condition examples

All conditions in a flow must match. The source must match exactly and the flow
must be enabled. These examples assume the named event fields exist:

| Operator | Example A | Example B |
|---|---|---|
| `eq` | `data.state`, Text, `failed` | `data.enabled`, Boolean, `true` |
| `ne` | `data.state`, Text, `passed` | `data.enabled`, Boolean, `false` |
| `gt` | `data.count`, Number, `2` | `data.temperature`, Number, `80` |
| `lt` | `data.free`, Number, `10` | `data.age`, Number, `3600` |
| `contains` | `data.message`, Text, `Build` | `data.project`, Text, `release-` |

`contains` is a case-sensitive substring test, not a regex or array membership.
A missing field fails every operator, including `ne`. Numeric comparisons are
strict; equality does not satisfy `gt` or `lt`. Choose Number only for numeric
payload fields: form `count=3` does not match numeric `gt 2`. Use JSON numbers or
an explicitly approved adapter when you need a type conversion. Dot paths can
select ordinary array indices (`data.items.0.name`), not `01`, negative indices
or arbitrary expressions. Keys containing dots cannot be escaped in this path
syntax; normalize such data with an adapter if necessary.

To verify without effects, enable your flow in the draft, then simulate its exact
source/type with normalized JSON data. For XML example A use
`{"xml":{"attributes":{"state":"failed"}}}` to test that condition. Simulation
checks conditions/templates, not XML parsing or authentication. Actual HTTP
requests exercise those boundaries and can execute active flows.

## Outbound formats

Register a destination using POST, PUT or PATCH as required by its API. Set the
HTTP action body for that destination's selected format:

| Output | Example A | Example B |
|---|---|---|
| JSON | `{"state":"{{type}}"}` | `{"count":"{{data.count}}"}` |
| Form | `{"project":"{{data.project}}"}` | `{"enabled":true,"count":3}` |
| Raw | `Build: {{data.message}}` | `State: {{type}}` |

An exact JSON field placeholder preserves its value type: numeric count stays
numeric. Form output first resolves a JSON object and then URL-encodes scalar
fields; arrays/objects as form values are rejected. Raw uses text/plain. None of
these formats accepts an arbitrary remote URL from event data: the registered
destination and its permissions still apply. Check the resolved body in simulation
and the receiver/History for real delivery; a local preview proves no delivery.
