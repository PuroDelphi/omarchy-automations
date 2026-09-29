# Product name and compatibility

[Español](../es/product-name.md) · [Documentation](README.md)

The public name is **Omarchy Automations**. It appears in the manifest, main title,
notifications, OAuth authorization, new credential-store labels, documentation
and service descriptions. The bar uses **Automations** to save space. English is
the initial language; users can select Spanish.

The following identifiers remain unchanged so existing installations can update
without duplicate plugins, lost credentials or broken integrations:

| Contract | Retained identifier |
|---|---|
| Plugin and shell preferences | `quatrro.automations` |
| Binaries and user service | `quatrrod`, `quatrroctl`, `quatrrod.service` |
| Broker, socket and Polkit actions | `quatrro-broker` and `org.quatrro.automations.*` names |
| XDG directories and Secret Service attribute | `quatrro` |
| Development profile | `QUATRRO_PROFILE` |
| Generic webhook signature | `X-Quatrro-*` headers |
| Go module | `quatrro.local/automations` |

The rename does not change the SQLite schema, protocols, approved permissions,
active revisions or bar selection. There is no need to rename data files or enter
secrets again. Existing credential labels and user-defined flow titles are kept;
new defaults use the updated name.

Compatibility error translations recognize both the old and new wording to
support components from different generations. Installation still requires
compatible engine, CLI and panel versions. The local work directory may still be
named `quatrro-automations`; the remote repository is `omarchy-automations`.
