# Option reference

[EN / ES](../es/options.md) · [Guide / Guía](user-guide.md)

This installment covers shared resource fields, entries, destinations, flows, conditions, schedules, monitors, actions, scripts and adapters. Includes credentials, retention, operating controls, simulation and preparation. Semantic review and verification scope are recorded in docs/documentation-audit.md. Defaults describe a new UI form, not implicit defaults for every imported JSON field. Examples are independent values, not complete configurations. Resource references must exist. Technical tokens remain identical in both languages.

- [common](#group-common) (3)
- [entries](#group-entries) (3)
- [destinations](#group-destinations) (7)
- [flows](#group-flows) (3)
- [conditions](#group-conditions) (4)
- [timers](#group-timers) (6)
- [monitors](#group-monitors) (12)
- [actions](#group-actions) (16)
- [scripts](#group-scripts) (3)
- [adapters](#group-adapters) (5)
- [parameters](#group-parameters) (7)
- [arguments](#group-arguments) (2)
- [directories](#group-directories) (4)
- [credentials](#group-credentials) (8)
- [storage](#group-storage) (4)
- [panel](#group-panel) (11)
- [files](#group-files) (1)
- [history](#group-history) (4)
- [simulation](#group-simulation) (5)
- [security](#group-security) (9)
- [code](#group-code) (1)
- [administration](#group-administration) (1)

<a id="group-common"></a>
## common

<a id="common.id"></a>
### Identifier — `common.id`

**Initial value:** `""`.

Required, unique within a resource kind. Start with a lowercase letter; then lowercase letters, digits, dot, underscore or hyphen; 1–64 characters. Changing an ID requires updating references.

- Example 1: `deploy`.
- Example 2: `disk-space`.

<a id="common.name"></a>
### Name — `common.name`

**Initial value:** `""`.

Optional display name for entries, destinations, flows and schedules. It does not change the technical source or endpoint.

- Example 1: `Deployment alerts`.
- Example 2: `Daily reminder`.

<a id="common.enabled"></a>
### Enabled when the revision is activated — `common.enabled`

**Initial value:** `true`.

Applies to entries, flows, monitors and schedules. Saving does not activate it. Import forces false. A disabled flow will not match in simulation or dispatch.

- Example 1: `true`.
- Example 2: `false`.

<a id="group-entries"></a>
## entries

<a id="entries.auth"></a>
### Authentication — `entries.auth`

**Initial value:** `hmac`.

Choose hmac, github, slack or bearer. The sender must implement that exact contract; generic HMAC/Slack timestamps allow five minutes of skew. Slack accepts only json/form. Wrong signatures or missing credentials reject the request before event creation.

- Example 1: `hmac`.
- Example 2: `slack`.

<a id="entries.secret"></a>
### Credential reference — `entries.secret`

**Initial value:** `""`.

Required credential ID created in Security, using the same ID syntax as resources. Enter its reference, not its value. The credential must exist when authenticating real requests.

- Example 1: `deploy-signing`.
- Example 2: `slack-signing`.

<a id="entries.format"></a>
### Body format — `entries.format`

**Initial value:** `json`.

Choose json, form, raw, xml or multipart; the sender content type/body must agree. JSON must be an object; duplicate form names are rejected. Format is fixed per entry, not auto-detected. HTTP input is limited to 256 KiB.

- Example 1: `json`.
- Example 2: `multipart`.

<a id="group-destinations"></a>
## destinations

<a id="destinations.url"></a>
### HTTPS address — `destinations.url`

**Initial value:** `""`.

Required HTTPS URL with a host, without userinfo or fragment. No redirect following. Private addresses require the exact exception below; a certificate trusted by the system is still required. Example addresses must be replaced with your receiver.

- Example 1: `https://status.example.com/events`.
- Example 2: `https://receiver.example.com:8443/hook`.

<a id="destinations.method"></a>
### Method — `destinations.method`

**Initial value:** `POST`.

Choose POST, PUT or PATCH to match the receiver. The selected method is used for HTTP actions; connectivity monitors always use HEAD. An unsupported method is a configuration error.

- Example 1: `PUT`.
- Example 2: `PATCH`.

<a id="destinations.format"></a>
### Output format — `destinations.format`

**Initial value:** `json`.

Choose json, form or raw. JSON/form action bodies are valid JSON templates encoded after substitution; raw bodies are text. An omitted format in imported configuration behaves as JSON.

- Example 1: `form`.
- Example 2: `raw`.

<a id="destinations.auth"></a>
### Authentication — `destinations.auth`

**Initial value:** `""`.

Empty means no added authentication; other choices are hmac, bearer and oauth2-google. Authenticated modes require a credential reference. Google mode additionally restricts hosts and port.

- Example 1: `bearer`.
- Example 2: `oauth2-google`.

<a id="destinations.secret"></a>
### Credential reference — `destinations.secret`

**Initial value:** `""`.

Visible for authenticated destinations. Use an existing generic credential for HMAC/bearer or a Google connection for oauth2-google. A Google token bundle is not a generic bearer secret.

- Example 1: `status-token`.
- Example 2: `google-calendar`.

<a id="destinations.headers"></a>
### Additional headers — `destinations.headers`

**Initial value:** `{}`.

Name/value pairs sent with requests and exported with configuration. Do not put secrets here. Authorization, Host, Cookie, Proxy-* and CR/LF in names/values are rejected.

- Example 1: `{"X-Application":"omarchy"}`.
- Example 2: `{"X-Environment":"staging"}`.

<a id="destinations.private_hosts"></a>
### Private network exceptions — `destinations.private_hosts`

**Initial value:** `[]`.

Empty blocks private destinations. At most one exception, exactly matching this URL host and explicit port (443 when the URL omits a port); bracket IPv6. It authorizes local unicast only, not arbitrary networks, and never disables TLS. Each example requires its matching URL.

- Example 1: `["status.internal:443"]`.
- Example 2: `["[fd00::10]:8443"]`.

<a id="group-flows"></a>
## flows

<a id="flows.source"></a>
### When an event arrives from — `flows.source`

**Initial value:** `local:demo`.

Exact source: entry:ID, monitor:ID or timer:ID for a registered resource, or local:NAME/hook:NAME. An HTTP sender cannot call the local control API. Selecting a source does not enable it or grant its actions.

- Example 1: `entry:deploy`.
- Example 2: `monitor:disk-space`.

<a id="flows.conditions"></a>
### If these conditions match — `flows.conditions`

**Initial value:** `[]`.

Zero to 32 conditions; all must match. Use a field path, operator and typed scalar value. Missing fields never match. Paths are at most 128 characters with at most eight dots. See the nested condition options below.

- Example 1: `[{"field":"data.state","op":"eq","value":"failed"}]`.
- Example 2: `[{"field":"data.percent","op":"gt","value":90}]`.

<a id="flows.steps"></a>
### Run in this order — `flows.steps`

**Initial value:** `[]`.

Required list of 1–32 registered action IDs, executed in this order. An empty flow cannot be saved. Each action needs its reviewed grant; later steps can consume adapter output.

- Example 1: `["notice"]`.
- Example 2: `["normalize","report","notice"]`.

<a id="group-conditions"></a>
## conditions

<a id="conditions.field"></a>
### Field — `conditions.field`

**Initial value:** `data.state`.

Path in the event: source, type or fields under data. Array indices use ordinary nonnegative decimal notation. It selects data; it does not evaluate code.

- Example 1: `data.repository.full_name`.
- Example 2: `data.xml.children.0.text`.

<a id="conditions.op"></a>
### Operator — `conditions.op`

**Initial value:** `eq`.

eq/ne compare typed scalar values; gt/lt require numbers; contains requires text and checks a substring. A missing field fails even ne.

- Example 1: `eq`.
- Example 2: `contains`.

<a id="conditions.type"></a>
### Value type — `conditions.type`

**Initial value:** `string`.

Choose Text, Number or Boolean in the condition row. Types matter: text "90" is not number 90. Boolean input is true or false; selecting Boolean converts according to whether the current text is exactly true.

- Example 1: `number: 90`.
- Example 2: `boolean: true`.

<a id="conditions.value"></a>
### Value — `conditions.value`

**Initial value:** `failed`.

The value to compare with the selected field. The new-condition default is text failed. Choose the type before entering a numeric/boolean comparison.

- Example 1: `failed`.
- Example 2: `90 (number)`.

<a id="group-timers"></a>
## timers

<a id="timers.kind"></a>
### Schedules — `timers.kind`

**Initial value:** `interval`.

interval repeats after a number of seconds; calendar selects a local clock time in an explicit timezone. Changing the timer configuration creates a new schedule.

- Example 1: `interval`.
- Example 2: `calendar`.

<a id="timers.interval_seconds"></a>
### Interval in seconds (minimum 5) — `timers.interval_seconds`

**Initial value:** `60`.

For interval only: integer 5–31536000 seconds. First due time starts from the first evaluation; restarts preserve its stored next due time. It is not a real-time guarantee.

- Example 1: `300`.
- Example 2: `3600`.

<a id="timers.at"></a>
### Local time HH:MM — `timers.at`

**Initial value:** `09:00`.

For calendar only: exact 24-hour HH:MM. A nonexistent DST time is skipped; in a repeated hour only the first occurrence is eligible. Use separate timers for multiple daily times.

- Example 1: `08:30`.
- Example 2: `18:00`.

<a id="timers.timezone"></a>
### IANA time zone — `timers.timezone`

**Initial value:** `America/Bogota`.

For calendar only: an explicit known IANA zone or UTC. Local is rejected. The desktop timezone is not substituted implicitly.

- Example 1: `UTC`.
- Example 2: `Europe/Madrid`.

<a id="timers.weekdays"></a>
### Weekdays — `timers.weekdays`

**Initial value:** `[]`.

For calendar: unique codes mon tue wed thu fri sat sun, one per line in the UI. Empty means daily. Unknown or duplicate codes fail validation.

- Example 1: `["mon","tue","wed","thu","fri"]`.
- Example 2: `["sat","sun"]`.

<a id="timers.missed"></a>
### When runs are overdue — `timers.missed`

**Initial value:** `coalesce`.

coalesce emits one event for accumulated missed times; skip discards times over 5 seconds late for intervals or 60 seconds for calendars. Neither wakes a suspended computer.

- Example 1: `coalesce`.
- Example 2: `skip`.

<a id="group-monitors"></a>
## monitors

<a id="monitors.metric"></a>
### Metric — `monitors.metric`

**Initial value:** `disk`.

Choose cpu, memory, disk, battery, service, process, file_exists, file_age, file_size, connectivity, temperature or journal. CPU/memory use percent used; disk/battery use percent available. Presence/connectivity reads 0 or 100. Age is seconds, size bytes and temperature Celsius. Adjust both thresholds when changing metric.

- Example 1: `cpu`.
- Example 2: `file_age`.

<a id="monitors.path"></a>
### Disk path — `monitors.path`

**Initial value:** `/`.

The label changes with the metric. Disk: absolute filesystem path. Process: absolute canonical executable path, current user only. File metrics: absolute canonical regular-file path, no symlink components; metadata only. Temperature: explicit thermal/hwmon sensor. Missing/unreadable resources may be unavailable; file_exists reports 0 for a missing file.

- Example 1: `/home (disk)`.
- Example 2: `/sys/class/thermal/thermal_zone0/temp (temperature)`.

<a id="monitors.unit"></a>
### User unit — `monitors.unit`

**Initial value:** `""`.

Required for service/journal: exact user .service unit. Not a system unit or a wildcard. Service reports 100 when active and 0 when inactive. Journal reads only the permitted user unit and does not include MESSAGE content in its events.

- Example 1: `backup.service`.
- Example 2: `worker.service`.

<a id="monitors.destination"></a>
### HTTPS destination to check with HEAD — `monitors.destination`

**Initial value:** `config.destinations[0]?.id`.

The selector initially chooses the first registered destination, if any. For connectivity: a registered HTTPS destination. Uses HEAD with its credentials/headers, a three-second deadline and no redirects. HTTP 2xx gives 100, transport/HTTP failure 0; missing credentials or network-policy rejection produce unavailable. Changing the destination requires reviewing the monitor grant.

- Example 1: `status`.
- Example 2: `health-endpoint`.

<a id="monitors.priority"></a>
### Maximum journal priority (0 emergency … 7 debug) — `monitors.priority`

**Initial value:** `3`.

Journal only: integer 0–7, includes this priority and more urgent records. 0 emergency, 3 error, 4 warning, 6 information, 7 debug. Initial sampling starts now; it does not import earlier history.

- Example 1: `3`.
- Example 2: `4`.

<a id="monitors.threshold"></a>
### Alert threshold — `monitors.threshold`

**Initial value:** `10`.

Not journal. Alerts when falling for disk/battery/service/process/file_exists/connectivity, rising for other value metrics. Percent range 0–100; temperature −273.15–1000; age 0–31536000; size 0–10^15. An invalid recovery direction rejects configuration. Comparisons are strict: equality does not start an alert. See monitor-recipes.md for two configurations per metric.

- Example 1: `10 (disk; recovery 15)`.
- Example 2: `90 (cpu; recovery 75)`.

<a id="monitors.recovery"></a>
### Recovery threshold — `monitors.recovery`

**Initial value:** `15`.

Same units/ranges as threshold. Must be strictly above the alert threshold for falling metrics and strictly below for rising metrics. Returning across it emits recovered after the recovery duration. Equality does not recover; a value must cross the threshold strictly.

- Example 1: `15 (disk; threshold 10)`.
- Example 2: `75 (cpu; threshold 90)`.

<a id="monitors.duration_seconds"></a>
### Minimum condition duration (seconds) — `monitors.duration_seconds`

**Initial value:** `60`.

Not journal: integer 0–86400 seconds of observed alert condition before confirming. Zero allows confirmation on an eligible sample. A prolonged sampling gap resets confirmation; it is not continuous real-time observation.

- Example 1: `0`.
- Example 2: `120`.

<a id="monitors.recovery_duration_seconds"></a>
### Minimum recovery duration (seconds) — `monitors.recovery_duration_seconds`

**Initial value:** `0`.

Not journal: integer 0–86400 seconds of observed recovery before emitting recovered. Zero removes the additional confirmation delay, not the sampling interval.

- Example 1: `0`.
- Example 2: `30`.

<a id="monitors.interval_seconds"></a>
### Sampling interval (seconds) — `monitors.interval_seconds`

**Initial value:** `10`.

Integer 5–3600 seconds for every metric, including journal. Sampling is sequential and slow checks may delay later ones. The engine does not wake the machine to sample.

- Example 1: `5`.
- Example 2: `60`.

<a id="monitors.cooldown_seconds"></a>
### Minimum time between alerts (seconds) — `monitors.cooldown_seconds`

**Initial value:** `300`.

Not journal: minimum five seconds between alerts. It limits alert frequency; it does not change sample frequency or guarantee repeated reminders while a condition persists.

- Example 1: `60`.
- Example 2: `600`.

<a id="monitors.reset"></a>
### Reset cursor — `monitors.reset`

Journal monitors only. Confirmation discards the saved cursor; the next authorized sample starts from then, without recovering older records. Does not grant permissions and records an audit event.

- Example 1: `Reset after journal rotation removed the cursor`.
- Example 2: `Start a new observation period intentionally`.

<a id="group-actions"></a>
## actions

<a id="actions.kind"></a>
### Action — `actions.kind`

**Initial value:** `notify`.

Select notify, http, service, command, omarchy, script, adapter or system-service. Only applicable fields are saved. A flow must reference the action and grant its exact capability before execution.

- Example 1: `notify`.
- Example 2: `http`.

<a id="actions.title"></a>
### Title — `actions.title`

**Initial value:** `Omarchy Automations`.

Notification title, at most 200 bytes. Supports scalar event placeholders. The title is plain text; missing fields fail rendering.

- Example 1: `Deployment finished`.
- Example 2: `Backup: {{data.project}}`.

<a id="actions.body"></a>
### Message — `actions.body`

**Initial value:** `{{data.message}}`.

Notification message (up to 4096 bytes) or HTTP body (up to 16 KiB on configuration validation). HTTP labels this field Request body and requires a JSON template for json/form; raw uses text. Missing or nonscalar substituted fields may fail rendering; simulation previews the resolved value. Rendering also limits each textual template to 4096 bytes and expanded text to 16384 bytes; a configuration accepted by the size check can still fail rendering.

- Example 1: `Build: {{data.message}}`.
- Example 2: `{"state":"{{type}}","value":"{{data.value}}"}`.

<a id="actions.destination"></a>
### Registered destination — `actions.destination`

**Initial value:** `config.destinations[0]?.id`.

For http: select an existing destination, initially the first available. Its URL, headers and credential reference are included in the reviewed capability; changing them requires renewed review.

- Example 1: `status`.
- Example 2: `deploy-receiver`.

<a id="actions.unit"></a>
### User unit — `actions.unit`

**Initial value:** `""`.

For service: exact user .service. For system-service the label is System unit and requires optional broker policy and Polkit authorization. No patterns or arbitrary systemctl arguments. A policy check neither runs the service nor grants a flow.

- Example 1: `backup.service`.
- Example 2: `worker.service`.

<a id="actions.operation"></a>
### Operation — `actions.operation`

**Initial value:** `status`.

User/system services offer status, start, stop and restart. Omarchy offers theme.current, nightlight.status, nightlight.toggle and system.lock; switching to Omarchy selects the first valid operation. Status is a check, not a service mutation. Lock and toggle have real desktop effects. For user-service status, inactive is a failed action: it checks is-active rather than returning a detailed status report.

- Example 1: `restart (service)`.
- Example 2: `nightlight.status (omarchy)`.

<a id="actions.command_profile"></a>
### Command profile — `actions.command_profile`

**Initial value:** `fixed`.

For command: fixed runs a registered executable with fixed arguments; file-exists tests a direct-child regular file in an approved mount; make-directory creates a direct child with mode 0700 and requires write access. Named profiles do not accept executable/argument overrides.

- Example 1: `file-exists`.
- Example 2: `make-directory`.

<a id="actions.command_path"></a>
### Profile path — `actions.command_path`

**Initial value:** `""`.

For file-exists/make-directory: absolute canonical path naming a direct child of a prepared mount, without templates. The parent must be an authorized directory target. make-directory fails if the child already exists.

- Example 1: `/work/data/report.json`.
- Example 2: `/work/data/new-backup`.

<a id="actions.executable"></a>
### Local executable — `actions.executable`

**Initial value:** `""`.

For fixed commands: absolute canonical executable under /usr/bin/. It must be available inside the sandbox; no home access or IP network by default. A command error is not automatically retried.

- Example 1: `/usr/bin/true`.
- Example 2: `/usr/bin/test`.

<a id="actions.args"></a>
### Fixed arguments — `actions.args`

**Initial value:** `[]`.

For fixed commands: at most 32 arguments, one per UI line; each at most 4096 bytes, without NUL or {{ templates. No shell expansion. Review the executable too: an interpreter can deliberately treat an argument as code.

- Example 1: `["-f","/work/data/report.json"]`.
- Example 2: `["-d","/work/data"]`.

<a id="actions.adapter"></a>
### Registered adapter — `actions.adapter`

**Initial value:** `config.adapters[0]?.id`.

For adapter: select a registered adapter, inspect its code/revision, and explicitly choose that revision. Changing code or limits invalidates the old reference. The action uses manifest limits rather than a separate timeout.

- Example 1: `normalize`.
- Example 2: `summarize`.

<a id="actions.script"></a>
### Registered script — `actions.script`

**Initial value:** `config.scripts[0]?.id`.

For script: select a registered script and explicitly select its exact revision. Supply every parameter once as either a literal or an event binding. The original source file is not reopened during execution.

- Example 1: `check-project`.
- Example 2: `prepare-report`.

<a id="actions.timeout_seconds"></a>
### Timeout (seconds) — `actions.timeout_seconds`

**Initial value:** `30`.

For command/script: integer 1–300 seconds. The isolated process group is terminated on timeout; an effect already performed is not undone. Adapter timeouts are configured on the adapter itself.

- Example 1: `5`.
- Example 2: `120`.

<a id="actions.script_revision"></a>
### Use this revision and reset parameters — `actions.script_revision`

**Initial value:** `""`.

Inspect the available script code/hash before selecting. This pins that exact revision and resets each value: false for boolean, minimum for integer, first choice or empty text for string. A changed script requires selecting again; it is never silently substituted. Example hashes are obtained from preparation, not invented.

- Example 1: `check-project: select prepared revision`.
- Example 2: `prepare-report: select updated revision`.

<a id="actions.adapter_revision"></a>
### Use this revision — `actions.adapter_revision`

**Initial value:** `""`.

Inspect adapter code and its manifest limits, then pin the prepared revision. Preparing or selecting is not execution or a grant. Any code/manifest change requires a new exact selection and reviewed capability.

- Example 1: `normalize: select prepared revision`.
- Example 2: `summarize: select updated revision`.

<a id="actions.working_directory"></a>
### Working directory — `actions.working_directory`

**Initial value:** `/tmp`.

Choose private /tmp or one of this action’s approved mount targets, not an arbitrary subdirectory. Removing the chosen target resets the UI to /tmp.

- Example 1: `/tmp`.
- Example 2: `/work/reports`.

<a id="group-scripts"></a>
## scripts

<a id="scripts.path"></a>
### Local script file — `scripts.path`

**Initial value:** `""`.

Absolute path to a regular local UTF-8 file, 1–32768 bytes, without NUL or symlink components. Prepare revision reads and copies the code for inspection; it neither executes nor authorizes it. Edit the source and prepare again to replace code; exports include the prepared code.

- Example 1: `/home/alex/automations/check.py`.
- Example 2: `/home/alex/automations/report.py`.

<a id="scripts.interpreter"></a>
### Interpreter — `scripts.interpreter`

**Initial value:** `bash`.

bash or python3. Bash uses --noprofile --norc; Python uses -I -S. Code and interpreter are part of the revision. Parameters are separate argv values, not interpolated shell code.

- Example 1: `bash`.
- Example 2: `python3`.

<a id="scripts.parameters"></a>
### Script parameters — `scripts.parameters`

**Initial value:** `[]`.

Ordered list of at most 16 required typed parameters. Names are unique resource-style IDs; accepted types are string, integer and boolean. The declared order becomes argv order. No extra or missing values are accepted.

- Example 1: `[{"name":"message","type":"string","max_length":256}]`.
- Example 2: `[{"name":"count","type":"integer","minimum":1,"maximum":10}]`.

<a id="group-adapters"></a>
## adapters

<a id="adapters.path"></a>
### Local adapter file — `adapters.path`

**Initial value:** `""`.

Absolute path to a regular local UTF-8 file, 1–32768 bytes, without NUL or symlink components. Prepare revision reads and copies the code for inspection; it neither executes nor authorizes it. Edit the source and prepare again to replace code; exports include the prepared code.

- Example 1: `/home/alex/automations/check.py`.
- Example 2: `/home/alex/automations/report.py`.

<a id="adapters.runtime"></a>
### Interpreter — `adapters.runtime`

**Initial value:** `python3`.

Only python3 is supported. The process uses isolated Python (-I -S) and the version-1 JSON request/response protocol. This is not a choice of arbitrary runtimes; both examples use the one supported runtime for different transforms.

- Example 1: `python3: normalize fields`.
- Example 2: `python3: calculate a summary`.

<a id="adapters.input_limit"></a>
### Input limit (bytes) — `adapters.input_limit`

**Initial value:** `262144`.

Integer 1–262144 bytes for the complete encoded protocol request including its newline, not just event data. Oversize input fails before launching the adapter. Choose enough space for the envelope.

- Example 1: `4096`.
- Example 2: `65536`.

<a id="adapters.output_limit"></a>
### Output limit (bytes) — `adapters.output_limit`

**Initial value:** `4096`.

Integer 1–65536 bytes for the complete JSON response. Oversize, extra output, wrong invocation ID or invalid data rejects the result. Adapter data is available to following steps under data.adapter.

- Example 1: `2048`.
- Example 2: `16384`.

<a id="adapters.timeout_seconds"></a>
### Timeout (seconds) — `adapters.timeout_seconds`

**Initial value:** `5`.

Integer 1–30 seconds. The isolated transform must finish within it. No network or authorized host directories are offered to adapters. Simulation reports the dependency but does not execute the adapter.

- Example 1: `2`.
- Example 2: `10`.

<a id="group-parameters"></a>
## parameters

<a id="parameters.name"></a>
### Parameter name — `parameters.name`

**Initial value:** `parameterN`.

A newly added parameter uses parameter plus its position. Replace with a unique identifier (same syntax as resource IDs). Renaming changes the contract and requires preparing/reviewing a new revision.

- Example 1: `message`.
- Example 2: `count`.

<a id="parameters.type"></a>
### Value type — `parameters.type`

**Initial value:** `string`.

Choose string, integer or boolean. Changing type resets its bounds: string starts max_length 256, integer starts minimum 0/maximum 100, boolean has no bounds.

- Example 1: `integer`.
- Example 2: `boolean`.

<a id="parameters.max_length"></a>
### Maximum length (bytes) — `parameters.max_length`

**Initial value:** `256`.

String only: integer 1–4096 UTF-8 bytes, not characters. Values exceeding it are rejected before execution.

- Example 1: `64`.
- Example 2: `1024`.

<a id="parameters.pattern"></a>
### RE2 pattern (optional) — `parameters.pattern`

**Initial value:** `""`.

String only: optional RE2 pattern up to 256 bytes, matched against the entire value. Empty means no pattern restriction. Invalid patterns fail preparation.

- Example 1: `[a-z][a-z0-9-]*`.
- Example 2: `[0-9]{4}`.

<a id="parameters.choices"></a>
### Allowed choices (one per line) — `parameters.choices`

**Initial value:** `[]`.

String only: at most 32 unique allowed values, one per line. Empty imposes no enumeration. Every listed choice must itself satisfy length/pattern limits.

- Example 1: `["staging","production"]`.
- Example 2: `["small","large"]`.

<a id="parameters.minimum"></a>
### Minimum — `parameters.minimum`

**Initial value:** `0`.

Integer only: explicit inclusive lower bound, no larger than maximum. Bounds lie within ±9007199254740991. Text and fractional values do not become integers automatically.

- Example 1: `1`.
- Example 2: `-10`.

<a id="parameters.maximum"></a>
### Maximum — `parameters.maximum`

**Initial value:** `100`.

Integer only: explicit inclusive upper bound, no smaller than minimum and within the exact JSON integer range. Limits are checked again after resolving an event binding.

- Example 1: `10`.
- Example 2: `1000`.

<a id="group-arguments"></a>
## arguments

<a id="arguments.mode"></a>
### Literal value — `arguments.mode`

**Initial value:** `script_values`.

Per parameter: choose Literal value or Event field. Exactly one source is required. Switching to Event field suggests data.PARAMETER; switching back resets a type-appropriate literal. Actual values must pass the selected script contract.

- Example 1: `script_values: {"count":3}`.
- Example 2: `script_bindings: {"count":"data.count"}`.

<a id="arguments.value"></a>
### Value — `arguments.value`

**Initial value:** `"" / minimum / false / data.PARAMETER`.

Literals preserve string/integer/boolean types; boolean uses a toggle. Event bindings accept source, type or data.FIELD with at most 128 characters/eight dots. Missing fields, wrong types and out-of-range values fail before launching the script.

- Example 1: `data.message`.
- Example 2: `true (boolean literal)`.

<a id="group-directories"></a>
## directories

<a id="directories.source"></a>
### Host directory — `directories.source`

**Initial value:** `""`.

For command/script only: existing absolute canonical directory, without symlink components. Root and /proc, /sys, /dev, /run trees are rejected. Preparation captures device/inode identity; a replaced directory requires re-preparation and review.

- Example 1: `/home/alex/reports`.
- Example 2: `/home/alex/project-data`.

<a id="directories.target"></a>
### Sandbox mount target — `directories.target`

**Initial value:** `/work/data`.

Target is /work/ID, using lowercase resource-ID syntax. At most eight unique targets per action. Preparing an existing target replaces its draft entry; the source tree is exposed at this sandbox path.

- Example 1: `/work/reports`.
- Example 2: `/work/project`.

<a id="directories.access"></a>
### Read only — `directories.access`

**Initial value:** `ro`.

Choose read-only (ro) or read/write (rw). Permission covers the whole tree. Write allows modifying/deleting host files; sockets in an exposed tree may communicate with host services. No permissions are granted until activation.

- Example 1: `ro`.
- Example 2: `rw`.

<a id="directories.prepare"></a>
### Prepare and add directory — `directories.prepare`

Verifies the existing directory identity and adds/replaces its target in the local action draft, up to eight targets. Does not grant access until the action revision is approved.

- Example 1: `Prepare a read-only report tree`.
- Example 2: `Prepare an output tree with explicit write access`.

<a id="group-credentials"></a>
## credentials

<a id="credentials.type"></a>
### Credential type — `credentials.type`

**Initial value:** `generic`.

Generic saves an HMAC/bearer secret. Google import saves an existing refresh-token bundle without contacting Google. Google browser authorization starts a ten-minute consent session; it does not grant flows. Choose the mode matching the integration.

- Example 1: `generic: HMAC signing secret`.
- Example 2: `Google OAuth2: browser authorization`.

<a id="credentials.id"></a>
### Identifier — `credentials.id`

**Initial value:** `""`.

Local credential reference using resource-ID syntax. Reusing an ID rotates its value in the same backend. A different backend is rejected until the existing reference is deleted; deletion affects all consumers of the reference.

- Example 1: `deploy-token`.
- Example 2: `google-calendar`.

<a id="credentials.value"></a>
### New value (16–8192 bytes) — `credentials.value`

**Initial value:** `""`.

Generic only: 16–8192 bytes without CR, LF or NUL. The masked value is cleared after Save/Cancel. Examples describe sources of private values; do not use published example text as a production credential. Store a signing secret for inbound HMAC/Slack or the token expected by a bearer receiver.

- Example 1: `private signing secret from your webhook sender`.
- Example 2: `private API token from your HTTPS receiver`.

<a id="credentials.backend"></a>
### Credential store — `credentials.backend`

**Initial value:** `keyring`.

Desktop keyring uses Secret Service and may need an unlocked session. Private file explicitly stores an unencrypted local value with restrictive permissions. No automatic fallback. Configuration exports contain references, not these stored values.

- Example 1: `keyring`.
- Example 2: `file`.

<a id="credentials.client_id"></a>
### Client ID — `credentials.client_id`

**Initial value:** `""`.

Google modes: your own Desktop OAuth client ID, not your Google email or password. Required printable ASCII without spaces/control characters, at most 8192 bytes per value; the entire stored credential bundle also has an 8192-byte limit. Examples are nonfunctional placeholders.

- Example 1: `YOUR_DESKTOP_CLIENT_A.apps.googleusercontent.com`.
- Example 2: `YOUR_DESKTOP_CLIENT_B.apps.googleusercontent.com`.

<a id="credentials.client_secret"></a>
### Client secret — `credentials.client_secret`

**Initial value:** `""`.

Google modes: secret belonging to the selected Desktop client. Same printable-ASCII/value and bundle limits as client ID. Masked and cleared after Save/Cancel. Supply locally; examples are placeholders, not usable secrets.

- Example 1: `CLIENT_SECRET_FROM_DESKTOP_CLIENT_A`.
- Example 2: `CLIENT_SECRET_FROM_DESKTOP_CLIENT_B`.

<a id="credentials.refresh_token"></a>
### Refresh token — `credentials.refresh_token`

**Initial value:** `""`.

Google import only: an existing refresh token issued to the matching client, not a short-lived access token. Saving does not validate granted scopes with Google. Same ASCII and size limits apply. Examples are placeholders; actual values belong in the local masked field.

- Example 1: `REFRESH_TOKEN_FOR_ACCOUNT_A`.
- Example 2: `REFRESH_TOKEN_FOR_ACCOUNT_B`.

<a id="credentials.scopes"></a>
### OAuth scopes — `credentials.scopes`

**Initial value:** `""`.

Browser authorization only: 1–16 distinct HTTPS scope URLs under www.googleapis.com/auth/, at most 256 bytes each, separated by whitespace in the UI. No query, fragment, userinfo or port. The calendar.events placeholder is not a selected default. Request only scopes needed for your destination API; Google must grant all requested scopes. The example scope names are listed in the [official Google Calendar scope reference](https://developers.google.com/workspace/calendar/api/auth). A valid scope does not add GET support: outbound actions still offer POST/PUT/PATCH.

- Example 1: `https://www.googleapis.com/auth/calendar.events`.
- Example 2: `https://www.googleapis.com/auth/calendar.readonly`.

<a id="group-storage"></a>
## storage

<a id="storage.max_events"></a>
### Maximum events — `storage.max_events`

**Initial value:** `10000`.

Integer 100–100000. Save limits applies immediately, separately from draft activation. Reaching a quota rejects new events rather than deleting unfinished or uncertain work. A larger limit is not a substitute for resolving a stuck queue.

- Example 1: `5000`.
- Example 2: `20000`.

<a id="storage.max_payload_bytes"></a>
### Event data (MiB) — `storage.max_payload_bytes`

**Initial value:** `64 MiB`.

UI accepts integer 1–128 MiB; API stores bytes (MiB × 1048576). Counts event payloads plus private execution contexts, not total SQLite/WAL disk usage. Saving limits does not export or upload event data.

- Example 1: `16 MiB = 16777216 bytes`.
- Example 2: `128 MiB = 134217728 bytes`.

<a id="storage.retention_days"></a>
### Keep history (days) — `storage.retention_days`

**Initial value:** `7`.

Integer 1–90 days for terminal history. Pending/uncertain work is retained. Shortening retention affects later cleanup; keep dedup_days at least as large.

- Example 1: `3`.
- Example 2: `30`.

<a id="storage.dedup_days"></a>
### Deduplication window (days) — `storage.dedup_days`

**Initial value:** `30`.

Integer between retention_days and 365. Retains delivery identities independently of normal payload retention. It does not make an external HTTP receiver idempotent or prove freshness of an old provider signature outside the retained window.

- Example 1: `30`.
- Example 2: `90`.

<a id="group-panel"></a>
## panel

<a id="panel.language"></a>
### Language — `panel.language`

**Initial value:** `en`.

English or Español, stored per profile immediately. Does not change resource IDs or user-written messages. Requires engine connection.

- Example 1: `English for an English desktop`.
- Example 2: `Español for a Spanish desktop`.

<a id="panel.save"></a>
### Save draft — `panel.save`

Validates and saves the edited configuration. It does not grant capabilities or change the active revision. The dot indicates local unsaved edits; fix validation errors before review.

- Example 1: `Save a new notification flow`.
- Example 2: `Save changed recovery thresholds`.

<a id="panel.activate"></a>
### Review and activate — `panel.activate`

Saves the draft, then shows exact capabilities and their revision hash. OK activates that reviewed revision; Cancel leaves it unactivated. Resource changes require another review.

- Example 1: `Approve one notification capability`.
- Example 2: `Review a monitor plus an outbound action`.

<a id="panel.pause"></a>
### Pause — `panel.pause`

Toggles Pause/Resume with admission retain. Pause stops subsequent dispatch while retaining incoming events; it does not undo effects already started. Resume lets queued work continue.

- Example 1: `Pause while inspecting pending work`.
- Example 2: `Resume after restoring a credential`.

<a id="panel.stop"></a>
### Stop executions — `panel.stop`

Opens confirmation. Accept cancels pending work and attempts to stop active processes; effects already sent cannot be undone and results may be uncertain. It does not reject future incoming events.

- Example 1: `Cancel queued tests after a mistake`.
- Example 2: `Stop an active long-running script`.

<a id="panel.new"></a>
### New — `panel.new`

Opens a new resource editor in the current section. Save in that editor stages the resource locally; Save draft then validates the full configuration. Cancel abandons that editor submission.

- Example 1: `Create a destination in Connections`.
- Example 2: `Create an interval in Scheduling`.

<a id="panel.edit"></a>
### Edit — `panel.edit`

Opens the selected resource for editing. The existing identifier is immutable in the form; references and exact revisions still require validation. Saving does not activate.

- Example 1: `Change a notification message`.
- Example 2: `Change a timer timezone`.

<a id="panel.remove"></a>
### Remove — `panel.remove`

Removes the resource from local draft edits, without a confirmation dialog. Save validates remaining references; activation is separate. Removing a referenced action requires updating its flows first.

- Example 1: `Remove an unused destination`.
- Example 2: `Remove an obsolete disabled timer`.

<a id="panel.refresh"></a>
### Refresh — `panel.refresh`

Refreshes section status, such as monitor/timer state, queue or security metadata. It does not activate a draft or itself test an external service.

- Example 1: `Refresh after a delivery completes`.
- Example 2: `Refresh credential metadata after consent`.

<a id="panel.import"></a>
### Import — `panel.import`

Reads an absolute regular .json file up to 1 MiB and replaces the saved draft with sources/flows disabled. Does not merge with the draft or replace active revision. Export edits you wish to keep first.

- Example 1: `Import the notification example`.
- Example 2: `Import a previously exported configuration`.

<a id="panel.export"></a>
### Export — `panel.export`

Requires a saved draft. Writes configuration to a new absolute .json path; refuses overwrite. Includes code, ordinary headers and body text, but only references to stored credentials. No automatic upload.

- Example 1: `Export before importing another setup`.
- Example 2: `Export a configuration for local backup`.

<a id="group-files"></a>
## files

<a id="files.path"></a>
### File path — `files.path`

**Initial value:** `""`.

Used by import, configuration export and diagnosis export. Import requires an existing regular file; export requires a new filename. A final symlink is rejected. Example parent directories must exist.

- Example 1: `/home/alex/automations-backup.json`.
- Example 2: `/home/alex/diagnostics-01.json`.

<a id="group-history"></a>
## history

<a id="history.queue"></a>
### Pending / uncertain queue only — `history.queue`

**Initial value:** `false`.

Off shows the latest 100 executions. On shows pending, running and uncertain work with pages of 100. This is a live queue: counts and page contents may change as work progresses.

- Example 1: `Enable to inspect a stuck delivery`.
- Example 2: `Disable to find a terminal failed delivery`.

<a id="history.pages"></a>
### Previous — `history.pages`

Previous/Next moves through queue pages by 100, when available; it does not change work state. Refresh can update the page contents.

- Example 1: `Next to inspect jobs after the first 100`.
- Example 2: `Previous to return to the first page`.

<a id="history.details"></a>
### Details — `history.details`

Opens step states and HTTP attempt details without payload bodies or credentials. A completed step can belong to a pending execution. Inspect uncertain external outcomes before retry decisions.

- Example 1: `Read the HTTP status of a failed send`.
- Example 2: `Find which step lost its permission`.

<a id="history.retry"></a>
### Retry HTTP — `history.retry`

Shown for failed/cancelled executions; the engine only accepts eligible HTTP steps. Confirmation reuses the original body/idempotency key. Receiver-side idempotency is required to avoid duplicates. Non-HTTP actions are not retried by this button.

- Example 1: `Retry after repairing the receiver`.
- Example 2: `Retry after fixing rejected credentials`.

<a id="group-simulation"></a>
## simulation

<a id="simulation.source"></a>
### Source — `simulation.source`

**Initial value:** `local:demo`.

Exact event source. Simulation accepts registered entry/monitor/timer sources and local/hook sources. Run real test is available only for local: and hook:; it does not forge a signed webhook.

- Example 1: `local:demo`.
- Example 2: `monitor:disk-space`.

<a id="simulation.type"></a>
### Type — `simulation.type`

**Initial value:** `test`.

Event type used by conditions and templates. It does not select an entry authentication mode or a system action by itself.

- Example 1: `alert`.
- Example 2: `recovered`.

<a id="simulation.data"></a>
### Sample data (JSON) — `simulation.data`

**Initial value:** `{"message":"…","state":"failed"}`.

A valid JSON object, not a JSON array or unquoted text. Use data matching fields referenced by your conditions/actions; avoid credentials in sample payloads. Initial message follows the interface language.

- Example 1: `{"message":"Build finished","state":"ok"}`.
- Example 2: `{"metric":"disk","value":8,"state":"alert"}`.

<a id="simulation.preview"></a>
### Simulate without effects — `simulation.preview`

Saves a dirty draft first, then explains matching and rejected flows and resolves steps without executing actions, reading credentials or creating work. Adapters remain unresolved dependencies.

- Example 1: `Preview a notification body`.
- Example 2: `Verify a numeric condition does not match`.

<a id="simulation.real"></a>
### Run real test — `simulation.real`

Opens a separate confirmation and sends the sample local/hook event to the active revision. It can notify, execute or send HTTP under active grants. Cancel sends no event. It does not activate unsaved changes.

- Example 1: `Send a real local notification test`.
- Example 2: `Exercise an authorized hook flow`.

<a id="group-security"></a>
## security

<a id="security.create"></a>
### Create or rotate credential — `security.create`

Opens credential fields. Save applies the selected generic/import/browser operation immediately, independently of draft activation. Cancel clears sensitive inputs and submits nothing.

- Example 1: `Create a new inbound signing reference`.
- Example 2: `Rotate a bearer token under its existing ID`.

<a id="security.delete"></a>
### Delete — `security.delete`

Opens credential deletion confirmation. Accept removes the selected local value/reference, causing dependent authentication to fail until restored. It neither undoes completed requests nor revokes Google consent remotely.

- Example 1: `Delete an unused API credential`.
- Example 2: `Remove a Google connection locally`.

<a id="security.revoke"></a>
### Revoke — `security.revoke`

Immediately revokes the selected active capability, without a confirmation dialog. Prevents future authorized dispatch under that grant, not completed effects. Revoking a monitor/timer stops new events but does not revoke action grants for queued work.

- Example 1: `Revoke an outbound action grant`.
- Example 2: `Revoke a timer event capability`.

<a id="security.oauth_open"></a>
### Open consent in browser — `security.oauth_open`

Available while a Google authorization session awaits consent. Opens its provider URL in the browser; review the requested permissions there. The session expires after ten minutes. Do not share the URL/code as troubleshooting data.

- Example 1: `Open consent for a new connection`.
- Example 2: `Reopen the waiting session after panel reload`.

<a id="security.oauth_cancel"></a>
### Cancel authorization — `security.oauth_cancel`

Cancels a pending local consent/exchange session and closes its listener. A successfully stored connection is not revoked by cancelling. Starting again creates a new session; engine restart also requires starting over.

- Example 1: `Cancel after choosing the wrong client`.
- Example 2: `Cancel an authorization you no longer need`.

<a id="security.oauth_status"></a>
### OAuth2 status — `security.oauth_status`

Shows locally stored connection metadata, including renewal/reconnection state. Does not contact Google or prove the token remains valid remotely. Missing/rejected scopes or invalid_grant can require reconnecting.

- Example 1: `Inspect after a successful consent`.
- Example 2: `Inspect after an HTTP 401`.

<a id="security.diagnostics"></a>
### View diagnostics — `security.diagnostics`

Shows a report of versions, counters, pause/storage state and tool availability. Excludes configuration, URLs, paths, payloads and stored secret values. Tool presence alone does not establish that its service works.

- Example 1: `Check component versions after updating`.
- Example 2: `Inspect queue/storage counts during an incident`.

<a id="security.export_diagnostics"></a>
### Export diagnostics — `security.export_diagnostics`

Writes that bounded diagnostic report to a new absolute .json path with mode 0600, without overwriting or uploading. A failed write can leave an incomplete file; choose another new path after inspecting the error.

- Example 1: `Save a report before upgrading`.
- Example 2: `Save a report after a failed delivery`.

<a id="security.save_limits"></a>
### Save limits — `security.save_limits`

Applies all four storage-policy values immediately, separately from the draft/review workflow. Invalid combinations are rejected. It does not erase pending/uncertain work to meet the new quota.

- Example 1: `Increase capacity after inspecting queue growth`.
- Example 2: `Shorten terminal history and retain longer deduplication`.

<a id="group-code"></a>
## code

<a id="code.prepare"></a>
### Prepare revision — `code.prepare`

Reads the selected local script/adapter and prepares code plus exact revision. Review before Save. Changing preparation fields invalidates it and requires preparing again. Never runs the code or grants capabilities.

- Example 1: `Prepare a Python script with typed parameters`.
- Example 2: `Prepare a changed adapter manifest and code`.

<a id="group-administration"></a>
## administration

<a id="administration.check"></a>
### Check administrative permissions — `administration.check`

For system-service: checks exact unit/operation against the optional broker without executing systemctl or granting a flow. Authorized does not prove the unit exists or guarantee future permission; execution rechecks policy/Polkit.

- Example 1: `Check status authorization for backup.service`.
- Example 2: `Check restart authorization for worker.service`.
