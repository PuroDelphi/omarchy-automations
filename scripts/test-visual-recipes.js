// Portable logic checks; native interaction coverage lives in test-visual-flows.py.
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const context = {};
vm.createContext(context);
vm.runInContext(fs.readFileSync(path.join(root, 'qml/Recipes.js'), 'utf8').replace('.pragma library', ''), context);
let config = {version:1, entries:[], destinations:[], actions:[], flows:[], monitors:[]};
assert(context.recipes.length >= 30);
for (const recipe of context.recipes) {
    assert(recipe.en && recipe.es && recipe.body_en && recipe.body_es);
    const previous = JSON.stringify(config);
    const next = context.instantiate(recipe, config, 'en');
    assert.equal(JSON.stringify(config), previous, 'template mutated the input');
    config = next;
}
// Repeated use must not overwrite IDs; locales must not affect references.
config = context.instantiate(context.recipes[0], config, 'es');
for (const kind of ['actions','flows','monitors','timers']) {
    const ids = config[kind].map(x => x.id);
    assert.equal(new Set(ids).size, ids.length);
    if (kind !== 'actions') assert(config[kind].every(x => x.enabled === false));
}
for (const flow of config.flows) assert(flow.steps.every(id => config.actions.some(a => a.id === id)));
if (process.argv[2]) fs.writeFileSync(process.argv[2], JSON.stringify(config));
console.log(`Validated ${context.recipes.length} bilingual templates, immutability and repeat insertion`);
