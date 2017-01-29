// Miscellaneous post component rendering functions

import { page, mine } from '../../state'
import lang from '../../lang'
import { PostLinks } from '../../common'
import { makeFrag } from "../../util"

// Render a link to other posts
export function renderPostLink(num: number, board: string, op: number): string {
    let html = `<a class="history post-link" data-id="${num}" href="`
    const cross = op !== page.thread

    if (cross) {
        html += `/${board}/${op}`
    }
    html += `#p${num}">>>`

    if (cross) {
        html += `>/${board}/`
    }
    html += num

    if (mine.has(num)) { // Post, I made
        html += ' ' + lang.posts["you"]
    }

    html += `</a><a class="hash-link"> #</a>`

    return html
}

// Render links to posts that are linking to the target post
export function renderBacklinks(post: DocumentFragment, links: PostLinks) {
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
    let rendered: string[] = []
    for (let em of Array.from(el.children)) {
        rendered.push((em.firstChild as HTMLElement).dataset["id"])
    }

    let html = ""
    for (let id in links) {
        // Confirm link not already rendered
        if (rendered.includes(id)) {
            continue
        }

        const {board, op} = links[id]
        html += "<em>" + renderPostLink(parseInt(id), board, op) + "</em>"
    }

    el.append(makeFrag(html))
}
