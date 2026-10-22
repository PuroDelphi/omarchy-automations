.pragma library

// Templates are drafts: no grants, credentials or automatic activation.
var recipes = [
  {
    "id": "theme-set",
    "en": "Theme changed",
    "es": "Tema cambiado",
    "category": "Desktop",
    "source": "hook:theme-set",
    "body_en": "Omarchy event received: {{data.theme}}",
    "body_es": "Evento de Omarchy recibido: {{data.theme}}"
  },
  {
    "id": "font-set",
    "en": "Font changed",
    "es": "Fuente cambiada",
    "category": "Desktop",
    "source": "hook:font-set",
    "body_en": "Omarchy event received: {{data.font}}",
    "body_es": "Evento de Omarchy recibido: {{data.font}}"
  },
  {
    "id": "battery-low",
    "en": "Low battery",
    "es": "Batería baja",
    "category": "Desktop",
    "source": "hook:battery-low",
    "body_en": "Omarchy event received: {{data.percentage}}",
    "body_es": "Evento de Omarchy recibido: {{data.percentage}}"
  },
  {
    "id": "post-boot",
    "en": "Desktop ready",
    "es": "Escritorio listo",
    "category": "Desktop",
    "source": "hook:post-boot",
    "body_en": "Omarchy event received.",
    "body_es": "Evento de Omarchy recibido."
  },
  {
    "id": "post-update",
    "en": "Update finished",
    "es": "Actualización terminada",
    "category": "Desktop",
    "source": "hook:post-update",
    "body_en": "Omarchy event received.",
    "body_es": "Evento de Omarchy recibido."
  },
  {
    "id": "pre-refresh-pacman",
    "en": "Package synchronization starting",
    "es": "Inicio de sincronización de paquetes",
    "category": "Desktop",
    "source": "hook:pre-refresh-pacman",
    "body_en": "Omarchy event received.",
    "body_es": "Evento de Omarchy recibido."
  },
  {
    "id": "theme-tokyo",
    "en": "Tokyo Night theme",
    "es": "Tema Tokyo Night",
    "category": "Desktop",
    "source": "hook:theme-set",
    "body_en": "Your selected theme is active.",
    "body_es": "Tu tema seleccionado está activo.",
    "conditions": [
      {
        "field": "data.theme",
        "op": "eq",
        "value": "tokyo-night"
      }
    ]
  },
  {
    "id": "theme-catppuccin",
    "en": "Catppuccin theme",
    "es": "Tema Catppuccin",
    "category": "Desktop",
    "source": "hook:theme-set",
    "body_en": "Your selected theme is active.",
    "body_es": "Tu tema seleccionado está activo.",
    "conditions": [
      {
        "field": "data.theme",
        "op": "eq",
        "value": "catppuccin"
      }
    ]
  },
  {
    "id": "theme-nord",
    "en": "Nord theme",
    "es": "Tema Nord",
    "category": "Desktop",
    "source": "hook:theme-set",
    "body_en": "Your selected theme is active.",
    "body_es": "Tu tema seleccionado está activo.",
    "conditions": [
      {
        "field": "data.theme",
        "op": "eq",
        "value": "nord"
      }
    ]
  },
  {
    "id": "theme-gruvbox",
    "en": "Gruvbox theme",
    "es": "Tema Gruvbox",
    "category": "Desktop",
    "source": "hook:theme-set",
    "body_en": "Your selected theme is active.",
    "body_es": "Tu tema seleccionado está activo.",
    "conditions": [
      {
        "field": "data.theme",
        "op": "eq",
        "value": "gruvbox"
      }
    ]
  },
  {
    "id": "eyes",
    "en": "Rest your eyes",
    "es": "Descansa la vista",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Rest your eyes.",
    "body_es": "Descansa la vista.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 1200,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "stretch",
    "en": "Stretch break",
    "es": "Pausa para estirar",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Stretch break.",
    "body_es": "Pausa para estirar.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 3600,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "water",
    "en": "Drink water",
    "es": "Bebe agua",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Drink water.",
    "body_es": "Bebe agua.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 5400,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "posture",
    "en": "Check your posture",
    "es": "Revisa tu postura",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Check your posture.",
    "body_es": "Revisa tu postura.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 2700,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "save",
    "en": "Save your work",
    "es": "Guarda tu trabajo",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Save your work.",
    "body_es": "Guarda tu trabajo.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 1800,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "focus",
    "en": "Focus checkpoint",
    "es": "Revisión de concentración",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Focus checkpoint.",
    "body_es": "Revisión de concentración.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 1500,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "walk",
    "en": "Take a short walk",
    "es": "Da un paseo corto",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Take a short walk.",
    "body_es": "Da un paseo corto.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 7200,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "backup",
    "en": "Review your backups",
    "es": "Revisa tus copias de seguridad",
    "category": "Reminders",
    "source": "timer:trigger",
    "body_en": "Review your backups.",
    "body_es": "Revisa tus copias de seguridad.",
    "timer": {
      "id": "trigger",
      "kind": "interval",
      "interval_seconds": 86400,
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "start",
    "en": "Plan your workday",
    "es": "Planifica tu jornada",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "Plan your workday.",
    "body_es": "Planifica tu jornada.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "09:00",
      "timezone": "UTC",
      "weekdays": [
        "mon",
        "tue",
        "wed",
        "thu",
        "fri"
      ],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "lunch",
    "en": "Lunch break",
    "es": "Pausa para almorzar",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "Lunch break.",
    "body_es": "Pausa para almorzar.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "12:30",
      "timezone": "UTC",
      "weekdays": [],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "finish",
    "en": "End of workday",
    "es": "Fin de jornada",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "End of workday.",
    "body_es": "Fin de jornada.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "18:00",
      "timezone": "UTC",
      "weekdays": [
        "mon",
        "tue",
        "wed",
        "thu",
        "fri"
      ],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "weekly",
    "en": "Weekly review",
    "es": "Revisión semanal",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "Weekly review.",
    "body_es": "Revisión semanal.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "16:00",
      "timezone": "UTC",
      "weekdays": [
        "fri"
      ],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "monday",
    "en": "Weekly planning",
    "es": "Planificación semanal",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "Weekly planning.",
    "body_es": "Planificación semanal.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "09:00",
      "timezone": "UTC",
      "weekdays": [
        "mon"
      ],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "evening",
    "en": "Evening screen break",
    "es": "Descanso de pantallas",
    "category": "Schedules",
    "source": "timer:trigger",
    "body_en": "Evening screen break.",
    "body_es": "Descanso de pantallas.",
    "timer": {
      "id": "trigger",
      "kind": "calendar",
      "at": "21:00",
      "timezone": "UTC",
      "weekdays": [],
      "missed": "skip",
      "enabled": false
    }
  },
  {
    "id": "root-space",
    "en": "Low system disk space",
    "es": "Poco espacio en el sistema",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "disk",
      "path": "/",
      "threshold": 10,
      "recovery": 15,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "home-space",
    "en": "Low home disk space",
    "es": "Poco espacio en home",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "disk",
      "path": "/home",
      "threshold": 10,
      "recovery": 15,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "data-space",
    "en": "Low data disk space",
    "es": "Poco espacio en datos",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "disk",
      "path": "/data",
      "threshold": 10,
      "recovery": 15,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "cpu",
    "en": "Sustained high CPU use",
    "es": "Uso alto sostenido de CPU",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "cpu",
      "path": "",
      "threshold": 90,
      "recovery": 75,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "memory",
    "en": "High memory use",
    "es": "Uso alto de memoria",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "memory",
      "path": "",
      "threshold": 90,
      "recovery": 75,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "battery",
    "en": "Low battery monitor",
    "es": "Monitor de batería baja",
    "category": "Monitoring",
    "source": "monitor:trigger",
    "body_en": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "body_es": "{{data.metric}}: {{data.value}} · {{data.state}}",
    "monitor": {
      "id": "trigger",
      "metric": "battery",
      "path": "",
      "threshold": 20,
      "recovery": 30,
      "duration_seconds": 60,
      "interval_seconds": 15,
      "cooldown_seconds": 300,
      "enabled": false
    }
  },
  {
    "id": "build-success",
    "en": "Build completed",
    "es": "Compilación terminada",
    "category": "Local events",
    "source": "local:build-success",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "build-failed",
    "en": "Build failed",
    "es": "Compilación fallida",
    "category": "Local events",
    "source": "local:build-failed",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "backup-done",
    "en": "Backup completed",
    "es": "Copia de seguridad terminada",
    "category": "Local events",
    "source": "local:backup-done",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "download-done",
    "en": "Download completed",
    "es": "Descarga terminada",
    "category": "Local events",
    "source": "local:download-done",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "task-done",
    "en": "Task completed",
    "es": "Tarea terminada",
    "category": "Local events",
    "source": "local:task-done",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "meeting",
    "en": "Meeting reminder",
    "es": "Recordatorio de reunión",
    "category": "Local events",
    "source": "local:meeting",
    "body_en": "{{data.message}}",
    "body_es": "{{data.message}}"
  },
  {
    "id": "welcome-checklist",
    "en": "Startup checklist",
    "es": "Lista de inicio",
    "category": "Desktop",
    "source": "hook:post-boot",
    "body_en": "Review your priorities for today.",
    "body_es": "Revisa tus prioridades de hoy.",
    "second_en": "Check your calendar before starting.",
    "second_es": "Revisa tu calendario antes de empezar."
  },
  {
    "id": "update-checklist",
    "en": "Post-update checklist",
    "es": "Lista después de actualizar",
    "category": "Desktop",
    "source": "hook:post-update",
    "body_en": "The update hook has arrived.",
    "body_es": "Se recibió el evento de actualización.",
    "second_en": "Review the update output for any required restart.",
    "second_es": "Revisa la salida de la actualización por si requiere reiniciar."
  }
];

function text(recipe, key, language) { return recipe[key + (language === "es" ? "_es" : "_en")]; }
function title(recipe, language) { return language === "es" ? recipe.es : recipe.en; }
function instantiate(recipe, config, language) {
    var next = JSON.parse(JSON.stringify(config));
    ["actions", "flows", "monitors", "timers"].forEach(function(k) { next[k] = next[k] || []; });
    var serial = 1;
    var prefix;
    do { prefix = "recipe-" + recipe.id + "-" + serial++; }
    while (["actions", "flows", "monitors", "timers"].some(function(k) { return next[k].some(function(x) { return x.id.indexOf(prefix + ".") === 0; }); }));
    var action = {id:prefix + ".notice", kind:"notify", title:title(recipe,language), body:text(recipe,"body",language)};
    next.actions.push(action);
    var steps = [action.id];
    if (recipe.second_en) {
        next.actions.push({id:prefix + ".second",kind:"notify",title:title(recipe,language),body:text(recipe,"second",language)});
        steps.push(prefix + ".second");
    }
    var source = recipe.source;
    ["timer", "monitor"].forEach(function(kind) {
        if (!recipe[kind]) return;
        var resource = JSON.parse(JSON.stringify(recipe[kind]));
        resource.id = prefix + ".trigger";
        resource.enabled = false;
        next[kind + "s"].push(resource);
        source = kind + ":" + resource.id;
    });
    next.flows.push({id:prefix + ".flow",name:title(recipe,language),source:source,conditions:JSON.parse(JSON.stringify(recipe.conditions || [])),steps:steps,enabled:false});
    return next;
}
