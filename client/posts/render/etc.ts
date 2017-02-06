// Miscellaneous post component rendering functions

import { page, mine } from '../../state'
import lang from '../../lang'
import { PostLink } from '../../common'
import { makeFrag } from "../../util"

// Render a link to other posts
export function renderPostLink(id: number, op: number): string {
    const cross = op !== page.thread,
        url = `${cross ? `/cross/${op}` : ""}#p${id}`
    let html = `<a class="history post-link" data-id="${id}" href="${url}">>>${id}`
    if (cross) {
        html += " ➡"
    }
    if (mine.has(id)) { // Post, I made
        html += ' ' + lang.posts["you"]
    }
    html += `</a><a class="hash-link" href="${url}"> #</a>`
    return html
}

// Render links to posts that are linking to the target post
export function renderBacklinks(post: DocumentFragment, links: PostLink[]) {
    if (!links) {
        return
    }

    let el = post.querySelector(".backlinks")
    if (!el) {
        el = document.createElement("span")
        el.classList.add("spaced", "backlinks")
        post.append(el)
    }

    // Get already rendered backlink IDs
    const rendered = new Set<number>()
    for (let em of Array.from(el.children)) {
        const id = (em.firstChild as HTMLElement).getAttribute("data-id")
        rendered.add(parseInt(id))
    }

    let html = ""
    for (let [id, op] of links) {
        // Confirm link not already rendered
        if (rendered.has(id)) {
            continue
        }
        rendered.add(id)
        html += "<em>" + renderPostLink(id, op) + "</em>"
    }

    el.append(makeFrag(html))
}
