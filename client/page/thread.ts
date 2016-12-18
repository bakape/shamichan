import { PostData, ThreadData, Post, fileTypes } from '../posts/models'
import PostView from '../posts/view'
import { posts as postCollection, hidden, mine, seenReplies } from '../state'
import { threads } from '../render'
import options from "../options"
import lang from "../lang"
import { updateSyncTimestamp } from "../connection"
import notifyAboutReply from "../notification"
import { pluralize, escape } from "../util"
import { setTitle } from "../tab"

// Container for all rendered posts
export let threadContainer: HTMLElement

// Render the HTML of a thread page. Insert specifies if the fragment should be
// inserted into the DOM.
export default function (frag: DocumentFragment) {
    updateSyncTimestamp()

    if (frag) {
        threads.innerHTML = ""
        threads.append(frag)
    }

    threadContainer = threads.querySelector("#thread-container")
    if (!options.workModeToggle && (options.userBG || options.illyaDance)) {
        threadContainer.classList.add("custom-BG")
    }

    const data = JSON.parse(
        threads.querySelector("#post-data").textContent,
    ) as ThreadData
    const {posts} = data
    delete data.posts

    extractPost(data)
    postCollection.lowestID = posts.length ? posts[0].id : data.id
    if (data.image) {
        data.image.large = true
    }

    // Extra client-side localizations. Not done server-side for better
    // cacheability.
    localizeOmitted()
    if (options.anonymise) {
        for (let el of threads.querySelectorAll(".name")) {
            el.textContent = lang.posts["anon"]
        }
    } else if (options.lang !== "en_GB") { // Server renders in en_GB
        // Localize posts without a poster name or tripcode
        for (let el of threads.querySelectorAll(".name")) {
            if (el.textContent === "Anonymous") {
                el.textContent = lang.posts["anon"]
            }
        }
    }

    setThreadTitle(data)

    for (let post of posts) {
        extractPost(post)
    }
}

// Set thread title to tab
export function setThreadTitle(data: ThreadData) {
    setTitle(`/${data.board}/ - ${escape(data.subject)} (#${data.id})`)
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting. writeNow specifies, if the write to the DOM fragment should not
// be delayed.
function extractPost(post: PostData) {
    const el = document.getElementById(`p${post.id}`)

    if (hidden.has(post.id)) {
        return el.remove()
    }

    const model = new Post(post),
        view = new PostView(model, el)
    postCollection.add(model)

    // Apply client-specific formatting to a post rendered server-side

    // Render time-zone correction or relative time. Will do unneeded work,
    // if client is on UTC. Meh.
    view.renderTime()

    // Add (You) to posts linking to the user's posts and trigger desktop
    // notifications, if needed
    const {model: {links, backlinks, image}} = view
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

// Localize omitted post and image span
function localizeOmitted() {
    // Server renders in en_GB
    if (options.lang === "en_GB") {
        return
    }
    const el = document.querySelector(".omit")
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

    el.firstChild.replaceWith(text)
    el.querySelector("a.history").textContent = lang.posts["seeAll"]
}
