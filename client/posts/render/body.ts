import { config, boards } from '../../state'
import { renderPostLink } from './etc'
import { PostData, PostLinks, TextState } from '../models'
import { escape } from '../../util'
import { parseEmbeds } from "../embed"

// Render the text body of a post
export function renderBody(data: PostData): string {
    if (data.editing) {
        return parseOpenBody(data)
    }
    return parseClosedBody(data)
}

// Parse a text body of a closed post
function parseClosedBody(data: PostData): string {
    data.state = {
        spoiler: false,
        quote: false,
        iDice: 0,
    }
    let html = ""
    for (let line of data.body.split("\n")) {
        html += parseTerminatedLine(line, data)
    }
    data.state = null // Clean up a bit

    return html
}

// Parse a text body, that is still being edited
function parseOpenBody(data: PostData): string {
    const state: TextState = data.state = {
        spoiler: false,
        quote: false,
        iDice: 0,
    }
    let html = ""
    const lines = data.body.split("\n")
    for (let i = 0; i < lines.length - 1; i++) {
        html += parseTerminatedLine(lines[i], data)
    }

    state.line = lines[lines.length - 1]
    html += parseOpenLine(state)

    return html
}

// Parse a single line, that is no longer being edited
export function parseTerminatedLine(line: string, data: PostData): string {
    // For hiding redundant newlines using CSS
    if (!line) {
        return "<br>"
    }

    const {state} = data
    let html = initLine(line, state)

    if (line[0] == "#") {
        const m = line.match(/^#(flip|\d*d\d+|8ball|pyu|pcount)$/)
        if (m) {
            return html
                + parseCommand(m[1], data)
                + terminateTags(state, true)
        }
    }

    return html
        + parseSpoilers(line, data.state, frag =>
            parseFragment(frag, data))
        + terminateTags(state, true)
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
            html += fn(frag.slice(0, i)) + `<${state.spoiler ? '/' : ''}del>`
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
    state.spoiler = state.quote = false

    let html = "<span>"
    if (line[0] === ">") {
        state.quote = true
        html += "<em>"
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
    return html + "</span>"
}

// Parse a line that is still being edited
export function parseOpenLine(state: TextState): string {
    if (!state.line) {
        return "<span></span>"
    }
    return initLine(state.line, state)
        + parseSpoilers(state.line, state, escape)
        + terminateTags(state, false)
}

// Parse a line fragment
function parseFragment(frag: string, data: PostData): string {
    let html = ""
    const words = frag.split(" ")
    for (let i = 0; i < words.length; i++) {
        const word = words[i]
        if (i !== 0) {
            html += " "
        }
        if (!word) {
            continue
        }
        if (word[0] === ">") {
            // Post links
            let m = word.match(/^>>(>*)(\d+)$/)
            if (m) {
                html += parsePostLink(m, data.links)
                continue
            }

            // Internal and custom reference URLs
            m = word.match(/^>>>(>*)\/(\w+)\/$/)
            if (m) {
                html += parseReference(m)
                continue
            }
        } else {
            // Generic HTTP(S) URLs, magnet links and embeds
            let match: boolean
            // Checking the first byte is much cheaper than a function call. Do
            // that first, as most cases won't match.
            switch (word[0]) {
                case "h":
                    match = word.startsWith("http")
                    break
                case "m":
                    match = word.startsWith("magnet:?")
                    break
            }
            if (match) {
                html += parseURL(word)
                continue
            }
        }
        html += escape(word)
    }

    return html
}

// Verify and render a link to other posts
function parsePostLink(m: string[], links: PostLinks): string {
    if (!links) {
        return m[0]
    }
    const id = parseInt(m[2]),
        verified = links[id]
    if (!verified) {
        return m[0]
    }
    return m[1] + renderPostLink(id, verified.board, verified.op)
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
    return `<a href="${escape(href)}" target="_blank">${escape(text)}</a>`
}

// Parse generic URLs and embed, if applicable
function parseURL(bit: string): string {
    const embed = parseEmbeds(bit)
    if (embed) {
        return embed
    }

    const m = /^(?:magnet:\?|https?:\/\/)[-a-zA-Z0-9@:%_\+\.~#\?&\/=]+$/
        .test(bit)
    if (!m) {
        return escape(bit)
    }
    if (bit[0] == "m") { // Don't open a new tab for magnet links
        bit = escape(bit)
        return bit.link(bit)
    }
    return newTabLink(bit, bit)
}

// Parse a hash command
function parseCommand(bit: string, {commands, state}: PostData): string {
    // Guard against invalid dice rolls and parsing lines in the post form
    if (!commands || !commands[state.iDice]) {
        return "#" + bit
    }

    // TODO: Sycnwatch

    let inner: string
    switch (bit) {
        case "flip":
        case "8ball":
        case "pyu":
        case "pcount":
            inner = commands[state.iDice++].val.toString()
            break
        default:
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
