.pragma library

function fields(kind, config) {
    var common = [{key:"id", label:"Identificador", hint:"ejemplo: despliegues", required:true}]
    var named = [{key:"name", label:"Nombre"}]
    var enabled = [{key:"enabled", label:"Habilitado al activar la revisión", type:"bool"}]
    var actions = (config.actions || []).map(function(x) { return x.id })
    var destinations = (config.destinations || []).map(function(x) { return x.id })
    if (kind === "adapters") return common.concat([
        {key:"path",label:"Archivo local del adaptador",hint:"Ruta absoluta; no se admiten enlaces simbólicos"},
        {key:"runtime",label:"Intérprete",type:"choice",options:["python3"]},
        {key:"input_limit",label:"Límite de entrada (bytes)",type:"number"},
        {key:"output_limit",label:"Límite de salida (bytes)",type:"number"},
        {key:"timeout_seconds",label:"Tiempo máximo (segundos)",type:"number"}
    ])
    if (kind === "scripts") return common.concat([
        {key:"path",label:"Archivo local del script",hint:"Ruta absoluta; no se admiten enlaces simbólicos"},
        {key:"interpreter",label:"Intérprete",type:"choice",options:["bash","python3"]},
        {key:"parameters",label:"Parámetros del script",type:"script-parameters"}
    ])
    if (kind === "entries") return common.concat(named, [
        {key:"auth",label:"Autenticación",type:"choice",options:["hmac","github","slack","bearer"]},
        {key:"secret",label:"Referencia de credencial",hint:"Créala en Seguridad; no pegues el secreto aquí"},
        {key:"format",label:"Formato del cuerpo",type:"choice",options:["json","form","raw","xml","multipart"]}
    ],enabled)
    if (kind === "destinations") return common.concat(named, [
        {key:"url",label:"Dirección HTTPS",hint:"https://servicio.example/webhook"},
        {key:"method",label:"Método",type:"choice",options:["POST","PUT","PATCH"]},
        {key:"format",label:"Formato de salida",type:"choice",options:["json","form","raw"]},
        {key:"auth",label:"Autenticación",type:"choice",options:["","hmac","bearer","oauth2-google"]},
        {key:"secret",label:"Referencia de credencial",when:"auth",not:""},
        {key:"headers",label:"Cabeceras adicionales",type:"pairs",hint:"Nombre y valor; las credenciales se gestionan por referencia"},
        {key:"private_hosts",label:"Excepciones de red privada",type:"lines",hint:"Solo el host:puerto exacto de este destino (443 si se omite). IPv6 entre corchetes. Vacío bloquea redes internas."}
    ])
    if (kind === "actions") return common.concat([
        {key:"kind",label:"Acción",type:"choice",options:["notify","http","service","command","omarchy","script","adapter","system-service"]},
        {key:"title",label:"Título",when:"kind",is:"notify"},
        {key:"body",label:"Mensaje",type:"multiline",when:"kind",is:"notify",hint:"Puedes usar {{data.message}}"},
        {key:"destination",label:"Destino registrado",type:"choice",options:destinations,when:"kind",is:"http"},
        {key:"body",label:"Cuerpo del envío",type:"multiline",when:"kind",is:"http",hint:'{"message":"{{data.message}}"} — JSON para salida JSON/formulario; texto para salida raw'},
        {key:"unit",label:"Unidad de sistema",hint:"backup.service — nombre exacto autorizado por el administrador",when:"kind",is:"system-service"},
        {key:"operation",label:"Operación administrativa",type:"choice",options:["status","start","stop","restart"],when:"kind",is:"system-service"},
        {key:"unit",label:"Unidad de usuario",hint:"my-app.service",when:"kind",is:"service"},
        {key:"operation",label:"Operación",type:"choice",options:["status","start","stop","restart"],when:"kind",is:"service"},
        {key:"operation",label:"Operación de Omarchy",type:"choice",options:["theme.current","nightlight.status","nightlight.toggle","system.lock"],when:"kind",is:"omarchy"},
        {key:"command_profile",label:"Perfil de comando",type:"choice",options:["fixed","file-exists","make-directory"],when:"kind",is:"command"},
        {key:"command_path",label:"Ruta del perfil",hint:"Un hijo directo del montaje: /work/data/nombre. make-directory requiere escritura y crea con permisos 0700.",when:"kind",is:"command",profiles:["file-exists","make-directory"]},
        {key:"executable",profiles:["fixed"],label:"Ejecutable local",hint:"/usr/bin/…",when:"kind",is:"command"},
        {key:"args",profiles:["fixed"],label:"Argumentos fijos",type:"lines",hint:"Un argumento por línea; sin expansión de shell",when:"kind",is:"command"},
        {key:"adapter",label:"Adaptador registrado",type:"choice",options:(config.adapters || []).map(function(x) { return x.id }),when:"kind",is:"adapter"},
        {key:"script",label:"Script registrado",type:"choice",options:(config.scripts || []).map(function(x) { return x.id }),when:"kind",is:"script"},
        {key:"timeout_seconds",label:"Tiempo máximo (segundos)",type:"number",when:"kind",oneOf:["command","script"]}
    ])
    if (kind === "flows") return common.concat(named,[
        {key:"source",label:"Cuando llegue un evento de",type:"editable-choice",options:["local:demo"].concat((config.entries||[]).map(function(x){return "entry:"+x.id}),(config.monitors||[]).map(function(x){return "monitor:"+x.id}),(config.timers||[]).map(function(x){return "timer:"+x.id})),hint:"Entrada, monitor, programación, local:nombre o hook:evento"},
        {key:"conditions",label:"Si se cumplen estas condiciones",type:"conditions"},
        {key:"steps",label:"Ejecutar en este orden",type:"steps",options:actions}
    ],enabled)
    if (kind === "timers") return common.concat(named,[
        {key:"kind",label:"Programación",type:"choice",options:["interval","calendar"]},
        {key:"interval_seconds",label:"Intervalo en segundos (mínimo 5)",type:"number",when:"kind",is:"interval"},
        {key:"at",label:"Hora local HH:MM",hint:"09:00",when:"kind",is:"calendar"},
        {key:"timezone",label:"Zona horaria IANA",hint:"America/Bogota",when:"kind",is:"calendar"},
        {key:"weekdays",label:"Días de la semana",type:"lines",hint:"Un día por línea: mon tue wed thu fri sat sun. Vacío: todos los días.",when:"kind",is:"calendar"},
        {key:"missed",label:"Si hay ejecuciones atrasadas",type:"choice",options:["coalesce","skip"],hint:"coalesce: una al reanudar; skip: descartar atrasos mayores de 5 s (intervalos) o 60 s (calendario)."}
    ],enabled)
    if (kind === "monitors") return common.concat([
        {key:"metric",label:"Métrica",type:"choice",options:["cpu","memory","disk","battery","service","process","file_exists","file_age","file_size","connectivity","temperature","journal"]},
        {key:"path",label:"Ruta del disco",when:"metric",is:"disk"},
        {key:"unit",label:"Unidad de usuario",when:"metric",oneOf:["service","journal"]},
        {key:"path",label:"Ejecutable del proceso (usuario actual)",hint:"/usr/bin/my-program",when:"metric",is:"process"},
        {key:"path",label:"Archivo autorizado (sin enlaces simbólicos)",when:"metric",oneOf:["file_exists","file_age","file_size"]},
        {key:"path",label:"Sensor de temperatura",hint:"/sys/class/thermal/thermal_zone0/temp",when:"metric",is:"temperature"},
        {key:"destination",label:"Destino HTTPS para comprobar con HEAD",type:"choice",options:destinations,when:"metric",is:"connectivity"},
        {key:"priority",label:"Prioridad máxima de journal (0 emergencia … 7 debug)",type:"number",when:"metric",is:"journal"},
        {key:"threshold",label:"Umbral de alerta",type:"number",when:"metric",not:"journal",hint:"Temperatura: °C; antigüedad: segundos; tamaño: bytes; otras métricas: %. Presencia/conectividad: 0 o 100."},
        {key:"recovery",label:"Umbral de recuperación",type:"number",when:"metric",not:"journal"},
        {key:"duration_seconds",label:"Duración mínima de la condición (segundos)",type:"number",when:"metric",not:"journal"},
        {key:"recovery_duration_seconds",label:"Duración mínima de recuperación (segundos)",type:"number",when:"metric",not:"journal"},
        {key:"interval_seconds",label:"Intervalo de muestreo (segundos)",type:"number"},
        {key:"cooldown_seconds",label:"Separación mínima entre alertas (segundos)",type:"number",when:"metric",not:"journal"}
    ],enabled)
    return common
}
function defaults(kind) {
    if (kind === "adapters") return {id:"",path:"",protocol:1,runtime:"python3",capabilities:["event.read","data.write"],input_limit:262144,output_limit:4096,timeout_seconds:5}
    if (kind === "scripts") return {id:"",path:"",interpreter:"bash",parameters:[]}
    if (kind === "entries") return {id:"",name:"",auth:"hmac",secret:"",format:"json",enabled:true}
    if (kind === "destinations") return {id:"",name:"",url:"",method:"POST",format:"json",auth:"",headers:{},private_hosts:[]}
    if (kind === "actions") return {id:"",kind:"notify",title:"Omarchy Automations",body:"{{data.message}}",operation:"status",args:[],timeout_seconds:30}
    if (kind === "timers") return {id:"",name:"",kind:"interval",interval_seconds:60,at:"09:00",timezone:"America/Bogota",weekdays:[],missed:"coalesce",enabled:true}
    if (kind === "flows") return {id:"",name:"",source:"local:demo",conditions:[],steps:[],enabled:true}
    return {id:"",metric:"disk",path:"/",threshold:10,recovery:15,duration_seconds:60,recovery_duration_seconds:0,interval_seconds:10,cooldown_seconds:300,priority:3,enabled:true}
}
function visible(field, draft) { if (field.profiles && field.profiles.indexOf(draft.command_profile || "fixed") < 0) return false; return !field.when || (field.oneOf ? field.oneOf.indexOf(draft[field.when]) >= 0 : field.is !== undefined ? draft[field.when] === field.is : draft[field.when] !== field.not) }
