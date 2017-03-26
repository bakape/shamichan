import { ThreadData } from "../common"
import { escape, threads } from '../util'
import { setSyncCounter } from "../connection"
import { setTitle } from "../ui"
import {
    extractConfigs, isBanned, extractPost, localizeThreads, reparseOpenPosts
} from "./common"
import { findSyncwatches } from "../posts"

const counters = document.getElementById("thread-post-counters")
let postCtr = 0,
    imgCtr = 0

// Render the HTML of a thread page
export default function (html: string) {
    if (html) {
        threads.innerHTML = html
    }
    if (isBanned()) {
        return
    }
    extractConfigs()

    const text = document.getElementById("post-data").textContent,
        data = JSON.parse(text) as ThreadData,
        { posts } = data
    setSyncCounter(data.logCtr)
    delete data.posts
    setPostCount(data.postCtr, data.imageCtr)

    extractPost(data, data.id)
    if (data.image) {
        data.image.large = true
    }

    setThreadTitle(data)

    for (let post of posts) {
        extractPost(post, data.id)
    }
    localizeThreads()
    reparseOpenPosts()
    findSyncwatches(threads)
}

// Set thread title to tab
export function setThreadTitle(data: ThreadData) {
    setTitle(`/${data.board}/ - ${escape(data.subject)} (#${data.id})`)
}

// Increment thread post counters and rerender the indicator in the banner
export function incrementPostCount(post: boolean, hasImage: boolean) {
    if (post) {
        postCtr++
    }
    if (hasImage) {
        imgCtr++
    }
    renderPostCounter()
}

// Externally set thread image post count
export function setPostCount(posts: number, images: number) {
    postCtr = posts
    imgCtr = images
    renderPostCounter()
}

function renderPostCounter() {
    let text: string
    if (!postCtr && !imgCtr) {
        text = ""
    } else {
        text = `${postCtr} / ${imgCtr}`
    }
    counters.textContent = text
}
