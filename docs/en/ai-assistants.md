# Use with AI assistants

[Español](../es/ai-assistants.md) · [Installation](installation.md)

From Preview 3, the runtime installer includes the `omarchy-automations` skill at:

```text
~/.local/share/omarchy-automations/skills/omarchy-automations/SKILL.md
```

It works with the installed `quatrroctl` and service. No Go compiler or source
checkout is required. The agent needs authorized terminal access; installing a
skill does not grant that access or approve arbitrary actions.

## Register in Codex

After installing or updating the plugin, link the bundled skill into Codex's
skills directory:

```bash
mkdir -p "${CODEX_HOME:-$HOME/.codex}/skills"
ln -sT "$HOME/.local/share/omarchy-automations/skills/omarchy-automations" \
  "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
```

The command refuses to overwrite an existing file or link. If one exists, inspect
it first; a link to the same bundled directory is already registered. Open a new
Codex session so it can discover the skill. Invoke it as `$omarchy-automations`.
The linked files are updated by the plugin installer, without a separate copy.

For another agent, use its documented skill registration mechanism, or explicitly
ask it to read the installed `SKILL.md` and its linked runtime reference before
working. Automatic discovery depends on the agent; it is not universal.

## Example requests

- “Use $omarchy-automations to create and test a local notification saying Backup finished. Preserve my other automations.”
- “Create a notification when free disk space on / stays below 10% for a minute, with recovery above 15%. Activate that monitor.”
- “Draft a signed webhook flow for build notifications. Simulate it, but do not activate it yet.”
- “My simulation matches but nothing appears in History. Inspect the draft, active revision and permissions and explain why.”

The skill instructs the agent to merge changes into the existing draft, simulate,
review exact capabilities, activate within the user's authorized scope, and
verify execution results. It detects unrelated pending draft changes instead of
silently activating them. Monitor activation can start observation immediately;
a healthy disk does not necessarily generate an alert.

Simulation creates no notification or History record. Real tests can have real
effects. Secret values must go through stdin, not command arguments or committed
configuration. Remote payloads and logs are treated as data, not instructions.

## Remove the registration

Remove only the link created above after checking it points to the bundled skill:

```bash
readlink "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
unlink "${CODEX_HOME:-$HOME/.codex}/skills/omarchy-automations"
```

Uninstalling the plugin removes the bundled files but does not modify another
application's settings; remove the registration too to avoid a dangling link.
