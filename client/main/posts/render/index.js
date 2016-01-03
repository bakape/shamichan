/**
 * Posts rendering module
 */

import {parseHTML} from '../util'
import {renderHeader} from '../header'

/**
 * Render the OP
 * @param {Post} data - Post model
 * @param {string=} cls - Class to assign to post
 * @returns {string}
 */
export function renderSection(data, cls = '') {
    if (data.locked) {
        cls += ' locked'
    }
    if (data.editing) {
        cls += ' editing'
    }
    data.image.large = true // Larger thumbnails

    return parseHTML
        `<section id="p${data.num}" class="${cls}">
            <div class="background glass">
                ${renderPost(data)}
                <span class="omit"></span>
            </div>
        </section>`
}

/**
 * Render a reply post
 * @param {Post} data
 * @returns {string}
 */
export function renderArticle(data) {
    let cls = 'glass'
    if (data.editing) {
        cls += ' editing'
    }
    return parseHTML
        `<article id="p${data.num}" class="${cls}">
            ${renderPost(data)}
        </article>`
}

function renderPost(data) {
    const {image, mod, body, backlinks, banned} = data

    return parseHTML
        `${renderHeader(data)}
        ${renderImage(image)}
        <div class="container">
            ${renderModInfo(mod)}
            <blockquote>
                ${renderBody(body)}
            </blockquote>
            <small>
                ${backlinks ? renderBacklinks(backlinks) : ''}
            </small>
            ${banned ? renderBanned(): ''}
        </div>`
}
