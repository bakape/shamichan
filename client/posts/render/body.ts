import { config, boards, boardConfig, posts } from '../../state'
import { renderPostLink, renderTempLink } from './etc'
import { PostData, PostLink, TextState, commandType } from '../../common'
import { escape, makeAttrs } from '../../util'
import { parseEmbeds } from "../embed"
import highlightSyntax from "./code"

// URLs supported for linkification
const urlPrefixes = {
    'h': "http",
    'm': "magnet:?",
    'f': "ftp",
    'b': "bitcoin",
}

// Render the text body of a post
export default function renderBody(data: PostData): string {
    const state: TextState = data.state = {
        spoiler: false,
        quote: false,
        code: false,
        bold: false,
        italic: false,
        red: false,
        blue: false,
        haveSyncwatch: false,
        successive_newlines: 0,
        iDice: 0,
    }
    let html = ""

    const fn = data.editing ? parseOpenLine : parseTerminatedLine
    for (let l of data.body.split("\n")) {
        state.quote = false

        // Prevent successive empty lines
        if (html && state.successive_newlines < 2) {
            html += "<br>"
        }
        if (!l.length) {
            state.successive_newlines++
            continue
        }

        state.successive_newlines = 0
        if (l[0] === ">") {
            state.quote = true
            html += "<em>"
        }
        if (state.spoiler) {
            html += "<del>"
        }
        if (state.bold) {
            html += "<b>"
        }
        if (state.italic) {
            html += "<i>"
        }
        if (state.red) {
            html += "<span class=\"red\">"
        }
        if (state.blue) {
            html += "<span class=\"blue\">"
        }

        html += fn(l, data)

        // Close any unclosed tags
        if (state.blue) {
            html += "</span>"
        }
        if (state.red) {
            html += "</span>"
        }
        if (state.italic) {
            html += "</i>"
        }
        if (state.bold) {
            html += "</b>"
        }
        if (state.spoiler) {
            html += "</del>"
        }
        if (state.quote) {
            html += "</em>"
        }
    }

    return html
}

// Open and close any tags up to level, if they are set.
// Increment level by 1 for each tag deeper you go.
function wrapTags(level: number, state: TextState): string {
    const states = [
        state.spoiler,
        state.bold,
        state.italic,
        state.red,
        state.blue,
    ]
    const opening = [
        "<del>",
        "<b>",
        "<i>",
        "<span class=\"red\">",
        "<span class=\"blue\">",
    ]
    const closing = [
        "</del>",
        "</b>",
        "</i>",
        "</span>",
        "</span>",
    ]

    let html = ""
    for (let i = states.length - 1; i >= level; i--) {
        if (states[i]) {
            html += closing[i]
        }
    }
    if (!states[level]) {
        html += opening[level]
    }
    for (let i = level + 1; i < states.length; i++) {
        if (states[i]) {
            html += opening[i]
        }
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

// Inject spoiler tags and call fn on the remaining parts
function parseSpoilers(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    const _fn = (frag: string) =>
        parseBolds(frag, state, fn)
    let html = ""
    while (true) {
        const i = frag.indexOf("**")
        if (i !== -1) {
            html += _fn(frag.slice(0, i)) + wrapTags(0, state)
            state.spoiler = !state.spoiler
            frag = frag.substring(i + 2)
        } else {
            html += _fn(frag)
            break
        }
    }
    return html
}

// Inject bold tags and call fn on the remaining parts
function parseBolds(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    const _fn = (frag: string) =>
        parseItalics(frag, state, fn)
    let html = ""
    while (true) {
        const i = frag.indexOf("@@")
        if (i !== -1) {
            html += _fn(frag.slice(0, i)) + wrapTags(1, state)
            state.bold = !state.bold
            frag = frag.substring(i + 2)
        } else {
            html += _fn(frag)
            break
        }
    }
    return html
}

// Inject italic tags and call fn on the remaining parts
function parseItalics(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    const _fn = (frag: string) => parseReds(frag, state, fn)
    let html = ""
    while (true) {
        const i = frag.indexOf("~~")
        if (i !== -1) {
            html += _fn(frag.slice(0, i)) + wrapTags(2, state)
            state.italic = !state.italic
            frag = frag.substring(i + 2)
        } else {
            html += _fn(frag)
            break
        }
    }
    return html
}

// Inject red color tags and call fn on the remaining parts
function parseReds(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    const _fn = (frag: string) =>
        parseBlues(frag, state, fn)
    const _rbText = boardConfig.rbText ? () => {
        const wrapped = wrapTags(3, state)
        state.red = !state.red
        return wrapped
    } : () => ""
    let html = ""

    while (true) {
        const i = frag.indexOf("^r")
        if (i !== -1) {
            html += _fn(frag.slice(0, i)) + _rbText()
            frag = frag.substring(i + 2)
        } else {
            html += _fn(frag)
            break
        }
    }

    return html
}

// Inject blue color tags and call fn on the remaining parts
function parseBlues(
    frag: string,
    state: TextState,
    fn: (frag: string) => string,
): string {
    const _rbText = boardConfig.rbText ? () => {
        const wrapped = wrapTags(4, state)
        state.blue = !state.blue
        return wrapped
    } : () => ""
    let html = ""

    while (true) {
        const i = frag.indexOf("^b")
        if (i !== -1) {
            html += fn(frag.slice(0, i)) + _rbText()
            frag = frag.substring(i + 2)
        } else {
            html += fn(frag)
            break
        }
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
        let [leadPunct, word, trailPunct] = splitPunctuation(words[i])
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
                }
                break
            case "#": // Hash commands
                if (data.state.quote) {
                    break
                }
                m = word.match(/^#(flip|\d*d\d+|8ball|pyu|pcount|sw(?:\d+:)?\d+:\d+(?:[+-]\d+)?|roulette|rcount)$/)
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
    let data: PostLink
    for (let l of links) {
        if (l.id === id) {
            data = l
            break
        }
    }
    if (!data) {
        return m[0]
    }
    return m[1] + renderPostLink(data)
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
    // Guard against invalid dice rolls
    if (!commands || !commands[state.iDice]) {
        return "#" + bit
    }

    // Protect from index shifts on boardConfig.pyu toggle
    if (!boardConfig.pyu) {
        switch (commands[state.iDice].type) {
            case commandType.pyu:
            case commandType.pcount:
                state.iDice++
                return "#" + bit
        }
    }

    let formatting = "<strong>"
    let inner: string
    let indirect = false;
    switch (bit) {
        case "flip":
            inner = commands[state.iDice++].val ? "flap" : "flop"
            break
        case "8ball":
            inner = escape(commands[state.iDice++].val.toString())
            break
        case "pyu":
        case "pcount":
            // Protect from index shifts on boardConfig.pyu toggle
            if (!boardConfig.pyu) {
                break
            }
        case "rcount":
            switch (commands[state.iDice].type) {
                case commandType.pyu:
                case commandType.pcount:
                case commandType.rcount:
                    inner = commands[state.iDice++].val.toString()
            }

            break
        case "roulette":
            let val = commands[state.iDice++].val
            inner = val[0].toString() + "/" + val[1].toString()
            // set formatting if the poster died
            if (val[0] == 1) {
                formatting = "<strong class=\"dead\">"
            }
            break
        default:
            indirect = true;
            if (bit.startsWith("sw")) {
                // Protect from various index shift attacks due to dynamic typing
                if (commands[state.iDice + 1].type !== commandType.syncWatch) {
                    return "#" + bit;
                }
                return formatSyncwatch(bit, commands[state.iDice++].val, state)
            }

            // Validate dice
            if (commands[state.iDice + 1].type !== commandType.dice) {
                return "#" + bit;
            }
            const m = bit.match(/^(\d*)d(\d+)$/)
            if (parseInt(m[1]) > 10 || parseInt(m[2]) > 10000) {
                return "#" + bit
            }
            const sides = parseInt(m[2])

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

            formatting = getRollFormatting(rolls.length, sides, sum)
    }

    // Protect from various index shift attacks due to dynamic typing
    const commandMatchers: { [bit: string]: commandType } = {
        flip: commandType.flip,
        "8ball": commandType.eightBall,
        pyu: commandType.pyu,
        pcount: commandType.pcount,
        rcount: commandType.rcount,
        roulette: commandType.roulette,
    }
    if (indirect && commandMatchers[bit] !== commands[state.iDice - 1].type) {
        return "#" + bit;
    }

    return `${formatting}#${bit} (${inner})</strong>`
}

function getRollFormatting(numberOfDice: number, facesPerDie: number, sum: number): string {
    const maxRoll = numberOfDice * facesPerDie
    // no special formatting for small rolls
    if (maxRoll < 10 || facesPerDie == 1) {
        return "<strong>"
    }

    if (maxRoll == sum) {
        return "<strong class=\"super_roll\">";
    } else if (sum == numberOfDice) {
        return "<strong class=\"kuso_roll\">";
    } else if (sum == 69 || sum == 6969) {
        return "<strong class=\"lewd_roll\">";
    } else if (checkEm(sum)) {
        if (sum < 100) {
            return "<strong class=\"dubs_roll\">";
        } else if (sum < 1000) {
            return "<strong class=\"trips_roll\">";
        } else if (sum < 10000) {
            return "<strong class=\"quads_roll\">";
        } else {// QUINTS!!!
            return "<strong class=\"rainbow_roll\">";
        }
    }
    return "<strong>"
}

// If num is made of the same digit repeating
function checkEm(num: number): boolean {
    if (num < 10) {
        return false
    }
    const digit = num % 10
    while (true) {
        num = Math.floor(num / 10)
        if (num == 0) {
            return true
        }
        if (num % 10 != digit) {
            return false
        }
    }
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
