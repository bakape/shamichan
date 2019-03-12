#!/usr/bin/env node
// For easily migrating between version of language packs

"use strict"

const fs = require("fs")

const en = {
    server: readJSON("lang/languages/en_GB/server.json", "utf8"),
    common: readJSON("lang/languages/en_GB/common.json", "utf8"),
}
const targets = fs.readdirSync("lang").filter(n =>
    n !== "en_GB" && /^\w{2}_\w{2}$/.test(n))

for (let key in en) {
    sortMaps(en[key])
    const path = `lang/languages/en_GB/${key}.json`
    fs.unlinkSync(path)
    fs.writeFileSync(path, JSON.stringify(en[key], null, "\t"))
}

for (let t of targets) {
    const source = {
        arrays: {},
        strings: {},
    }
    const dir = `lang/${t}`

    for (let f of fs.readdirSync(dir)) {
        const path = `${dir}/${f}`
        traverse(source, "_lang", readJSON(path))
        fs.unlinkSync(path)
    }

    const dest = JSON.parse(JSON.stringify(en)) // Deep clone
    traverseCopy(source, dest)
    for (let key in dest) {
        const path = `${dir}/${key}.json`
        fs.writeFileSync(path, JSON.stringify(dest[key], null, "\t"))
    }
}

function readJSON(path) {
    return JSON.parse(fs.readFileSync(path, "utf8"))
}

function traverse(map, key, val) {
    if (isMap(val)) {
        for (let key in val) {
            traverse(map, key, val[key])
        }
    } else {
        const dict = map[Array.isArray(val) ? "arrays" : "strings"]
        dict[key] = val
    }
}

function traverseCopy(src, dest) {
    for (let key in dest) {
        const val = dest[key]
        if (isMap(val)) {
            traverseCopy(src, val)
            continue
        }

        const dict = src[Array.isArray(val) ? "arrays" : "strings"]
        if (key in dict) {
            dest[key] = dict[key]
        }
    }
}

function isMap(val) {
    return typeof val === "object" && !Array.isArray(val)
}

// Resort objects for cleaner language packs
function sortMaps(val) {
    if (!isMap(val)) {
        return
    }
    const keys = Object.keys(val).sort()
    const copy = JSON.parse(JSON.stringify(val))
    for (let key of keys) {
        delete val[key]
    }
    for (let key of keys) {
        val[key] = copy[key]
    }
    for (let key in val) {
        sortMaps(val[key])
    }
}
