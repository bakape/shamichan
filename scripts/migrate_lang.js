#!/usr/bin/env node
// For easily migrating between version of language packs

"use strict"

const fs = require("fs")

const en = {
    server: readJSON("lang/en_GB/server.json", "utf8"),
    common: readJSON("lang/en_GB/common.json", "utf8"),
}
const targets = fs.readdirSync("lang").filter(n =>
    n !== "en_GB" && /^\w{2}_\w{2}$/.test(n))

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
    if (typeof val === "object" && !Array.isArray(val)) {
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
        if (typeof val === "object" && !Array.isArray(val)) {
            traverseCopy(src, val)
            continue
        }

        const dict = src[Array.isArray(val) ? "arrays" : "strings"]
        if (key in dict) {
            dest[key] = dict[key]
        }
    }
}
