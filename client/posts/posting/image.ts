
let consecutiveFailures = 0;
const pendingRetries: Set<HTMLImageElement> = new Set()
const TICK_BASE = 2000;
const TICK_MAX = 60 * 1000;

// Reload image if error
// Handle event if image error
function onImageErr(e: Event) {
    const el = (e.target as HTMLImageElement)

    if (el.tagName !== "IMG"
        || (el.complete && el.naturalWidth !== 0)) {
        return
    }

    e.stopPropagation()
    e.preventDefault()

    // there were no pending entries we need to start a new timer chain
    if (pendingRetries.size == 0) {
        setTimeout(() => tick(), TICK_BASE)
    }

    el.dataset.scheduledRetry = "pending"

    if (pendingRetries.has(el)) {
        consecutiveFailures += 1
        // Set maintains insertion-order. We can use that as a queue.
        pendingRetries.delete(el)
        pendingRetries.add(el)
    } else {
        pendingRetries.add(el)
    }
}

function onImageLoad(e: Event) {
    const el = (e.target as HTMLImageElement)

    if (el.tagName !== "IMG") {
        return
    }

    if (pendingRetries.has(el)) {
        consecutiveFailures = consecutiveFailures / 2
        pendingRetries.delete(el)
        delete el.dataset.scheduledRetry
    }
}


function tick() {
    for(let el of pendingRetries) {
        // one reload at a time, to avoid hammering the server
        if (el.dataset.scheduledRetry === "active") {
            break
        }
        if (el.dataset.scheduledRetry === "pending") {
            el.dataset.scheduledRetry = "active"
            el.src = el.src
            break
        }
        // skip detached dom nodes and completed images
        if (!document.contains(el)
            || (el.complete && el.naturalWidth !== 0)) {
            pendingRetries.delete(el)
        }
    }

    if (pendingRetries.size > 0) {
        // slow down refreshing if we're experiencing failures
        let delay = Math.min(TICK_MAX,  TICK_BASE * Math.pow(2, consecutiveFailures))
        setTimeout(() => tick(), delay)
    }
}

// Bind listeners
export default () => {
    document.addEventListener("error", onImageErr, true)
    document.addEventListener("load", onImageLoad, true)
}
