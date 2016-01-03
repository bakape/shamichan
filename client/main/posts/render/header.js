import {escape} from 'underscore'
import {parseHTML, pad} from '../util'

/**
 * Render the header with various post information
 * @param {Post} data
 * @returns {string}
 */
export function renderHeader(data) {
    const {num, op, subject} = data
    return parseHTML
        `<header>
            <input type="checkbox" class="postCheckbox">
            ${subject ? `<h3>「${escape(data.subject)}」</h3>` : ''}
            ${renderName(data)}
            ${renderTime(data.time)}
            <nav>
                <a href="${this.postURL(num, op)}" class="history">
                    No.
                </a>
                <a href="${this.postURL(num, op)}" class="quote">
                    ${num}
                </a>
            </nav>
        </header>
        <span class="oi control" data-glyph="chevron-bottom"></span>`
}

/**
 * Render the name of a post's poster
 * @param {Post} data
 * @returns {string}
 */
export function renderName(data) {
    let html = '<b class="name'
    const {auth, email} = data
    if (auth) {
        html += ` ${auth === 'admin' ? 'admin' : 'moderator'}`
    }
    html += '">'
    if (email) {
        const attrs = {
            class: 'email',
            href: 'mailto:' + encodeURI(email),
            target: 'blank'
        }
        html += parseHTML `<a ${attrs}>`
    }
    html += resolveName(data)
    if (email) {
        html += '</a>'
    }
    html += '</b>'
    if (data.mnemonic) {
        html += ' ' + renderMnemonic(data.mnemonic)
    }
    return html
}

/**
 * Determine the name and tripcode combination to render
 * @param {Post} data
 * @returns {string}
 */
function resolveName(data) {
    let html = ''
    const {trip, name, auth} = data
    if (name || !trip) {
        if (name) {
            html += escape(name)
        } else {
            html += lang.anon
        }
        if (trip) {
            html += ' '
        }
    }
    if (trip) {
        html += `<code>${escape(trip)}</code>`
    }
    if (auth) {
        html += ` ## ${imports.hotConfig.staff_aliases[auth] || auth}`
    }
    return html
}

/**
 * Renders a poster identification mnemonic
 * @param {string} mnemonic
 * @returns {string}
 */
export function renderMnemonic(mnemonic) {
    return `<b class="mod addr">${mnem}</b>`
}

/**
 * Renders a time element. Can be either absolute or relative.
 * @param {int} time
 * @param {string}
 */
export function renderTime(time) {
    // Format according to client's relative post timestamp setting
    let title, text
    const readable = readableTime(time)
    if (options.get('relativeTime')) {
        title = readable
        text = relativeTime(time, Date.now())
    }
    return parseHTML
        `<time title="${title}">
            ${text || readable}
        </time>`
}

/**
 * Renders classic absolute timestamp
 * @param {int} time
 * @returns {string}
 */
function readableTime(time) {
	let d = new Date(time)
	return pad(d.getDate()) + ' '
		+ lang.year[d.getMonth()] + ' '
		+ d.getFullYear()
		+ `(${lang.week[d.getDay()]})`
		+`${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/**
 * Renders readable elapsed time since post
 * @param {int} then
 * @param {int} now
 * @returns {string}
 */
function relativeTime(then, now) {
    let time = Math.floor((now - then) / 60000),
        isFuture
    if (time < 1) {
        if (time > -5) { // Assume to be client clock imprecission
            return this.lang.just_now
        }
        else {
            isFuture = true
            time = -time
        }
    }

    const divide = [60, 24, 30, 12],
        unit = ['minute', 'hour', 'day', 'month']
    for (let i = 0; i < divide.length; i++) {
        if (time < divide[i]) {
            return lang.ago(time, lang.time[unit[i]], isFuture)
        }
        time = Math.floor(time / divide[i])
    }

    return lang.ago(time, lang.time.year, isFuture)
}
