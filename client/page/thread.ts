import { ThreadData } from "../common"
import {
    extractConfigs, isBanned, extractPost, localizeThreads, reparseOpenPosts,
    extractPageData,
} from "./common"
import { findSyncwatches } from "../posts"
import { config } from "../state"

const counters = document.getElementById("thread-post-counters"),
    threads = document.getElementById("threads")
let postCtr = 0,
    imgCtr = 0,
    bumpTime = 0

// Render the HTML of a thread page
export default function () {
    if (isBanned()) {
        return
    }
    extractConfigs()

    const { threads: data, backlinks } = extractPageData<ThreadData>(),
        { posts } = data

    data.posts = null
    setPostCount(data.postCtr, data.imageCtr, data.bumpTime)

    extractPost(data, data.id, data.board, backlinks)
    if (data.image) {
        data.image.large = true
    }

    for (let post of posts) {
        extractPost(post, data.id, data.board, backlinks)
    }
    localizeThreads()
    reparseOpenPosts()
    findSyncwatches(threads)
}

// Increment thread post counters and rerender the indicator in the banner
export function incrementPostCount(post: boolean, hasImage: boolean) {
    if (post) {
        postCtr++
        bumpTime = Math.floor(Date.now() / 1000) // An estimate, but good enough
    }
    if (hasImage) {
        imgCtr++
    }
    renderPostCounter()
}

// Externally set thread image post count
export function setPostCount(posts: number, images: number, bump: number) {
    postCtr = posts
    imgCtr = images
    bumpTime = bump
    renderPostCounter()
}

function renderPostCounter() {
    let text = ""
    if (postCtr) {
        text = `${postCtr} / ${imgCtr}`

        // Calculate estimated thread expiry time
        if (config.pruneThreads) {
            // Calculate expiry age
            const min = config.threadExpiryMin,
                max = config.threadExpiryMax
            let days = min + (-max + min) * (postCtr / 3000 - 1) ** 3
            if (days < min) {
                days = min
            }

            // Subtract current bump time
            days -= (Date.now() / 1000 - bumpTime) / (3600 * 24)

            text += ` / `
            if (days > 1) {
                text += `${Math.round(days)}d`
            } else {
                text += `${Math.round(days / 24)}h`
            }
        }
    }
    counters.textContent = text
}
