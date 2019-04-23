import { ThreadData, ModerationAction } from "../common"
import {
    extractConfigs, extractPost, reparseOpenPosts, extractPageData, hidePosts,
} from "./common"
import { findSyncwatches } from "../posts"
import { config } from "../state"
import { postSM, postState } from "../posts"

const counters = document.getElementById("thread-post-counters");
const threads = document.getElementById("threads");

const bumpLimit = 1000;

let image_count = 0,
    bump_time = 0,
    isDeleted = false

export let post_count = 0;
export let subject = "";

// Render the HTML of a thread page
export default function () {
    extractConfigs()

    const { threads: data, backlinks } = extractPageData<ThreadData>(),
        { posts } = data

    data.posts = null

    post_count = data.post_count;
    subject = data.subject;
    image_count = data.image_count
    bump_time = data.bump_time
    if (data.moderation) {
        for (let { type } of data.moderation) {
            if (type === ModerationAction.banPost) {
                isDeleted = true;
                break;
            }
        }
    }
    renderPostCounter()

    extractPost(data, data.id, data.board, backlinks)

    for (let post of posts) {
        extractPost(post, data.id, data.board, backlinks)
    }
    hidePosts()
    reparseOpenPosts()
    findSyncwatches(threads)

    // Needs to be done, to  stop the FSM
    if (data.locked) {
        postSM.state = postState.threadLocked
    }
}

// Increment thread post counters and rerender the indicator in the banner
export function incrementPostCount(post: boolean, hasImage: boolean) {
    if (post) {
        post_count++
        if (post_count < bumpLimit) {
            // An estimate, but good enough
            bump_time = Math.floor(Date.now() / 1000)
        }
    }
    if (hasImage) {
        image_count++
    }
    renderPostCounter()
}

function renderPostCounter() {
    let text = ""
    if (post_count) {
        text = `${post_count} / ${image_count}`

        // Calculate estimated thread expiry time
        if (config.pruneThreads) {
            // Calculate expiry age
            const min = config.threadExpiryMin,
                max = config.threadExpiryMax
            let days = min + (-max + min) * (post_count / bumpLimit - 1) ** 3
            if (isDeleted) {
                days /= 3
            }
            if (days < min) {
                days = min
            }

            // Subtract current bump time
            days -= (Date.now() / 1000 - bump_time) / (3600 * 24)

            text += ` / `
            if (days > 1) {
                text += `${Math.round(days)}d`
            } else {
                text += `${Math.round(days * 24)}h`
            }
        }
    }
    counters.textContent = text
}
