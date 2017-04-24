import { config, boards, posts } from '../../state'
import { renderPostLink, renderTempLink } from './etc'
import { PostData, PostLink, TextState } from '../../common'
import { escape, makeAttrs } from '../../util'
import { parseEmbeds } from "../embed"
import highlightSyntax from "./code"

// URLs supported for linkification
const urlPrefixes = {
    'h': "http",
    'm': "magnet:?",
    'i': "irc",
    'f': "ftp",
    'b': "bitcoin",
}

// Render the text body of a post
export default function renderBody(data: PostData): string {
    const state: TextState = data.state = {
        spoiler: false,
        quote: false,
        lastLineEmpty: false,
        code: false,
        haveSyncwatch: false,
        iDice: 0,
    }
    let html = ""

    const fn = data.editing ? parseOpenLine : parseTerminatedLine,
        lines = data.body.split("\n"),
        last = lines.length - 1
    for (let i = 0; i < lines.length; i++) {
        const l = lines[i]

        // Prevent successive empty lines
        if (!l.length) {
            // Don't break, if body ends with newline
            if (!state.lastLineEmpty && i !== last) {
                html += "<br>"
            }
            state.lastLineEmpty = true
            state.quote = false
            continue
        }

        html += initLine(l, state)
            + fn(l, data)
            + terminateTags(state, i != last)
    }

    return html
}


// Parse a single line, that is no longer being edited
function parseTerminatedLine(line: string, data: PostData): string {
    return parseCode(line, data.state, frag =>
        parseFragment(frag, data))
}

// Detect code tags
function parseCode(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    let html = ""
    while (true) {
        const i = frag.indexOf("``")
        if (i !== -1) {
            html += formatCode(frag.slice(0, i), state, fn)
            frag = frag.substring(i + 2)
            state.code = !state.code
        } else {
            html += formatCode(frag, state, fn)
            break
        }
    }
    return html
}

function formatCode(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    let html = ""
    if (state.code) {
        // Strip quotes
        while (frag[0] === '>') {
            html += "&gt;"
            frag = frag.slice(1)
        }
        html += highlightSyntax(frag)
    } else {
        html += parseSpoilers(frag, state, fn)
    }
    return html
}

// Injects spoiler tags and calls fn on the remaining parts
function parseSpoilers(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    let html = ""
    while (true) {
        const i = frag.indexOf("**")
        if (i !== -1) {
            html += fn(frag.slice(0, i))
            if (state.quote) {
                html += "</em>"
            }
            html += `<${state.spoiler ? '/' : ''}del>`
            if (state.quote) {
                html += "<em>"
            }

            state.spoiler = !state.spoiler
            frag = frag.substring(i + 2)
        } else {
            html += fn(frag)
            break
        }
    }
    return html
}

// Open a new line container and check for quotes
function initLine(line: string, state: TextState): string {
    let html = ""
    state.quote = state.lastLineEmpty = false
    if (line[0] === ">") {
        state.quote = true
        html += "<em>"
    }
    if (state.spoiler) {
        html += "<del>"
    }
    return html
}

// Close all open tags at line end
function terminateTags(state: TextState, newLine: boolean): string {
    let html = ""
    if (state.spoiler) {
        html += "</del>"
    }
    if (state.quote) {
        html += "</em>"
    }
    if (newLine) {
        html += "<br>"
    }
    return html
}

// Parse a line that is still being edited
function parseOpenLine(line: string, { state }: PostData): string {
    return parseCode(line, state, parseOpenLinks)
}

// Parse temporary links, that still may be edited
function parseOpenLinks(frag: string): string {
    let html = ""
    const words = frag.split(" ")
    for (let i = 0; i < words.length; i++) {
        if (i !== 0) {
            html += " "
        }

        // Split leading and trailing punctuation, if any
        const [leadPunct, word, trailPunct] = splitPunctuation(words[i])
        if (leadPunct) {
            html += leadPunct
        }

        let matched = false
        if (word && word[0] === ">") {
            const m = word.match(/^>>(>*)(\d+)$/)
            if (m) {
                const id = parseInt(m[2])
                if (posts.has(id)) {
                    html += m[1] + renderTempLink(id)
                    matched = true
                }
            }
        }
        if (!matched) {
            html += escape(word)
        }
        if (trailPunct) {
            html += trailPunct
        }
    }
    return html
}

// Parse a line fragment
function parseFragment(frag: string, data: PostData): string {
    let html = ""
    const words = frag.split(" ")
    for (let i = 0; i < words.length; i++) {
        if (i !== 0) {
            html += " "
        }

        // Split leading and trailing punctuation, if any
        const [leadPunct, word, trailPunct] = splitPunctuation(words[i])
        if (leadPunct) {
            html += leadPunct
        }
        if (!word) {
            if (trailPunct) {
                html += trailPunct
            }
            continue
        }

        let m: RegExpMatchArray,
            matched = false
        switch (word[0]) {
            case ">":
                // Post links
                m = word.match(/^>>(>*)(\d+)$/)
                if (m) {
                    html += parsePostLink(m, data.links)
                    matched = true
                    break
                }

                // Internal and custom reference URLs
                m = word.match(/^>>>(>*)\/(\w+)\/$/)
                if (m) {
                    html += parseReference(m)
                    matched = true
                    break
                }
                break
            case "#": // Hash commands
                m = word.match(/^#(flip|\d*d\d+|8ball|pyu|pcount|sw(?:\d+:)?\d+:\d+(?:[+-]\d+)?)$/)
                if (m) {
                    html += parseCommand(m[1], data)
                    matched = true
                    break
                }
                break
            default:
                // Generic HTTP(S) URLs, magnet links and embeds
                // Checking the first byte is much cheaper than a function call.
                // Do that first, as most cases won't match.
                const pre = urlPrefixes[word[0]]
                if (pre && word.startsWith(pre)) {
                    html += parseURL(word)
                    matched = true
                    break
                }
        }

        if (!matched) {
            html += escape(word)
        }
        if (trailPunct) {
            html += trailPunct
        }
    }

    return html
}

// Verify and render a link to other posts
function parsePostLink(m: string[], links: PostLink[]): string {
    if (!links) {
        return m[0]
    }
    const id = parseInt(m[2])
    let op: number
    for (let l of links) {
        if (l[0] === id) {
            op = l[1]
            break
        }
    }
    if (!op) {
        return m[0]
    }
    return m[1] + renderPostLink(id, op)
}

// Parse internal or customly set reference URL
function parseReference(m: string[]): string {
    let href: string
    if (boards.includes(m[2])) {
        href = `/${m[2]}/`
    } else if (m[2] in config.links) {
        href = config.links[m[2]]
    } else {
        return m[0]
    }
    return m[1] + newTabLink(href, `>>>/${m[2]}/`)
}

// Render and anchor link that opens in a new tab
function newTabLink(href: string, text: string): string {
    const attrs = {
        rel: "noreferrer",
        href: escape(href),
        target: "_blank",
    }
    return `<a ${makeAttrs(attrs)}>${escape(text)}</a>`
}

// Parse generic URLs and embed, if applicable
function parseURL(bit: string): string {
    const embed = parseEmbeds(bit)
    if (embed) {
        return embed
    }

    try {
        new URL(bit) // Will throw, if invalid URL
        if (bit[0] == "m") { // Don't open a new tab for magnet links
            bit = escape(bit)
            return bit.link(bit)
        }
        return newTabLink(bit, bit)
    } catch (e) {
        return escape(bit)
    }
}

// Parse a hash command
function parseCommand(bit: string, { commands, state }: PostData): string {
    // Guard against invalid dice rolls and parsing lines in the post form
    if (!commands || !commands[state.iDice]) {
        return "#" + bit
    }

    let inner: string
    switch (bit) {
        case "flip":
        case "8ball":
        case "pyu":
        case "pcount":
            inner = commands[state.iDice++].val.toString()
            break
        default:
            if (bit.startsWith("sw")) {
                return formatSyncwatch(bit, commands[state.iDice++].val, state)
            }

            // Validate dice
            const m = bit.match(/^(\d*)d(\d+)$/)
            if (parseInt(m[1]) > 10 || parseInt(m[2]) > 100) {
                return "#" + bit
            }

            const rolls = commands[state.iDice++].val as number[]
            inner = ""
            let sum = 0
            for (let i = 0; i < rolls.length; i++) {
                if (i) {
                    inner += " + "
                }
                sum += rolls[i]
                inner += rolls[i]
            }
            if (rolls.length > 1) {
                inner += " = " + sum
            }
    }

    return `<strong>#${bit} (${inner})</strong>`
}

// Format a synchronized time counter
function formatSyncwatch(bit: string, val: number[], state: TextState): string {
    state.haveSyncwatch = true
    const attrs = {
        class: "embed syncwatch",
        "data-hour": val[0].toString(),
        "data-min": val[1].toString(),
        "data-sec": val[2].toString(),
        "data-start": val[3].toString(),
        "data-end": val[4].toString()
    }
    return `<em><strong ${makeAttrs(attrs)}>syncwatch</strong></em>`
}

// Splits off one byte of leading and trailing punctuation, if any, and returns
// the 3 split parts. If there is no edge punctuation, the respective string
// is empty.
function splitPunctuation(word: string): [string, string, string] {
    const re: [string, string, string] = ["", word, ""]
    re[1] = word

    // Split leading
    if (re[1].length < 2) {
        return re
    }
    if (isPunctuation(re[1][0])) {
        re[0] = re[1][0]
        re[1] = re[1].slice(1)
    }

    // Split trailing
    const l = re[1].length
    if (l < 2) {
        return re
    }
    if (isPunctuation(re[1][l - 1])) {
        re[2] = re[1][l - 1]
        re[1] = re[1].slice(0, -1)
    }

    return re
}

// Return if b is a punctuation byte
function isPunctuation(b: string): boolean {
    switch (b) {
        case '!':
        case '"':
        case '\'':
        case '(':
        case ')':
        case ',':
        case '-':
        case '.':
        case '/':
        case ':':
        case ';':
        case '?':
        case '[':
        case ']':
            return true
        default:
            return false
    }
}
