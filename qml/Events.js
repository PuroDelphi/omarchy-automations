.pragma library

// Stable event values are independent of the display language.
var hooks = [
    {value:"hook:theme-set", label:"Al cambiar el tema", field:"data.theme", fieldLabel:"Nombre del tema", type:"string"},
    {value:"hook:font-set", label:"Al cambiar la fuente", field:"data.font", fieldLabel:"Nombre de la fuente", type:"string"},
    {value:"hook:battery-low", label:"Con batería baja", field:"data.percentage", fieldLabel:"Porcentaje de batería", type:"number"},
    {value:"hook:post-boot", label:"Al iniciar Omarchy"},
    {value:"hook:post-update", label:"Después de actualizar Omarchy"},
    {value:"hook:pre-refresh-pacman", label:"Antes de sincronizar los paquetes"}
];
function fields(source) {
    return hooks.filter(function(h) { return h.value === source && h.field; });
}
function label(source) {
    var match = hooks.find(function(h) { return h.value === source; });
    return match ? match.label : source;
}
