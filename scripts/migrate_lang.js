#!/usr/bin/env node

// Synchronize changes between language packs and print a unified view of the result.

"use strict"

const fs = require("fs")

function isObject(value) {
    return typeof value === "object" && value !== null
}

function walk(fn, target, ...rest) {
    for (const key of Object.keys(target)) {
        fn(key, target, ...rest)
        if (isObject(target[key])) {
            walk(fn, target[key], ...rest.map(value => isObject(value) ? value[key] : undefined))
        }
    }
}

function clone(object, replacements) {
    const target = {}
    walk((key, src, dst, replace) => {
        dst[key] = Array.isArray(src[key]) ? []
            : isObject(src[key]) ? {}
            : replace?.[key] ?? src[key]
    }, object, target, replacements)
    return target
}

function stringify(object) {
    const keys = new Set()
    walk(key => keys.add(key), object)
    return JSON.stringify(object, [...keys].sort(), "\t") + "\n"
}

function merge(packs) {
    const target = {}
    for (const [lang, pack] of Object.entries(packs)) {
        walk((key, src, dst) => {
            if (isObject(src[key])) {
                dst[key] ??= Array.isArray(src[key]) ? [] : {}
            } else {
                dst[key] ??= {}
                dst[key][lang] = src[key]
            }
        }, pack, target)
    }
    return target
}

const root = "static/src/lang"

const packs = {}
for (const lang of fs.readdirSync(root)) {
    const pack = {}
    for (const file of fs.readdirSync(`${root}/${lang}`)) {
        pack[file] = JSON.parse(fs.readFileSync(`${root}/${lang}/${file}`, "utf8"))
    }
    packs[lang] = pack
}

for (const [file, base] of Object.entries(packs["en_GB"])) {
    for (const [lang, pack] of Object.entries(packs)) {
        pack[file] = clone(base, pack[file])
        fs.writeFileSync(`${root}/${lang}/${file}`, stringify(pack[file]))
    }
}

process.stdout.write(stringify(merge(packs)))
