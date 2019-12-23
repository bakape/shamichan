#!/usr/bin/env node
// For easily migrating between version of language packs

"use strict"

const fs = require("fs");

const root = "lang";
const en = readJSON(`${root}/en_GB.json`);
const targets = fs.readdirSync(root).filter(n =>
    n !== "en_GB.json" && /^\w{2}_\w{2}.json$/.test(n));

sortMaps(en);
const path = `${root}/en_GB.json`;
fs.unlinkSync(path)
fs.writeFileSync(path, JSON.stringify(en, null, "\t"));

for (let t of targets) {
    const source = {
        arrays: {},
        strings: {},
    };
    const path = `${root}/${t}`;

    traverse(source, "_lang", readJSON(path));
    fs.unlinkSync(path);

    const dest = JSON.parse(JSON.stringify(en)) // Deep clone
    traverseCopy(source, dest);
    fs.writeFileSync(path, JSON.stringify(dest, null, "\t"));
}

function readJSON(path) {
    return JSON.parse(fs.readFileSync(path, "utf8"));
}

function traverse(map, key, val) {
    if (isMap(val)) {
        for (let key in val) {
            traverse(map, key, val[key])
        }
    } else {
        const dict = map[Array.isArray(val) ? "arrays" : "strings"];
        dict[key] = val;
    }
}

function traverseCopy(src, dest) {
    for (let key in dest) {
        const val = dest[key];
        if (isMap(val)) {
            traverseCopy(src, val);
            continue;
        }

        const dict = src[Array.isArray(val) ? "arrays" : "strings"];
        if (key in dict) {
            dest[key] = dict[key];
        }
    }
}

function isMap(val) {
    return typeof val === "object" && !Array.isArray(val);
}

// Resort objects for cleaner language packs
function sortMaps(val) {
    if (!isMap(val)) {
        return;
    }
    const keys = Object.keys(val).sort();
    const copy = JSON.parse(JSON.stringify(val));
    for (let key of keys) {
        delete val[key];
    }
    for (let key of keys) {
        val[key] = copy[key];
    }
    for (let key in val) {
        sortMaps(val[key]);
    }
}
