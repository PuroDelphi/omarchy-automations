# Development release notes — 0.1.0-dev

[Español](../es/release-notes.md) · [Documentation](README.md)

This is a published development preview for Linux amd64, not the stable 1.0 release.
Preview 7 adds visual authoring of all six Omarchy hooks, a native event connection
manager, suggested condition fields and translated operators, inline action editing,
and a searchable library of 38 English/Spanish templates. Templates use fresh IDs
and enter the draft disabled. Hook management preserves modified and unrelated
files and rejects symlinks. Explicit Commons.Color qualification fixes a Qt type
name collision affecting the panel palette.

The reproducible runtime archive includes these changes and is pinned by SHA256
in the checkout installer. The release tag identifies the exact source commit.
Preview 6's release-pin helper and existing compatibility contracts are retained.

## Included

- Authenticated inbound webhooks, HTTPS outbound delivery, conditions and reviewed flow permissions.
- Notifications, user services, isolated commands/scripts, local adapters, monitors and schedules.
- Optional explicit private-network exceptions, Google OAuth implementation and separately installed administrative broker.
- Native Omarchy controls/icon, English default, persistent Spanish, responsive layout and accessible labels.
- Paired guides, option reference, complete examples, screenshots, installer and recovery procedures.

## Compatibility and upgrade

Engine/CLI version 0.1.0-dev, local protocol 2, operation contract 1, UI contract 1;
SQLite schema 5. The optional broker retains protocol 1. Technical identifiers
quatrroctl, quatrrod.service and quatrro.automations remain compatible. Install
matching components together and stop the engine before updating. The installer
verifies owned files and records backups; restored databases start paused with
grants removed. See the [user guide](user-guide.md).

## Verification and remaining gates

Current Linux amd64 binaries reproduced identically with Go 1.26.8 in separate
source trees and initially empty build caches. Archive determinism, file hashes
and extracted install/update/remove passed. Functional, security, native UI and
bilingual documentation reviews are recorded in the roadmap and validation log.

Real Google-account consent/refresh/keyring acceptance and physical suspend/full
logout testing remain pending. Local provider fixtures and cgroup freeze/target
restart do not substitute for these tests. The project uses the [MIT license](../../LICENSE);
[dependency notices](../third-party-notices.md) retain their original terms.
These open gates prevent declaring stable 1.0 complete.
