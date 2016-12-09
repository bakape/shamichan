import { PostData, ThreadData, Post, fileTypes } from '../posts/models'
import PostView from '../posts/view'
import { posts as postCollection, hidden, mine, seenReplies } from '../state'
import { threads } from '../render'
import options from "../options"
import lang from "../lang"
import { updateSyncTimestamp } from "../connection"
import notifyAboutReply from "../notification"
import { maybeWriteNow } from "./common"
import { pluralize, escape } from "../util"
import { setTitle } from "../tab"

// Container for all rendered posts
export let threadContainer: HTMLElement

// Render the HTML of a thread page. writeNow specifies, if the write to the DOM
// fragment should not be delayed.
export default function (frag: DocumentFragment, writeNow: boolean) {
    updateSyncTimestamp()

    threadContainer = frag.querySelector("#thread-container")
    if (!options.workModeToggle && (options.userBG || options.illyaDance)) {
        maybeWriteNow(writeNow, () =>
            threadContainer.classList.add("custom-BG"))
    }

    const data = JSON.parse(
        frag.querySelector("#post-data").textContent,
    ) as ThreadData
    const {posts} = data
    delete data.posts

    extractPost(data, frag, writeNow)
    postCollection.lowestID = posts.length ? posts[0].id : data.id
    if (data.image) {
        data.image.large = true
    }

    // Extra client-side localizations. Not done server-side for better
    // cacheability.
    localizeOmitted(frag, writeNow)
    if (options.anonymise) {
        maybeWriteNow(writeNow, () => {
            for (let el of frag.querySelectorAll(".name")) {
                el.textContent = lang.posts["anon"]
            }
        })
    } else {
        localizeAnonymous(frag, writeNow)
    }

    setThreadTitle(data)

    for (let post of posts) {
        extractPost(post, frag, writeNow)
    }

    if (writeNow) {
        threads.innerHTML = ""
        threads.append(frag)
    }
}

// Set thread title to tab
export function setThreadTitle(data: ThreadData) {
    setTitle(`/${data.board}/ - ${escape(data.subject)} (#${data.id})`)
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting. writeNow specifies, if the write to the DOM fragment should not
// be delayed.
function extractPost(
    post: PostData,
    frag: NodeSelector,
    writeNow: boolean,
) {
    const el = frag.querySelector(`#p${post.id}`)

    if (hidden.has(post.id)) {
        return maybeWriteNow(writeNow, () =>
            el.remove())
    }

    const model = new Post(post),
        view = new PostView(model, el)
    postCollection.add(model)
    maybeWriteNow(writeNow, () =>
        formatPost(view))
}

// Apply client-specific formatting to a post rendered server-side
function formatPost(view: PostView) {
    // Render time-zone correction or relative time. Will do unneeded work,
    // if client is on UTC. Meh.
    view.renderTime()

    addYou(view)

    const {model: {image}} = view
    if (image) {
        const should =
            options.hideThumbs
            || options.workModeToggle
            || (image.spoiler && !options.spoilers)
            || (image.fileType === fileTypes.gif && options.autogif)
        if (should) {
            view.renderImage(false, false)
        }
    }
}

// Add (You) to posts linking to the user's posts and trigger desktop
// notifications, if needed
function addYou(view: PostView) {
    const {model: {links, backlinks}} = view
    for (let l of [links, backlinks]) {
        if (!l) {
            continue
        }
        for (let idStr in l) {
            const id = parseInt(idStr)
            if (!mine.has(id)) {
                continue
            }
            for (let el of view.el.querySelectorAll(`a[data-id="${id}"]`)) {
                el.textContent += " " + lang.posts["you"]
            }
            if (!seenReplies.has(id)) {
                notifyAboutReply(view.model)
            }
        }
    }
}

// Localize omitted post and image span
function localizeOmitted(frag: DocumentFragment, writeNow: boolean) {
    // Server renders in en_GB
    if (options.lang === "en_GB") {
        return
    }
    const el = frag.querySelector(".omit")
    if (!el) {
        return
    }

    const posts = parseInt(el.getAttribute("data-omit")),
        images = parseInt(el.getAttribute("data-image-omit"))
    let text = pluralize(posts, lang.plurals["post"])
    if (images) {
        text += ` ${lang.posts["and"]} `
            + pluralize(images, lang.plurals["image"])
    }
    text += ` ${lang.posts["omitted"]} `

    maybeWriteNow(writeNow, () => {
        el.firstChild.replaceWith(text)
        el.querySelector("a.history").textContent = lang.posts["seeAll"]
    })
}

// Localize posts without a poster name or tripcode
function localizeAnonymous(frag: DocumentFragment, writeNow: boolean) {
    // Server renders in en_GB
    if (options.lang === "en_GB") {
        return
    }

    const toMod: Element[] = []
    for (let el of frag.querySelectorAll(".name")) {
        if (el.textContent === "Anonymous") {
            toMod.push(el)
        }
    }
    maybeWriteNow(writeNow, () => {
        for (let el of toMod) {
            el.textContent = lang.posts["anon"]
        }
    })
}
